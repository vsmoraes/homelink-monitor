package monitoring

import (
	"context"
	"log/slog"
	"net"
	"sync"
	"time"

	"homelink-monitor/services/api/internal/domain"
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
}

func NewService(st *store.Store, log *slog.Logger) *Service {
	return &Service{store: st, log: log, speedRunner: speedtest.Runner{}}
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
		if err == nil && settings.MonitoringEnabled && settings.SpeedTestScheduleMinutes > 0 {
			s.TriggerSpeedTest(ctx)
			return time.Duration(settings.SpeedTestScheduleMinutes) * time.Minute
		}
		return time.Hour
	})
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
	}
	s.updateOutage(ctx, settings, allFailed, anySuccess)
}

func (s *Service) checkDNS(ctx context.Context, settings domain.Settings) {
	resolver := net.DefaultResolver
	for _, domain := range settings.DNSDomains {
		start := time.Now()
		_, err := resolver.LookupHost(ctx, domain)
		duration := float64(time.Since(start).Microseconds()) / 1000
		result := domainCheck(domain, duration, err)
		if _, err := s.store.InsertDNS(ctx, result); err != nil {
			s.log.Error("store dns check", "error", err)
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
		if err := s.store.OpenOutage(ctx, time.Now().UTC(), "all latency targets failed"); err != nil {
			s.log.Error("open outage", "error", err)
		}
	}
	if ShouldCloseOutage(anySuccess, active != nil) {
		if err := s.store.CloseActiveOutage(ctx, time.Now().UTC()); err != nil {
			s.log.Error("close outage", "error", err)
		}
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
