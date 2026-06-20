package monitoring

import (
	"context"
	"time"

	"homelink-monitor/services/api/internal/domain"
	"homelink-monitor/services/api/internal/store"
)

func BuildSummary(ctx context.Context, st *store.Store, speedRunning bool, now time.Time) (domain.Summary, error) {
	settings, err := st.Settings(ctx)
	if err != nil {
		return domain.Summary{}, err
	}
	since := now.UTC().Add(-24 * time.Hour)
	latestSpeed, err := st.LatestSpeedTest(ctx)
	if err != nil {
		return domain.Summary{}, err
	}
	latestLatency, err := st.LatestLatency(ctx)
	if err != nil {
		return domain.Summary{}, err
	}
	latestDNS, err := st.LatestDNS(ctx)
	if err != nil {
		return domain.Summary{}, err
	}
	latency24h, err := st.LatencySummary(ctx, &since, nil)
	if err != nil {
		return domain.Summary{}, err
	}
	active, err := st.ActiveOutage(ctx)
	if err != nil {
		return domain.Summary{}, err
	}
	outageCount, err := st.OutageCountSince(ctx, since)
	if err != nil {
		return domain.Summary{}, err
	}
	minDown, maxDown, minUp, maxUp, err := st.SpeedBoundsSince(ctx, since)
	if err != nil {
		return domain.Summary{}, err
	}
	status := ConnectionStatus(settings, latestSpeed, latestLatency, latestDNS, active)
	return domain.Summary{
		Status: status, LatestSpeedTest: latestSpeed, LatestLatency: latestLatency, LatestDNSCheck: latestDNS,
		Latency24h: latency24h, OutageCount24h: outageCount, ActiveOutage: active,
		MinDownload24hMbps: minDown, MaxDownload24hMbps: maxDown, MinUpload24hMbps: minUp, MaxUpload24hMbps: maxUp,
		Settings: settings, SpeedTestIsRunning: speedRunning,
	}, nil
}

func ConnectionStatus(settings domain.Settings, speed *domain.SpeedTest, latency *domain.LatencyCheck, dns *domain.DNSCheck, active *domain.Outage) string {
	if active != nil {
		return "down"
	}
	if latency != nil && !latency.Success && dns != nil && !dns.Success {
		return "down"
	}
	if latency != nil && (!latency.Success || (latency.LatencyMs != nil && *latency.LatencyMs > settings.MaxLatencyMs)) {
		return "degraded"
	}
	if dns != nil && !dns.Success {
		return "degraded"
	}
	if speed != nil && speed.Success {
		if speed.DownloadMbps != nil && *speed.DownloadMbps < settings.MinDownloadMbps {
			return "degraded"
		}
		if speed.UploadMbps != nil && *speed.UploadMbps < settings.MinUploadMbps {
			return "degraded"
		}
	}
	return "healthy"
}

func ShouldOpenOutage(consecutiveFailures, threshold int, active bool) bool {
	return !active && threshold > 0 && consecutiveFailures >= threshold
}

func ShouldCloseOutage(anySuccess bool, active bool) bool {
	return active && anySuccess
}
