package speedtest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"homelink-monitor/services/api/internal/domain"
)

type Runner struct{}

func (Runner) Run(ctx context.Context, command string) domain.SpeedTest {
	started := time.Now().UTC()
	parts := strings.Fields(command)
	if len(parts) == 0 {
		finished := time.Now().UTC()
		return domain.SpeedTest{StartedAt: started, FinishedAt: &finished, Success: false, Error: "speed test command is empty"}
	}
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	finished := time.Now().UTC()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return domain.SpeedTest{StartedAt: started, FinishedAt: &finished, Success: false, Error: msg}
	}
	result, err := Parse(stdout.Bytes())
	if err != nil {
		return domain.SpeedTest{StartedAt: started, FinishedAt: &finished, Success: false, Error: err.Error()}
	}
	result.StartedAt = started
	result.FinishedAt = &finished
	result.Success = true
	return result
}

func Parse(raw []byte) (domain.SpeedTest, error) {
	var ookla struct {
		Type      string `json:"type"`
		Timestamp string `json:"timestamp"`
		Ping      struct {
			Latency float64 `json:"latency"`
			Jitter  float64 `json:"jitter"`
		} `json:"ping"`
		Download struct {
			Bandwidth float64 `json:"bandwidth"`
		} `json:"download"`
		Upload struct {
			Bandwidth float64 `json:"bandwidth"`
		} `json:"upload"`
		Server struct {
			Name     string `json:"name"`
			Location string `json:"location"`
		} `json:"server"`
	}
	if err := json.Unmarshal(raw, &ookla); err == nil && (ookla.Download.Bandwidth > 0 || ookla.Upload.Bandwidth > 0) {
		down := bytesPerSecondToMbps(ookla.Download.Bandwidth)
		up := bytesPerSecondToMbps(ookla.Upload.Bandwidth)
		ping := ookla.Ping.Latency
		jitter := ookla.Ping.Jitter
		return domain.SpeedTest{
			DownloadMbps: &down, UploadMbps: &up, PingMs: &ping, JitterMs: &jitter,
			ServerName: ookla.Server.Name, ServerLocation: ookla.Server.Location,
		}, nil
	}

	var simple struct {
		DownloadMbps   *float64 `json:"download_mbps"`
		UploadMbps     *float64 `json:"upload_mbps"`
		PingMs         *float64 `json:"ping_ms"`
		JitterMs       *float64 `json:"jitter_ms"`
		ServerName     string   `json:"server_name"`
		ServerLocation string   `json:"server_location"`
	}
	if err := json.Unmarshal(raw, &simple); err == nil && (simple.DownloadMbps != nil || simple.UploadMbps != nil) {
		return domain.SpeedTest{
			DownloadMbps: simple.DownloadMbps, UploadMbps: simple.UploadMbps, PingMs: simple.PingMs, JitterMs: simple.JitterMs,
			ServerName: simple.ServerName, ServerLocation: simple.ServerLocation,
		}, nil
	}
	return domain.SpeedTest{}, errors.New("could not parse speed test JSON output; configure a command that emits Ookla JSON or simple JSON")
}

func bytesPerSecondToMbps(v float64) float64 {
	return v * 8 / 1_000_000
}

func CommandHelp(command string) string {
	return fmt.Sprintf("configured speed test command %q failed; install the CLI in the container or update Settings", command)
}
