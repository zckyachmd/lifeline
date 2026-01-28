package services

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// SystemService wraps controlled system actions.
type SystemService struct{}

// RestartService restarts a known service via controlled adapters.
func (s *SystemService) RestartService(ctx context.Context, service string) (string, error) {
	name := strings.ToLower(service)
	switch name {
	case "cloudflared":
		return runCmd(ctx, "docker", "restart", "cloudflared")
	case "tailscale", "tailscaled":
		return runCmd(ctx, "systemctl", "restart", "tailscaled.service")
	case "docker":
		return runCmd(ctx, "systemctl", "restart", "docker.service")
	default:
		return "", fmt.Errorf("service not allowed")
	}
}

// Cleanup performs docker prune scoped action.
func (s *SystemService) Cleanup(ctx context.Context) (string, error) {
	return runCmd(ctx, "docker", "system", "prune", "-f")
}

// Reboot reboots host as last resort.
func (s *SystemService) Reboot(ctx context.Context) (string, error) {
	return runCmd(ctx, "systemctl", "reboot")
}

// TailLogs returns last lines of a service.
func (s *SystemService) TailLogs(ctx context.Context, service string, lines int) (string, error) {
	if lines <= 0 || lines > 500 {
		lines = 100
	}
	name := strings.ToLower(service)
	switch name {
	case "cloudflared":
		return runCmd(ctx, "docker", "logs", "--tail", fmt.Sprintf("%d", lines), "cloudflared")
	case "tailscale", "tailscaled":
		cmd := exec.CommandContext(ctx, "journalctl", "-u", "tailscaled.service", "-n", fmt.Sprintf("%d", lines))
		b, err := cmd.CombinedOutput()
		return string(b), err
	case "docker":
		cmd := exec.CommandContext(ctx, "journalctl", "-u", "docker.service", "-n", fmt.Sprintf("%d", lines))
		b, err := cmd.CombinedOutput()
		return string(b), err
	default:
		return "", fmt.Errorf("service not allowed")
	}
}

func runCmd(ctx context.Context, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	b, err := cmd.CombinedOutput()
	return string(b), err
}

func allowedService(name string) bool {
	allowed := []string{"tailscale", "tailscaled", "cloudflared", "docker"}
	for _, v := range allowed {
		if v == strings.ToLower(name) {
			return true
		}
	}
	return false
}

// IsAllowedService exposes allowlist for external checks.
func (s *SystemService) IsAllowedService(name string) bool {
	return allowedService(name)
}
