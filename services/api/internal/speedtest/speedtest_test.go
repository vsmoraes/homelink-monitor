package speedtest_test

import (
	"testing"

	"homelink-monitor/services/api/internal/speedtest"
)

func TestParseOoklaJSON(t *testing.T) {
	raw := []byte(`{"type":"result","ping":{"jitter":1.2,"latency":9.8},"download":{"bandwidth":12500000},"upload":{"bandwidth":2500000},"server":{"name":"Test ISP","location":"Madrid"}}`)
	result, err := speedtest.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if result.DownloadMbps == nil || *result.DownloadMbps != 100 {
		t.Fatalf("unexpected download: %#v", result.DownloadMbps)
	}
	if result.UploadMbps == nil || *result.UploadMbps != 20 {
		t.Fatalf("unexpected upload: %#v", result.UploadMbps)
	}
	if result.ServerName != "Test ISP" {
		t.Fatalf("unexpected server: %s", result.ServerName)
	}
}
