package monitoring_test

import (
	"testing"

	"homelink-monitor/services/api/internal/domain"
	"homelink-monitor/services/api/internal/monitoring"
)

func TestConnectionStatus(t *testing.T) {
	settings := domain.DefaultSettings()
	latency := 20.0
	check := &domain.LatencyCheck{Success: true, LatencyMs: &latency}
	dns := &domain.DNSCheck{Success: true}
	if got := monitoring.ConnectionStatus(settings, nil, check, dns, nil); got != "healthy" {
		t.Fatalf("expected healthy, got %s", got)
	}
	latency = 500
	if got := monitoring.ConnectionStatus(settings, nil, check, dns, nil); got != "degraded" {
		t.Fatalf("expected degraded, got %s", got)
	}
	dns.Success = false
	check.Success = false
	if got := monitoring.ConnectionStatus(settings, nil, check, dns, nil); got != "down" {
		t.Fatalf("expected down, got %s", got)
	}
}

func TestOutageTransitions(t *testing.T) {
	if monitoring.ShouldOpenOutage(2, 3, false) {
		t.Fatal("opened too early")
	}
	if !monitoring.ShouldOpenOutage(3, 3, false) {
		t.Fatal("expected outage to open")
	}
	if monitoring.ShouldOpenOutage(4, 3, true) {
		t.Fatal("should not open second active outage")
	}
	if !monitoring.ShouldCloseOutage(true, true) {
		t.Fatal("expected active outage to close on success")
	}
}
