package services

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SnapshotService aggregates system state into zip archive.
type SnapshotService struct {
	monitor *MonitoringService
	system  *SystemService
}

// NewSnapshot constructs service.
func NewSnapshot(m *MonitoringService, sys *SystemService) *SnapshotService {
	return &SnapshotService{monitor: m, system: sys}
}

// Build generates zip buffer with diagnostics.
func (s *SnapshotService) Build(ctx context.Context) ([]byte, error) {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	addFile := func(name string, content string) error {
		f, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = f.Write([]byte(content))
		return err
	}

	health, _ := s.monitor.Health(ctx)
	res, _ := s.monitor.Resources(ctx)
	status, _ := s.monitor.Status(ctx)
	diag, _ := s.monitor.DiagNet(ctx)
	_ = addFile("health.txt", health)
	_ = addFile("resources.txt", res)
	_ = addFile("status.txt", status)
	_ = addFile("diag-net.txt", diag)

	// Attach last logs for key services
	if out, err := s.system.TailLogs(ctx, "tailscale", 100); err == nil {
		_ = addFile("logs-tailscale.txt", out)
	}
	if out, err := s.system.TailLogs(ctx, "cloudflared", 100); err == nil {
		_ = addFile("logs-cloudflared.txt", out)
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Save writes buffer to temp path for sending.
func (s *SnapshotService) Save(buf []byte, dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := filepath.Join(dir, fmt.Sprintf("snapshot-%d.zip", time.Now().Unix()))
	if err := os.WriteFile(name, buf, 0o640); err != nil {
		return "", err
	}
	return name, nil
}
