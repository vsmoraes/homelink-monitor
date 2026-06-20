package events_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"homelink-monitor/services/api/internal/domain"
	"homelink-monitor/services/api/internal/events"
)

func TestOutboxWritesNotificationEvent(t *testing.T) {
	dir := t.TempDir()
	outbox := events.NewOutbox(dir)
	event := domain.NotificationEvent{
		ID:        "latency-test",
		Severity:  "warning",
		Metric:    "latency",
		Title:     "Latency check failed",
		Message:   "Latency target failed",
		Timestamp: time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC),
	}

	if err := outbox.Write(context.Background(), event); err != nil {
		t.Fatalf("write event: %v", err)
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		t.Fatalf("glob events: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected one event file, got %d", len(files))
	}
	raw, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatalf("read event: %v", err)
	}
	var got domain.NotificationEvent
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("parse event: %v", err)
	}
	if got.ID != event.ID || got.Metric != event.Metric || got.Severity != event.Severity {
		t.Fatalf("unexpected event: %#v", got)
	}
}

func TestOutboxDisabledWhenDirectoryIsEmpty(t *testing.T) {
	outbox := events.NewOutbox("")
	if outbox.Enabled() {
		t.Fatal("expected empty outbox directory to disable writes")
	}
	if err := outbox.Write(context.Background(), domain.NotificationEvent{}); err != nil {
		t.Fatalf("disabled outbox should ignore writes: %v", err)
	}
}
