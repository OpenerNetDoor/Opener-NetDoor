package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL   string
	token     string
	tenantID  string
	http      *http.Client
	userAgent string
}

func New(cfg Config) (*Client, error) {
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}
	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	ua := strings.TrimSpace(cfg.UserAgent)
	if ua == "" {
		ua = "opener-netdoor-sdk-go/0.2.0"
	}
	return &Client{
		baseURL:   strings.TrimRight(baseURL, "/"),
		token:     strings.TrimSpace(cfg.Token),
		tenantID:  strings.TrimSpace(cfg.TenantID),
		http:      &http.Client{Timeout: timeout},
		userAgent: ua,
	}, nil
}

func (c *Client) Health(ctx context.Context) (HealthResponse, error) {
	var out HealthResponse
	err := c.getJSON(ctx, "/v1/health", nil, &out)
	return out, err
}

func (c *Client) Ready(ctx context.Context) (HealthResponse, error) {
	var out HealthResponse
	err := c.getJSON(ctx, "/v1/ready", nil, &out)
	return out, err
}

func (c *Client) AuditLogs(ctx context.Context, limit int, offset int) (AuditLogListResponse, error) {
	var out AuditLogListResponse
	query := map[string]string{
		"limit":  fmt.Sprint(limit),
		"offset": fmt.Sprint(offset),
	}
	if c.tenantID != "" {
		query["tenant_id"] = c.tenantID
	}
	err := c.getJSON(ctx, "/v1/admin/audit/logs", query, &out)
	return out, err
}

func (c *Client) OpsSnapshot(ctx context.Context) (OpsSnapshot, error) {
	var out OpsSnapshot
	query := map[string]string{}
	if c.tenantID != "" {
		query["tenant_id"] = c.tenantID
	}
	err := c.getJSON(ctx, "/v1/admin/ops/snapshot", query, &out)
	return out, err
}

func (c *Client) getJSON(ctx context.Context, path string, query map[string]string, out any) error {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return fmt.Errorf("parse URL: %w", err)
	}
	if len(query) > 0 {
		q := u.Query()
		for k, v := range query {
			if strings.TrimSpace(v) != "" {
				q.Set(k, v)
			}
		}
		u.RawQuery = q.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("perform request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
