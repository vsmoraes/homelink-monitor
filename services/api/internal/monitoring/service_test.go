package monitoring_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"homelink-monitor/services/api/internal/domain"
	"homelink-monitor/services/api/internal/monitoring"
	"homelink-monitor/services/api/internal/testutil"
)

func TestTriggerSpeedTestSurvivesCanceledRequestContext(t *testing.T) {
	_, st := testutil.DB(t)
	script := filepath.Join(t.TempDir(), "speedtest-fixture")
	body := "#!/bin/sh\nprintf '%s\n' '{\"download_mbps\":100,\"upload_mbps\":40,\"ping_ms\":8,\"jitter_ms\":1}'\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	settings := domain.DefaultSettings()
	settings.SpeedTestCommand = script
	if err := st.SaveSettings(context.Background(), settings); err != nil {
		t.Fatal(err)
	}

	service := monitoring.NewService(st, slog.Default())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if !service.TriggerSpeedTest(ctx) {
		t.Fatal("expected speed test to start")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !service.SpeedRunning() {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if service.SpeedRunning() {
		t.Fatal("speed test did not finish")
	}
	latest, err := st.LatestSpeedTest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if latest == nil || !latest.Success || latest.DownloadMbps == nil || *latest.DownloadMbps != 100 {
		t.Fatalf("unexpected speed test result: %#v", latest)
	}
}

func TestSpeedTestDueUsesLatestRunAndSchedule(t *testing.T) {
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	latest := &domain.SpeedTest{StartedAt: now.Add(-9 * time.Minute)}

	if monitoring.SpeedTestDue(now, latest, 0) {
		t.Fatal("disabled schedule should not be due")
	}
	if monitoring.SpeedTestDue(now, latest, 10) {
		t.Fatal("latest run is still inside the schedule window")
	}
	if !monitoring.SpeedTestDue(now, &domain.SpeedTest{StartedAt: now.Add(-10 * time.Minute)}, 10) {
		t.Fatal("latest run at the schedule boundary should be due")
	}
	if !monitoring.SpeedTestDue(now, nil, 10) {
		t.Fatal("missing latest run should be due")
	}
}
