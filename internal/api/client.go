package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

// Client wraps Synology DSM API endpoints used by the bot.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// NewClient creates a new DSM client.
func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

// SystemHealth fetches system health info.
func (c *Client) SystemHealth(ctx context.Context) (map[string]any, error) {
	return c.get(ctx, "/webapi/entry.cgi", url.Values{
		"api":     {"SYNO.Core.System"},
		"version": {"1"},
		"method":  {"info"},
	})
}

// ResourceUsage fetches CPU/Mem/disk stats.
func (c *Client) ResourceUsage(ctx context.Context) (map[string]any, error) {
	return c.get(ctx, "/webapi/entry.cgi", url.Values{
		"api":     {"SYNO.Core.System.Utilization"},
		"version": {"1"},
		"method":  {"get"},
	})
}

// ListFiles lists a path through File Station API.
func (c *Client) ListFiles(ctx context.Context, folder string) (map[string]any, error) {
	return c.get(ctx, "/webapi/entry.cgi", url.Values{
		"api":         {"SYNO.FileStation.List"},
		"version":     {"2"},
		"method":      {"list"},
		"folder_path": {folder},
	})
}

// DownloadFile returns raw bytes for a given DSM path.
func (c *Client) DownloadFile(ctx context.Context, filePath string) ([]byte, error) {
	values := url.Values{
		"api":     {"SYNO.FileStation.Download"},
		"version": {"2"},
		"method":  {"download"},
		"path":    {filePath},
	}
	endpoint := c.buildURL("/webapi/entry.cgi") + "?" + values.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	c.attachAuth(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("dsm download status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// UploadFile uploads content to DSM File Station.
func (c *Client) UploadFile(ctx context.Context, destFolder, filename string, content []byte) error {
	values := url.Values{
		"api":            {"SYNO.FileStation.Upload"},
		"version":        {"2"},
		"method":         {"upload"},
		"path":           {destFolder},
		"create_parents": {"true"},
	}
	endpoint := c.buildURL("/webapi/entry.cgi") + "?" + values.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(content))
	if err != nil {
		return err
	}
	c.attachAuth(req)
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed: %s", string(body))
	}
	return nil
}

// RestartService calls DSM to restart a service.
func (c *Client) RestartService(ctx context.Context, service string) error {
	_, err := c.get(ctx, "/webapi/entry.cgi", url.Values{
		"api":     {"SYNO.Core.Service"},
		"version": {"1"},
		"method":  {"restart"},
		"service": {service},
	})
	return err
}

// RotateToken refreshes API token placeholder.
func (c *Client) RotateToken(newToken string) {
	c.token = newToken
}

func (c *Client) get(ctx context.Context, p string, q url.Values) (map[string]any, error) {
	endpoint := c.buildURL(p) + "?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	c.attachAuth(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("dsm status %d", resp.StatusCode)
	}
	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Client) attachAuth(req *http.Request) {
	req.Header.Set("X-SYNO-TOKEN", c.token)
}

func (c *Client) buildURL(p string) string {
	base, _ := url.Parse(c.baseURL)
	base.Path = path.Join(base.Path, p)
	return base.String()
}
