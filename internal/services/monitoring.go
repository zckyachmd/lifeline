package services

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"zckyachmd/lifeline/internal/api"
)

// MonitoringService wraps visibility operations.
type MonitoringService struct {
	dsm  *api.Client
	http *http.Client
}

// NewMonitoring creates monitoring service.
func NewMonitoring(dsm *api.Client) *MonitoringService {
	return &MonitoringService{
		dsm:  dsm,
		http: &http.Client{Timeout: 5 * time.Second},
	}
}

// Health returns combined health summary.
func (m *MonitoringService) Health(ctx context.Context) (string, error) {
	sys, err := m.dsm.SystemHealth(ctx)
	if err != nil {
		return "", err
	}
	res, _ := m.Resources(ctx)
	return fmt.Sprintf("DSM OK: %v\nResources:\n%s", sys["data"], res), nil
}

// Resources reports CPU/memory/disk.
func (m *MonitoringService) Resources(ctx context.Context) (string, error) {
	data, err := m.dsm.ResourceUsage(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("resource=%v", data["data"]), nil
}

// Status checks key services.
func (m *MonitoringService) Status(ctx context.Context) (string, error) {
	parts := []string{}
	checks := map[string][]string{
		"cloudflared": {"docker", "inspect", "-f", "{{.State.Status}}", "cloudflared"},
		"tailscale":   {"systemctl", "is-active", "tailscaled.service"},
		"docker":      {"systemctl", "is-active", "docker.service"},
	}
	for name, cmd := range checks {
		out, err := runCommand(ctx, cmd)
		if err != nil {
			parts = append(parts, fmt.Sprintf("%s=error:%v", name, err))
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%s", name, strings.TrimSpace(out)))
	}
	return strings.Join(parts, " \n"), nil
}

// DiagNet runs minimal network diagnostics.
func (m *MonitoringService) DiagNet(ctx context.Context) (string, error) {
	output, err := runCommand(ctx, []string{"ping", "-c", "1", "1.1.1.1"})
	if err != nil {
		return "", err
	}
	return output, nil
}

// DiagTime checks NTP sync and clock.
func (m *MonitoringService) DiagTime(ctx context.Context) (string, error) {
	out, err := runCommand(ctx, []string{"timedatectl"})
	if err != nil {
		return "", err
	}
	return out, nil
}

// PublicIP resolves external IP.
func (m *MonitoringService) PublicIP(ctx context.Context) (string, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.ipify.org", nil)
	resp, err := m.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func runCommand(ctx context.Context, parts []string) (string, error) {
	if len(parts) == 0 {
		return "", fmt.Errorf("empty command")
	}
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	b, err := cmd.CombinedOutput()
	return string(b), err
}
