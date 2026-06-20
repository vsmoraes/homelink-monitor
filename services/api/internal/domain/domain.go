package domain

import "time"

type Settings struct {
	SpeedTestScheduleMinutes int      `json:"speedTestScheduleMinutes"`
	SpeedTestCommand         string   `json:"speedTestCommand"`
	LatencyIntervalSeconds   int      `json:"latencyIntervalSeconds"`
	DNSIntervalSeconds       int      `json:"dnsIntervalSeconds"`
	LatencyTargets           []string `json:"latencyTargets"`
	DNSDomains               []string `json:"dnsDomains"`
	RouterIP                 string   `json:"routerIp"`
	MinDownloadMbps          float64  `json:"minDownloadMbps"`
	MinUploadMbps            float64  `json:"minUploadMbps"`
	MaxLatencyMs             float64  `json:"maxLatencyMs"`
	OutageFailureThreshold   int      `json:"outageFailureThreshold"`
	MonitoringEnabled        bool     `json:"monitoringEnabled"`
}

func DefaultSettings() Settings {
	return Settings{
		SpeedTestScheduleMinutes: 360,
		SpeedTestCommand:         "speedtest --accept-license --accept-gdpr --format=json",
		LatencyIntervalSeconds:   60,
		DNSIntervalSeconds:       120,
		LatencyTargets:           []string{"1.1.1.1:53", "8.8.8.8:53"},
		DNSDomains:               []string{"google.com", "cloudflare.com"},
		MinDownloadMbps:          50,
		MinUploadMbps:            10,
		MaxLatencyMs:             100,
		OutageFailureThreshold:   3,
		MonitoringEnabled:        true,
	}
}

type SpeedTest struct {
	ID             int64      `json:"id"`
	StartedAt      time.Time  `json:"startedAt"`
	FinishedAt     *time.Time `json:"finishedAt,omitempty"`
	DownloadMbps   *float64   `json:"downloadMbps,omitempty"`
	UploadMbps     *float64   `json:"uploadMbps,omitempty"`
	PingMs         *float64   `json:"pingMs,omitempty"`
	JitterMs       *float64   `json:"jitterMs,omitempty"`
	ServerName     string     `json:"serverName,omitempty"`
	ServerLocation string     `json:"serverLocation,omitempty"`
	Success        bool       `json:"success"`
	Error          string     `json:"error,omitempty"`
}

type LatencyCheck struct {
	ID        int64     `json:"id"`
	CheckedAt time.Time `json:"checkedAt"`
	Target    string    `json:"target"`
	LatencyMs *float64  `json:"latencyMs,omitempty"`
	Success   bool      `json:"success"`
	Error     string    `json:"error,omitempty"`
}

type DNSCheck struct {
	ID         int64     `json:"id"`
	CheckedAt  time.Time `json:"checkedAt"`
	Domain     string    `json:"domain"`
	Resolver   string    `json:"resolver,omitempty"`
	DurationMs *float64  `json:"durationMs,omitempty"`
	Success    bool      `json:"success"`
	Error      string    `json:"error,omitempty"`
}

type Outage struct {
	ID        int64      `json:"id"`
	StartedAt time.Time  `json:"startedAt"`
	EndedAt   *time.Time `json:"endedAt,omitempty"`
	Reason    string     `json:"reason"`
}

type LatencySummary struct {
	MinMs      *float64 `json:"minMs,omitempty"`
	AvgMs      *float64 `json:"avgMs,omitempty"`
	MaxMs      *float64 `json:"maxMs,omitempty"`
	PacketLoss float64  `json:"packetLoss"`
	Count      int      `json:"count"`
}

type Summary struct {
	Status             string         `json:"status"`
	LatestSpeedTest    *SpeedTest     `json:"latestSpeedTest,omitempty"`
	LatestLatency      *LatencyCheck  `json:"latestLatency,omitempty"`
	LatestDNSCheck     *DNSCheck      `json:"latestDnsCheck,omitempty"`
	Latency24h         LatencySummary `json:"latency24h"`
	OutageCount24h     int            `json:"outageCount24h"`
	ActiveOutage       *Outage        `json:"activeOutage,omitempty"`
	MinDownload24hMbps *float64       `json:"minDownload24hMbps,omitempty"`
	MaxDownload24hMbps *float64       `json:"maxDownload24hMbps,omitempty"`
	MinUpload24hMbps   *float64       `json:"minUpload24hMbps,omitempty"`
	MaxUpload24hMbps   *float64       `json:"maxUpload24hMbps,omitempty"`
	Settings           Settings       `json:"settings"`
	SpeedTestIsRunning bool           `json:"speedTestIsRunning"`
}

type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
