package store_test

import (
	"context"
	"testing"
	"time"

	"homelink-monitor/services/api/internal/domain"
	"homelink-monitor/services/api/internal/testutil"
)

func TestStoreInsertAndReadRecords(t *testing.T) {
	_, st := testutil.DB(t)
	ctx := context.Background()
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	down := 123.4
	up := 56.7
	if _, err := st.InsertSpeedTest(ctx, domain.SpeedTest{StartedAt: now, FinishedAt: &now, DownloadMbps: &down, UploadMbps: &up, Success: true}); err != nil {
		t.Fatal(err)
	}
	latest, err := st.LatestSpeedTest(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if latest == nil || latest.DownloadMbps == nil || *latest.DownloadMbps != down {
		t.Fatalf("unexpected latest speed test: %#v", latest)
	}

	latency := 12.5
	if _, err := st.InsertLatency(ctx, domain.LatencyCheck{CheckedAt: now, Target: "1.1.1.1:53", LatencyMs: &latency, Success: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.InsertLatency(ctx, domain.LatencyCheck{CheckedAt: now, Target: "8.8.8.8:53", Success: false, Error: "timeout"}); err != nil {
		t.Fatal(err)
	}
	summary, err := st.LatencySummary(ctx, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Count != 2 || summary.PacketLoss != 50 || summary.AvgMs == nil || *summary.AvgMs != latency {
		t.Fatalf("unexpected summary: %#v", summary)
	}
}

func TestSettingsRoundTrip(t *testing.T) {
	_, st := testutil.DB(t)
	ctx := context.Background()
	settings := domain.DefaultSettings()
	settings.MinDownloadMbps = 250
	settings.LatencyTargets = []string{"router:80"}
	settings.MonitoringEnabled = false
	if err := st.SaveSettings(ctx, settings); err != nil {
		t.Fatal(err)
	}
	got, err := st.Settings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.MinDownloadMbps != 250 || len(got.LatencyTargets) != 1 || got.LatencyTargets[0] != "router:80" || got.MonitoringEnabled {
		t.Fatalf("unexpected settings: %#v", got)
	}
}

func TestRouterTrafficRoundTrip(t *testing.T) {
	_, st := testutil.DB(t)
	ctx := context.Background()
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	down := 1024.0
	up := 512.0
	sample := domain.RouterTrafficSample{
		CheckedAt:         now,
		Provider:          "tplink-web",
		Success:           true,
		ClientCount:       1,
		DownloadBps:       &down,
		UploadBps:         &up,
		DownloadAvailable: true,
		UploadAvailable:   true,
	}
	clients := []domain.RouterTrafficClient{{MAC: "aa:bb:cc:dd:ee:ff", Hostname: "laptop", DownloadBps: &down, UploadBps: &up}}
	if _, err := st.InsertRouterTraffic(ctx, sample, clients); err != nil {
		t.Fatal(err)
	}
	got, gotClients, err := st.LatestRouterTraffic(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.DownloadBps == nil || *got.DownloadBps != down || !got.DownloadAvailable {
		t.Fatalf("unexpected sample: %#v", got)
	}
	if len(gotClients) != 1 || gotClients[0].MAC != clients[0].MAC {
		t.Fatalf("unexpected clients: %#v", gotClients)
	}
}

func TestRouterTrafficClientUsageSinceIntegratesRates(t *testing.T) {
	_, st := testutil.DB(t)
	ctx := context.Background()
	start := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	down := 100.0
	up := 50.0
	client := domain.RouterTrafficClient{MAC: "aa:bb:cc:dd:ee:ff", Hostname: "laptop", DownloadBps: &down, UploadBps: &up}
	for _, checkedAt := range []time.Time{start, start.Add(10 * time.Second), start.Add(20 * time.Second)} {
		if _, err := st.InsertRouterTraffic(ctx, domain.RouterTrafficSample{
			CheckedAt:         checkedAt,
			Provider:          "tplink-web",
			Success:           true,
			ClientCount:       1,
			DownloadBps:       &down,
			UploadBps:         &up,
			DownloadAvailable: true,
			UploadAvailable:   true,
		}, []domain.RouterTrafficClient{client}); err != nil {
			t.Fatal(err)
		}
	}
	usage, err := st.RouterTrafficClientUsageSince(ctx, start)
	if err != nil {
		t.Fatal(err)
	}
	got := usage[client.MAC]
	if got.DownloadBytes == nil || *got.DownloadBytes != 2000 {
		t.Fatalf("unexpected download bytes: %#v", got.DownloadBytes)
	}
	if got.UploadBytes == nil || *got.UploadBytes != 1000 {
		t.Fatalf("unexpected upload bytes: %#v", got.UploadBytes)
	}
}
