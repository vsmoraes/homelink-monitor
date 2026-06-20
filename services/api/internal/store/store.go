package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"homelink-monitor/services/api/internal/domain"
)

type Store struct{ db *sql.DB }

func New(db *sql.DB) *Store { return &Store{db: db} }

func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

func (s *Store) Settings(ctx context.Context) (domain.Settings, error) {
	settings := domain.DefaultSettings()
	rows, err := s.db.QueryContext(ctx, `SELECT key, value FROM settings`)
	if err != nil {
		return settings, err
	}
	defer rows.Close()
	values := map[string]string{}
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return settings, err
		}
		values[key] = value
	}
	if raw, ok := values["settings"]; ok {
		if err := json.Unmarshal([]byte(raw), &settings); err != nil {
			return settings, err
		}
	}
	return settings, rows.Err()
}

func (s *Store) SaveSettings(ctx context.Context, settings domain.Settings) error {
	raw, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO settings(key, value) VALUES('settings', ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, string(raw))
	return err
}

func (s *Store) InsertSpeedTest(ctx context.Context, item domain.SpeedTest) (int64, error) {
	res, err := s.db.ExecContext(ctx, `INSERT INTO speed_tests(started_at, finished_at, download_mbps, upload_mbps, ping_ms, jitter_ms, server_name, server_location, success, error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ts(item.StartedAt), nullableTime(item.FinishedAt), nullableFloat(item.DownloadMbps), nullableFloat(item.UploadMbps), nullableFloat(item.PingMs), nullableFloat(item.JitterMs), item.ServerName, item.ServerLocation, boolInt(item.Success), item.Error)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) LatestSpeedTest(ctx context.Context) (*domain.SpeedTest, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, started_at, finished_at, download_mbps, upload_mbps, ping_ms, jitter_ms, server_name, server_location, success, error FROM speed_tests ORDER BY started_at DESC, id DESC LIMIT 1`)
	return scanSpeed(row)
}

func (s *Store) SpeedTests(ctx context.Context, limit, offset int, from, to *time.Time) ([]domain.SpeedTest, error) {
	q := `SELECT id, started_at, finished_at, download_mbps, upload_mbps, ping_ms, jitter_ms, server_name, server_location, success, error FROM speed_tests WHERE (? IS NULL OR started_at >= ?) AND (? IS NULL OR started_at <= ?) ORDER BY started_at DESC, id DESC LIMIT ? OFFSET ?`
	rows, err := s.db.QueryContext(ctx, q, nullableTime(from), nullableTime(from), nullableTime(to), nullableTime(to), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.SpeedTest
	for rows.Next() {
		item, err := scanSpeedRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func (s *Store) InsertLatency(ctx context.Context, item domain.LatencyCheck) (int64, error) {
	res, err := s.db.ExecContext(ctx, `INSERT INTO latency_checks(checked_at, target, latency_ms, success, error) VALUES (?, ?, ?, ?, ?)`,
		ts(item.CheckedAt), item.Target, nullableFloat(item.LatencyMs), boolInt(item.Success), item.Error)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) Latency(ctx context.Context, target string, limit int, from, to *time.Time) ([]domain.LatencyCheck, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, checked_at, target, latency_ms, success, error FROM latency_checks
		WHERE (? = '' OR target = ?) AND (? IS NULL OR checked_at >= ?) AND (? IS NULL OR checked_at <= ?)
		ORDER BY checked_at DESC, id DESC LIMIT ?`, target, target, nullableTime(from), nullableTime(from), nullableTime(to), nullableTime(to), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.LatencyCheck
	for rows.Next() {
		item, err := scanLatencyRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func (s *Store) LatestLatency(ctx context.Context) (*domain.LatencyCheck, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, checked_at, target, latency_ms, success, error FROM latency_checks ORDER BY checked_at DESC, id DESC LIMIT 1`)
	return scanLatency(row)
}

func (s *Store) LatencySummary(ctx context.Context, from, to *time.Time) (domain.LatencySummary, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT latency_ms, success FROM latency_checks WHERE (? IS NULL OR checked_at >= ?) AND (? IS NULL OR checked_at <= ?)`, nullableTime(from), nullableTime(from), nullableTime(to), nullableTime(to))
	if err != nil {
		return domain.LatencySummary{}, err
	}
	defer rows.Close()
	var count, failed, okCount int
	var min, max, sum float64
	for rows.Next() {
		var latency sql.NullFloat64
		var success int
		if err := rows.Scan(&latency, &success); err != nil {
			return domain.LatencySummary{}, err
		}
		count++
		if success == 0 {
			failed++
			continue
		}
		if latency.Valid {
			if okCount == 0 || latency.Float64 < min {
				min = latency.Float64
			}
			if okCount == 0 || latency.Float64 > max {
				max = latency.Float64
			}
			sum += latency.Float64
			okCount++
		}
	}
	var summary domain.LatencySummary
	summary.Count = count
	if count > 0 {
		summary.PacketLoss = float64(failed) / float64(count) * 100
	}
	if okCount > 0 {
		avg := sum / float64(okCount)
		summary.MinMs, summary.AvgMs, summary.MaxMs = &min, &avg, &max
	}
	return summary, rows.Err()
}

func (s *Store) InsertDNS(ctx context.Context, item domain.DNSCheck) (int64, error) {
	res, err := s.db.ExecContext(ctx, `INSERT INTO dns_checks(checked_at, domain, resolver, duration_ms, success, error) VALUES (?, ?, ?, ?, ?, ?)`,
		ts(item.CheckedAt), item.Domain, item.Resolver, nullableFloat(item.DurationMs), boolInt(item.Success), item.Error)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) DNSChecks(ctx context.Context, limit int) ([]domain.DNSCheck, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, checked_at, domain, resolver, duration_ms, success, error FROM dns_checks ORDER BY checked_at DESC, id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.DNSCheck
	for rows.Next() {
		item, err := scanDNSRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func (s *Store) LatestDNS(ctx context.Context) (*domain.DNSCheck, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, checked_at, domain, resolver, duration_ms, success, error FROM dns_checks ORDER BY checked_at DESC, id DESC LIMIT 1`)
	return scanDNS(row)
}

func (s *Store) OpenOutage(ctx context.Context, started time.Time, reason string) error {
	var active int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM outages WHERE ended_at IS NULL`).Scan(&active); err != nil {
		return err
	}
	if active > 0 {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO outages(started_at, reason) VALUES (?, ?)`, ts(started), reason)
	return err
}

func (s *Store) CloseActiveOutage(ctx context.Context, ended time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE outages SET ended_at = ? WHERE ended_at IS NULL`, ts(ended))
	return err
}

func (s *Store) ActiveOutage(ctx context.Context) (*domain.Outage, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, started_at, ended_at, reason FROM outages WHERE ended_at IS NULL ORDER BY started_at DESC LIMIT 1`)
	return scanOutage(row)
}

func (s *Store) Outages(ctx context.Context, limit int) ([]domain.Outage, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, started_at, ended_at, reason FROM outages ORDER BY started_at DESC, id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Outage
	for rows.Next() {
		item, err := scanOutageRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func (s *Store) OutageCountSince(ctx context.Context, since time.Time) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM outages WHERE started_at >= ?`, ts(since)).Scan(&count)
	return count, err
}

func (s *Store) SpeedBoundsSince(ctx context.Context, since time.Time) (minDown, maxDown, minUp, maxUp *float64, err error) {
	row := s.db.QueryRowContext(ctx, `SELECT MIN(download_mbps), MAX(download_mbps), MIN(upload_mbps), MAX(upload_mbps) FROM speed_tests WHERE success = 1 AND started_at >= ?`, ts(since))
	var a, b, c, d sql.NullFloat64
	if err := row.Scan(&a, &b, &c, &d); err != nil {
		return nil, nil, nil, nil, err
	}
	return nullFloatPtr(a), nullFloatPtr(b), nullFloatPtr(c), nullFloatPtr(d), nil
}

func nullableFloat(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullableTime(v *time.Time) any {
	if v == nil {
		return nil
	}
	return ts(*v)
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func ts(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func parseTS(v string) (time.Time, error) { return time.Parse(time.RFC3339Nano, v) }

func nullFloatPtr(v sql.NullFloat64) *float64 {
	if !v.Valid {
		return nil
	}
	x := v.Float64
	return &x
}

type scanner interface{ Scan(dest ...any) error }

func scanSpeed(row scanner) (*domain.SpeedTest, error) {
	item, err := scanSpeedAny(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return item, err
}

func scanSpeedRows(row scanner) (*domain.SpeedTest, error) { return scanSpeedAny(row) }

func scanSpeedAny(row scanner) (*domain.SpeedTest, error) {
	var item domain.SpeedTest
	var started, finished sql.NullString
	var down, up, ping, jitter sql.NullFloat64
	var success int
	if err := row.Scan(&item.ID, &started, &finished, &down, &up, &ping, &jitter, &item.ServerName, &item.ServerLocation, &success, &item.Error); err != nil {
		return nil, err
	}
	t, err := parseTS(started.String)
	if err != nil {
		return nil, err
	}
	item.StartedAt = t
	if finished.Valid {
		t, err := parseTS(finished.String)
		if err != nil {
			return nil, err
		}
		item.FinishedAt = &t
	}
	item.DownloadMbps = nullFloatPtr(down)
	item.UploadMbps = nullFloatPtr(up)
	item.PingMs = nullFloatPtr(ping)
	item.JitterMs = nullFloatPtr(jitter)
	item.Success = success == 1
	return &item, nil
}

func scanLatency(row scanner) (*domain.LatencyCheck, error) {
	item, err := scanLatencyRows(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return item, err
}

func scanLatencyRows(row scanner) (*domain.LatencyCheck, error) {
	var item domain.LatencyCheck
	var checked string
	var latency sql.NullFloat64
	var success int
	if err := row.Scan(&item.ID, &checked, &item.Target, &latency, &success, &item.Error); err != nil {
		return nil, err
	}
	t, err := parseTS(checked)
	if err != nil {
		return nil, err
	}
	item.CheckedAt = t
	item.LatencyMs = nullFloatPtr(latency)
	item.Success = success == 1
	return &item, nil
}

func scanDNS(row scanner) (*domain.DNSCheck, error) {
	item, err := scanDNSRows(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return item, err
}

func scanDNSRows(row scanner) (*domain.DNSCheck, error) {
	var item domain.DNSCheck
	var checked string
	var duration sql.NullFloat64
	var success int
	if err := row.Scan(&item.ID, &checked, &item.Domain, &item.Resolver, &duration, &success, &item.Error); err != nil {
		return nil, err
	}
	t, err := parseTS(checked)
	if err != nil {
		return nil, err
	}
	item.CheckedAt = t
	item.DurationMs = nullFloatPtr(duration)
	item.Success = success == 1
	return &item, nil
}

func scanOutage(row scanner) (*domain.Outage, error) {
	item, err := scanOutageRows(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return item, err
}

func scanOutageRows(row scanner) (*domain.Outage, error) {
	var item domain.Outage
	var started string
	var ended sql.NullString
	if err := row.Scan(&item.ID, &started, &ended, &item.Reason); err != nil {
		return nil, err
	}
	t, err := parseTS(started)
	if err != nil {
		return nil, err
	}
	item.StartedAt = t
	if ended.Valid {
		e, err := parseTS(ended.String)
		if err != nil {
			return nil, err
		}
		item.EndedAt = &e
	}
	return &item, nil
}
