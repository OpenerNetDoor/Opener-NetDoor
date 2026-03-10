package proxy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL *url.URL
	http    *http.Client
}

func New(base string) (*Client, error) {
	u, err := url.Parse(base)
	if err != nil {
		return nil, err
	}
	return &Client{
		baseURL: u,
		http:    &http.Client{Timeout: 15 * time.Second},
	}, nil
}

func (c *Client) Ready(ctx context.Context) error {
	urlCopy := *c.baseURL
	urlCopy.Path = "/internal/ready"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlCopy.String(), nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("upstream returned %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) Forward(w http.ResponseWriter, r *http.Request, targetPath string, actorSub string, actorScopes []string, actorTenantID string) {
	urlCopy := *c.baseURL
	urlCopy.Path = targetPath
	urlCopy.RawQuery = r.URL.RawQuery

	var body io.Reader
	if r.Body != nil {
		body = r.Body
	}

	req, err := http.NewRequestWithContext(r.Context(), r.Method, urlCopy.String(), body)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":{"code":"proxy_build_failed","message":"failed to build upstream request"}}`))
		return
	}

	req.Header.Set("Content-Type", r.Header.Get("Content-Type"))
	req.Header.Set("X-Request-ID", r.Header.Get("X-Request-ID"))
	req.Header.Set("X-Actor-Sub", actorSub)
	req.Header.Set("X-Actor-Scopes", strings.Join(actorScopes, ","))
	req.Header.Set("X-Actor-Tenant-ID", actorTenantID)

	resp, err := c.http.Do(req)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":{"code":"proxy_upstream_failed","message":"upstream unavailable"}}`))
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		if strings.EqualFold(k, "X-Request-ID") {
			continue
		}
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}
