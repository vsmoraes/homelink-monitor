package router

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExtractClientsDetectsSeparateUploadDownload(t *testing.T) {
	responses := []endpointResponse{{
		source: "status/all",
		data: map[string]any{
			"devices": []any{
				map[string]any{
					"macaddr":    "aa:bb:cc:dd:ee:ff",
					"ipaddr":     "192.168.1.20",
					"hostname":   "laptop",
					"down_speed": float64(1024),
					"up_speed":   float64(512),
				},
			},
		},
	}}

	clients := ExtractClients(responses)
	if len(clients) != 1 {
		t.Fatalf("expected one client, got %#v", clients)
	}
	if clients[0].DownloadBps == nil || *clients[0].DownloadBps != 1024 {
		t.Fatalf("download speed not detected: %#v", clients[0])
	}
	if clients[0].UploadBps == nil || *clients[0].UploadBps != 512 {
		t.Fatalf("upload speed not detected: %#v", clients[0])
	}
}

func TestExtractClientsDoesNotInventUploadDownloadFromTotal(t *testing.T) {
	responses := []endpointResponse{{
		source: "wireless/statistics",
		data: map[string]any{
			"clients": []any{
				map[string]any{
					"mac":           "aa:bb:cc:dd:ee:ff",
					"traffic_usage": float64(4096),
				},
			},
		},
	}}

	clients := ExtractClients(responses)
	if len(clients) != 1 {
		t.Fatalf("expected one client, got %#v", clients)
	}
	if clients[0].DownloadBps != nil || clients[0].UploadBps != nil {
		t.Fatalf("unexpected split traffic values: %#v", clients[0])
	}
	if clients[0].TotalBytes == nil || *clients[0].TotalBytes != 4096 {
		t.Fatalf("total traffic not detected: %#v", clients[0])
	}
}

func TestBE550LoginUsesPlainRSAFormAndReadsTraffic(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	passwordModulus := fmt.Sprintf("%x", privateKey.PublicKey.N)
	passwordExponent := fmt.Sprintf("%x", privateKey.PublicKey.E)

	var sawLogin bool
	var sawTraffic bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawBody, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		body := string(rawBody)
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(r.URL.RequestURI(), "/login?form=keys"):
			assertFormBody(t, body, "operation=read")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"data": map[string]any{
					"password": []string{passwordModulus, passwordExponent},
				},
			})
		case strings.Contains(r.URL.RequestURI(), "/login?form=login"):
			sawLogin = true
			if strings.Contains(body, "sign=") || strings.Contains(body, "data=") {
				t.Fatalf("BE550 login must not use SG sign/data body: %q", body)
			}
			if !strings.HasPrefix(body, "operation=login&password=") {
				t.Fatalf("unexpected BE550 login body: %q", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"data":    map[string]any{"stok": "abc123"},
			})
		case strings.Contains(r.URL.RequestURI(), "admin/smart_network?form=game_accelerator"):
			if body == "operation=loadDevice" {
				_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"clientList": []map[string]any{}}})
				return
			}
			if body != "operation=loadSpeed" {
				t.Fatalf("unexpected game accelerator body: %q", body)
			}
			sawTraffic = true
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"data": map[string]any{
					"clientList": []map[string]any{{
						"mac":           "aa:bb:cc:dd:ee:ff",
						"ip":            "192.168.0.20",
						"deviceName":    "laptop",
						"downloadSpeed": 1250,
						"uploadSpeed":   250,
					}},
				},
			})
		case strings.Contains(r.URL.RequestURI(), "admin/system?form=logout"):
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
		default:
			http.Error(w, "unexpected request: "+r.URL.RequestURI(), http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewProvider(server.Client())
	snapshot := provider.ProbeAndCollect(context.Background(), Settings{URL: server.URL, Password: "changeme123"})

	if !sawLogin {
		t.Fatal("BE550 login endpoint was not called")
	}
	if !sawTraffic {
		t.Fatal("traffic endpoint was not called")
	}
	if !snapshot.Capability.Authenticated || !snapshot.Sample.Success {
		t.Fatalf("expected successful snapshot, got %#v", snapshot)
	}
	if len(snapshot.Clients) != 1 {
		t.Fatalf("expected one client, got %#v", snapshot.Clients)
	}
	assertFloatPtr(t, snapshot.Clients[0].DownloadBps, 1250)
	assertFloatPtr(t, snapshot.Clients[0].UploadBps, 250)
	assertFloatPtr(t, snapshot.Sample.DownloadBps, 1250)
	assertFloatPtr(t, snapshot.Sample.UploadBps, 250)
	if snapshot.Clients[0].MAC != "aa:bb:cc:dd:ee:ff" {
		t.Fatalf("unexpected client: %#v", snapshot.Clients[0])
	}
}

func TestExtractClientsDetectsBE550CamelCaseRates(t *testing.T) {
	responses := []endpointResponse{{
		source: "qos/game_accelerator_speeds",
		data: map[string]any{
			"clientList": []any{
				map[string]any{
					"mac":           "aa:bb:cc:dd:ee:ff",
					"ip":            "192.168.0.20",
					"deviceName":    "laptop",
					"downloadSpeed": float64(1250),
					"uploadSpeed":   float64(250),
					"trafficUsage":  float64(8192),
				},
			},
		},
	}}

	clients := ExtractClients(responses)
	if len(clients) != 1 {
		t.Fatalf("expected one client, got %#v", clients)
	}
	assertFloatPtr(t, clients[0].DownloadBps, 1250)
	assertFloatPtr(t, clients[0].UploadBps, 250)
	assertFloatPtr(t, clients[0].TotalBytes, 8192)
}

func assertFormBody(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("expected body %q, got %q", want, got)
	}
}

func assertFloatPtr(t *testing.T, got *float64, want float64) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("expected %v, got %#v", want, got)
	}
}
