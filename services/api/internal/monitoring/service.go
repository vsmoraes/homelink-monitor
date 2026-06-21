package monitoring

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"homelink-monitor/services/api/internal/domain"
	"homelink-monitor/services/api/internal/events"
	"homelink-monitor/services/api/internal/speedtest"
	"homelink-monitor/services/api/internal/store"
)

type Service struct {
	store       *store.Store
	log         *slog.Logger
	speedRunner speedtest.Runner
	speedMu     sync.Mutex
	speedActive bool
	failMu      sync.Mutex
	failCount   int
	outbox      *events.Outbox
}

const speedSchedulePollInterval = 30 * time.Second

func NewService(st *store.Store, log *slog.Logger) *Service {
	return &Service{store: st, log: log, speedRunner: speedtest.Runner{}}
}

func NewServiceWithOutbox(st *store.Store, log *slog.Logger, outbox *events.Outbox) *Service {
	service := NewService(st, log)
	service.outbox = outbox
	return service
}

func (s *Service) SpeedRunning() bool {
	s.speedMu.Lock()
	defer s.speedMu.Unlock()
	return s.speedActive
}

func (s *Service) TriggerSpeedTest(_ context.Context) bool {
	s.speedMu.Lock()
	if s.speedActive {
		s.speedMu.Unlock()
		return false
	}
	s.speedActive = true
	s.speedMu.Unlock()
	go func() {
		defer func() {
			s.speedMu.Lock()
			s.speedActive = false
			s.speedMu.Unlock()
		}()
		jobCtx := context.Background()
		settings, err := s.store.Settings(jobCtx)
		if err != nil {
			s.log.Error("load settings for speed test", "error", err)
			return
		}
		runCtx, cancel := context.WithTimeout(jobCtx, 10*time.Minute)
		defer cancel()
		result := s.speedRunner.Run(runCtx, settings.SpeedTestCommand)
		if !result.Success && result.Error != "" {
			result.Error = speedtest.CommandHelp(settings.SpeedTestCommand) + ": " + result.Error
		}
		if _, err := s.store.InsertSpeedTest(context.Background(), result); err != nil {
			s.log.Error("store speed test", "error", err)
		}
		s.emitSpeedTestEvent(context.Background(), result)
	}()
	return true
}

func (s *Service) Start(ctx context.Context) {
	go s.loop(ctx, "latency", time.Second, func(ctx context.Context) time.Duration {
		settings, err := s.store.Settings(ctx)
		if err != nil || !settings.MonitoringEnabled {
			return time.Minute
		}
		s.checkLatency(ctx, settings)
		return time.Duration(settings.LatencyIntervalSeconds) * time.Second
	})
	go s.loop(ctx, "dns", 2*time.Second, func(ctx context.Context) time.Duration {
		settings, err := s.store.Settings(ctx)
		if err != nil || !settings.MonitoringEnabled {
			return time.Minute
		}
		s.checkDNS(ctx, settings)
		return time.Duration(settings.DNSIntervalSeconds) * time.Second
	})
	go s.loop(ctx, "speed", 5*time.Second, func(ctx context.Context) time.Duration {
		settings, err := s.store.Settings(ctx)
		if err != nil || !settings.MonitoringEnabled || settings.SpeedTestScheduleMinutes <= 0 {
			return speedSchedulePollInterval
		}
		latest, err := s.store.LatestSpeedTest(ctx)
		if err != nil {
			s.log.Error("load latest speed test for schedule", "error", err)
			return speedSchedulePollInterval
		}
		if SpeedTestDue(time.Now().UTC(), latest, settings.SpeedTestScheduleMinutes) {
			s.TriggerSpeedTest(ctx)
		}
		return speedSchedulePollInterval
	})
}

func SpeedTestDue(now time.Time, latest *domain.SpeedTest, scheduleMinutes int) bool {
	if scheduleMinutes <= 0 {
		return false
	}
	if latest == nil {
		return true
	}
	return !latest.StartedAt.Add(time.Duration(scheduleMinutes) * time.Minute).After(now)
}

func (s *Service) loop(ctx context.Context, name string, initialDelay time.Duration, fn func(context.Context) time.Duration) {
	timer := time.NewTimer(initialDelay)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			next := fn(ctx)
			if next <= 0 {
				next = time.Minute
			}
			timer.Reset(next)
		}
	}
}

func (s *Service) checkLatency(ctx context.Context, settings domain.Settings) {
	targets := append([]string{}, settings.LatencyTargets...)
	if settings.RouterIP != "" {
		targets = append(targets, settings.RouterIP)
	}
	if len(targets) == 0 {
		return
	}
	allFailed := true
	anySuccess := false
	for _, target := range targets {
		result := tcpLatency(ctx, target)
		if result.Success {
			allFailed = false
			anySuccess = true
		}
		_, err := s.store.InsertLatency(ctx, result)
		if err != nil {
			s.log.Error("store latency check", "error", err)
		}
		if !result.Success {
			s.emit(ctx, domain.NotificationEvent{
				ID:        fmt.Sprintf("latency-%s-%d", result.Target, result.CheckedAt.UnixNano()),
				Severity:  "warning",
				Metric:    "latency",
				Title:     "Latency check failed",
				Message:   fmt.Sprintf("Latency check to %s failed: %s", result.Target, result.Error),
				Timestamp: result.CheckedAt,
			})
		}
	}
	s.updateOutage(ctx, settings, allFailed, anySuccess)
}

func (s *Service) checkDNS(ctx context.Context, settings domain.Settings) {
	resolver := net.DefaultResolver
	for _, name := range settings.DNSDomains {
		start := time.Now()
		_, err := resolver.LookupHost(ctx, name)
		duration := float64(time.Since(start).Microseconds()) / 1000
		result := domainCheck(name, duration, err)
		if _, err := s.store.InsertDNS(ctx, result); err != nil {
			s.log.Error("store dns check", "error", err)
		}
		if !result.Success {
			s.emit(ctx, domain.NotificationEvent{
				ID:        fmt.Sprintf("dns-%s-%d", result.Domain, result.CheckedAt.UnixNano()),
				Severity:  "warning",
				Metric:    "dns",
				Title:     "DNS check failed",
				Message:   fmt.Sprintf("DNS lookup for %s failed: %s", result.Domain, result.Error),
				Timestamp: result.CheckedAt,
			})
		}
	}
}

func (s *Service) updateOutage(ctx context.Context, settings domain.Settings, allFailed, anySuccess bool) {
	active, err := s.store.ActiveOutage(ctx)
	if err != nil {
		s.log.Error("load active outage", "error", err)
		return
	}
	s.failMu.Lock()
	if allFailed {
		s.failCount++
	} else {
		s.failCount = 0
	}
	failCount := s.failCount
	s.failMu.Unlock()
	if ShouldOpenOutage(failCount, settings.OutageFailureThreshold, active != nil) {
		now := time.Now().UTC()
		if err := s.store.OpenOutage(ctx, now, "all latency targets failed"); err != nil {
			s.log.Error("open outage", "error", err)
		} else {
			s.emit(ctx, domain.NotificationEvent{
				ID:        fmt.Sprintf("outage-open-%d", now.UnixNano()),
				Severity:  "critical",
				Metric:    "outage",
				Title:     "Connection outage detected",
				Message:   "All configured latency targets failed.",
				Timestamp: now,
			})
		}
	}
	if ShouldCloseOutage(anySuccess, active != nil) {
		now := time.Now().UTC()
		if err := s.store.CloseActiveOutage(ctx, now); err != nil {
			s.log.Error("close outage", "error", err)
		} else {
			s.emit(ctx, domain.NotificationEvent{
				ID:        fmt.Sprintf("outage-close-%d", now.UnixNano()),
				Severity:  "recovery",
				Metric:    "outage",
				Title:     "Connection recovered",
				Message:   "At least one configured latency target is reachable again.",
				Timestamp: now,
			})
		}
	}
}

func (s *Service) emitSpeedTestEvent(ctx context.Context, result domain.SpeedTest) {
	if result.Success {
		message := "Speed test completed successfully."
		if result.DownloadMbps != nil && result.UploadMbps != nil {
			message = fmt.Sprintf("Speed test completed: %.1f Mbps down, %.1f Mbps up.", *result.DownloadMbps, *result.UploadMbps)
		}
		s.emit(ctx, domain.NotificationEvent{
			ID:        fmt.Sprintf("speedtest-success-%d", result.StartedAt.UnixNano()),
			Severity:  "info",
			Metric:    "speedtest",
			Title:     "Speed test completed",
			Message:   message,
			Timestamp: time.Now().UTC(),
		})
		return
	}
	s.emit(ctx, domain.NotificationEvent{
		ID:        fmt.Sprintf("speedtest-failed-%d", result.StartedAt.UnixNano()),
		Severity:  "warning",
		Metric:    "speedtest",
		Title:     "Speed test failed",
		Message:   result.Error,
		Timestamp: time.Now().UTC(),
	})
}

func (s *Service) emit(ctx context.Context, event domain.NotificationEvent) {
	if s.outbox == nil || !s.outbox.Enabled() {
		return
	}
	if err := s.outbox.Write(ctx, event); err != nil {
		s.log.Warn("write notification event", "error", err, "metric", event.Metric)
	}
}

func tcpLatency(ctx context.Context, target string) domain.LatencyCheck {
	start := time.Now()
	dialer := net.Dialer{Timeout: 3 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", target)
	checked := time.Now().UTC()
	if err != nil {
		return domain.LatencyCheck{CheckedAt: checked, Target: target, Success: false, Error: err.Error()}
	}
	_ = conn.Close()
	latency := float64(time.Since(start).Microseconds()) / 1000
	return domain.LatencyCheck{CheckedAt: checked, Target: target, LatencyMs: &latency, Success: true}
}

func domainCheck(name string, duration float64, err error) domain.DNSCheck {
	item := domain.DNSCheck{CheckedAt: time.Now().UTC(), Domain: name, Resolver: "system"}
	if err != nil {
		item.Success = false
		item.Error = err.Error()
		return item
	}
	item.Success = true
	item.DurationMs = &duration
	return item
}
