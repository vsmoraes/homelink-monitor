package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"homelink-monitor/services/api/internal/domain"
)

func (s *Store) InsertRouterTraffic(ctx context.Context, sample domain.RouterTrafficSample, clients []domain.RouterTrafficClient) (int64, error) {
	if clients == nil {
		clients = []domain.RouterTrafficClient{}
	}
	rawClients, err := json.Marshal(clients)
	if err != nil {
		return 0, err
	}
	res, err := s.db.ExecContext(ctx, `INSERT INTO router_traffic_samples(
		checked_at, provider, success, error, client_count, download_bps, upload_bps, total_bps,
		download_available, upload_available, total_traffic_available, clients_json
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ts(sample.CheckedAt), sample.Provider, boolInt(sample.Success), sample.Error, sample.ClientCount,
		nullableFloat(sample.DownloadBps), nullableFloat(sample.UploadBps), nullableFloat(sample.TotalBps),
		boolInt(sample.DownloadAvailable), boolInt(sample.UploadAvailable), boolInt(sample.TotalTrafficAvailable), string(rawClients))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) LatestRouterTraffic(ctx context.Context) (*domain.RouterTrafficSample, []domain.RouterTrafficClient, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, checked_at, provider, success, error, client_count, download_bps, upload_bps, total_bps,
		download_available, upload_available, total_traffic_available, clients_json
		FROM router_traffic_samples ORDER BY checked_at DESC, id DESC LIMIT 1`)
	return scanRouterTraffic(row)
}

func (s *Store) RouterTrafficSamples(ctx context.Context, limit int) ([]domain.RouterTrafficSample, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, checked_at, provider, success, error, client_count, download_bps, upload_bps, total_bps,
		download_available, upload_available, total_traffic_available, clients_json
		FROM router_traffic_samples ORDER BY checked_at DESC, id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.RouterTrafficSample{}
	for rows.Next() {
		sample, _, err := scanRouterTraffic(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *sample)
	}
	return out, rows.Err()
}

func scanRouterTraffic(row scanner) (*domain.RouterTrafficSample, []domain.RouterTrafficClient, error) {
	var sample domain.RouterTrafficSample
	var checked string
	var success, downAvailable, upAvailable, totalAvailable int
	var down, up, total sql.NullFloat64
	var clientsJSON string
	err := row.Scan(&sample.ID, &checked, &sample.Provider, &success, &sample.Error, &sample.ClientCount, &down, &up, &total, &downAvailable, &upAvailable, &totalAvailable, &clientsJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	checkedAt, err := parseTS(checked)
	if err != nil {
		return nil, nil, err
	}
	sample.CheckedAt = checkedAt
	sample.Success = success == 1
	sample.DownloadBps = nullFloatPtr(down)
	sample.UploadBps = nullFloatPtr(up)
	sample.TotalBps = nullFloatPtr(total)
	sample.DownloadAvailable = downAvailable == 1
	sample.UploadAvailable = upAvailable == 1
	sample.TotalTrafficAvailable = totalAvailable == 1
	clients := []domain.RouterTrafficClient{}
	if clientsJSON != "" {
		if err := json.Unmarshal([]byte(clientsJSON), &clients); err != nil {
			return nil, nil, err
		}
	}
	return &sample, clients, nil
}
