package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"homelink-monitor/services/api/internal/auth"
	"homelink-monitor/services/api/internal/domain"
	"homelink-monitor/services/api/internal/httpapi"
	"homelink-monitor/services/api/internal/monitoring"
	"homelink-monitor/services/api/internal/testutil"
)

func TestSummaryHandlerUsesRealDB(t *testing.T) {
	_, st := testutil.DB(t)
	now := time.Now().UTC()
	lat := 10.0
	if _, err := st.InsertLatency(context.Background(), domain.LatencyCheck{CheckedAt: now, Target: "1.1.1.1:53", LatencyMs: &lat, Success: true}); err != nil {
		t.Fatal(err)
	}
	authService := auth.NewService(st)
	if err := authService.EnsureInitialAdmin(context.Background(), "admin", "password123"); err != nil {
		t.Fatal(err)
	}
	server := httpapi.New(st, monitoring.NewService(st, slog.Default()), authService, slog.Default(), t.TempDir()).Routes()

	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"username":"admin","password":"password123"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRes := httptest.NewRecorder()
	server.ServeHTTP(loginRes, loginReq)
	if loginRes.Code != http.StatusOK {
		t.Fatalf("login status %d body %s", loginRes.Code, loginRes.Body.String())
	}
	cookies := loginRes.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/summary", nil)
	req.AddCookie(cookies[0])
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status %d body %s", res.Code, res.Body.String())
	}
	var summary domain.Summary
	if err := json.NewDecoder(res.Body).Decode(&summary); err != nil {
		t.Fatal(err)
	}
	if summary.Status != "healthy" || summary.LatestLatency == nil {
		t.Fatalf("unexpected summary: %#v", summary)
	}
}
