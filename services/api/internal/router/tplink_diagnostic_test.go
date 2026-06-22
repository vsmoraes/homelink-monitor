package router

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestManualRouterEndpointDiagnostic(t *testing.T) {
	if os.Getenv("HOMELINK_ROUTER_DIAG") != "1" {
		t.Skip("set HOMELINK_ROUTER_DIAG=1 to probe a local router")
	}
	dbPath := os.Getenv("HOMELINK_ROUTER_DIAG_DB")
	if dbPath == "" {
		dbPath = "../../../data/connection-monitor.db"
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var raw string
	if err := db.QueryRow(`SELECT value FROM settings WHERE key = 'settings'`).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var settings struct {
		URL      string `json:"routerTrafficUrl"`
		Username string `json:"routerTrafficUsername"`
		Password string `json:"routerTrafficPassword"`
	}
	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		t.Fatal(err)
	}
	if overrideURL := strings.TrimSpace(os.Getenv("HOMELINK_ROUTER_DIAG_URL")); overrideURL != "" {
		settings.URL = overrideURL
	}
	if settings.URL == "" || settings.Password == "" {
		t.Fatal("router URL and password must be configured in settings")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	baseURL, err := normalizeBaseURL(settings.URL)
	if err != nil {
		t.Fatal(err)
	}
	s := &session{provider: NewProvider(nil), baseURL: baseURL, username: settings.Username, password: settings.Password}
	if err := s.login(ctx); err != nil {
		t.Fatal(err)
	}
	defer s.logout(context.Background())
	t.Logf("authenticated: base=%s stok_len=%d stok_starts_slash=%t", baseURL, len(s.stok), strings.HasPrefix(s.stok, "/"))

	for _, endpoint := range probeEndpoints {
		data, err := s.request(ctx, endpoint.path, endpoint.operation)
		if err != nil {
			t.Logf("%s: error=%v", endpoint.name, err)
			continue
		}
		t.Logf("%s: %s", endpoint.name, summarizeShape(data))
	}

	provider := NewProvider(nil)
	probeSettings := Settings{
		URL:      settings.URL,
		Username: settings.Username,
		Password: settings.Password,
	}
	start := time.Now()
	snapshot := provider.ProbeAndCollect(ctx, probeSettings)
	t.Logf("collector duration: %s", time.Since(start).Round(time.Millisecond))
	t.Logf("collector: success=%t clients=%d download_available=%t upload_available=%t error=%q",
		snapshot.Sample.Success,
		len(snapshot.Clients),
		snapshot.Sample.DownloadAvailable,
		snapshot.Sample.UploadAvailable,
		snapshot.Sample.Error,
	)
	if snapshot.Sample.DownloadBps != nil || snapshot.Sample.UploadBps != nil {
		t.Logf("collector totals: download_bps=%.0f upload_bps=%.0f", valueOrZero(snapshot.Sample.DownloadBps), valueOrZero(snapshot.Sample.UploadBps))
	}
	start = time.Now()
	snapshot = provider.ProbeAndCollect(ctx, probeSettings)
	t.Logf("cached collector duration: %s success=%t clients=%d", time.Since(start).Round(time.Millisecond), snapshot.Sample.Success, len(snapshot.Clients))
}

func valueOrZero(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func summarizeShape(v any) string {
	objects := collectObjects(v)
	keyCounts := map[string]int{}
	numericKeys := map[string]string{}
	for _, object := range objects {
		for key, value := range object {
			keyCounts[key]++
			if _, ok := numericKeys[key]; ok {
				continue
			}
			if n := numberValue(value); n != nil {
				numericKeys[key] = fmt.Sprintf("%.2f", *n)
			}
		}
	}
	keys := make([]string, 0, len(keyCounts))
	for key := range keyCounts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) > 80 {
		keys = keys[:80]
	}
	numerics := make([]string, 0, len(numericKeys))
	for key, value := range numericKeys {
		numerics = append(numerics, key+"="+value)
	}
	sort.Strings(numerics)
	if len(numerics) > 40 {
		numerics = numerics[:40]
	}
	return fmt.Sprintf("type=%T objects=%d keys=[%s] numeric=[%s]", v, len(objects), strings.Join(keys, ", "), strings.Join(numerics, ", "))
}
