package events

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"homelink-monitor/services/api/internal/domain"
)

type Outbox struct {
	dir string
}

func NewOutbox(dir string) *Outbox {
	return &Outbox{dir: dir}
}

func (o *Outbox) Enabled() bool {
	return strings.TrimSpace(o.dir) != ""
}

func (o *Outbox) Write(ctx context.Context, event domain.NotificationEvent) error {
	if !o.Enabled() {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if event.ID == "" || event.Severity == "" || event.Metric == "" || event.Title == "" || event.Message == "" {
		return errors.New("notification event is missing required fields")
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if err := os.MkdirAll(o.dir, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		return err
	}
	name := safeName(event.Timestamp.Format("20060102T150405.000000000Z") + "-" + event.ID + ".json")
	tmp := filepath.Join(o.dir, "."+name+".tmp")
	final := filepath.Join(o.dir, name)
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, final)
}

func safeName(name string) string {
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '_' {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('-')
	}
	return b.String()
}
