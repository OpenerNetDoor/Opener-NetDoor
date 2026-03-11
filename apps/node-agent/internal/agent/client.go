package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/opener-netdoor/opener-netdoor/apps/node-agent/internal/config"
)

type client struct {
	baseURL string
	http    *http.Client
}

type registerRequest struct {
	TenantID        string   `json:"tenant_id"`
	Region          string   `json:"region"`
	Hostname        string   `json:"hostname"`
	NodeKeyID       string   `json:"node_key_id"`
	NodePublicKey   string   `json:"node_public_key"`
	ContractVersion string   `json:"contract_version"`
	AgentVersion    string   `json:"agent_version"`
	Capabilities    []string `json:"capabilities"`
	Nonce           string   `json:"nonce"`
	SignedAt        int64    `json:"signed_at"`
	Signature       string   `json:"signature"`
}

type registerResponse struct {
	Node struct {
		ID string `json:"id"`
	} `json:"node"`
}

type heartbeatRequest struct {
	TenantID        string `json:"tenant_id"`
	NodeID          string `json:"node_id"`
	NodeKeyID       string `json:"node_key_id"`
	ContractVersion string `json:"contract_version"`
	AgentVersion    string `json:"agent_version"`
	Nonce           string `json:"nonce"`
	SignedAt        int64  `json:"signed_at"`
	Signature       string `json:"signature"`
}

func newClient(cfg config.Config) *client {
	return &client{
		baseURL: strings.TrimRight(cfg.CorePlatformURL, "/"),
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *client) register(ctx context.Context, cfg config.Config) (string, error) {
	now := time.Now().UTC().Unix()
	reqBody := registerRequest{
		TenantID:        cfg.TenantID,
		Region:          "local",
		Hostname:        cfg.NodeID,
		NodeKeyID:       cfg.NodeKeyID,
		NodePublicKey:   cfg.NodePublicKey,
		ContractVersion: cfg.ContractVersion,
		AgentVersion:    cfg.AgentVersion,
		Capabilities:    []string{"heartbeat.v1", "provisioning.v1"},
		Nonce:           cfg.RegistrationNonce,
		SignedAt:        now,
		Signature:       "local-dev-signature",
	}
	var out registerResponse
	if err := c.postJSON(ctx, "/internal/v1/nodes/register", reqBody, &out); err != nil {
		return "", err
	}
	if strings.TrimSpace(out.Node.ID) == "" {
		return "", fmt.Errorf("registration response missing node.id")
	}
	return out.Node.ID, nil
}

func (c *client) heartbeat(ctx context.Context, cfg config.Config, nodeID string, nonce string) error {
	reqBody := heartbeatRequest{
		TenantID:        cfg.TenantID,
		NodeID:          nodeID,
		NodeKeyID:       cfg.NodeKeyID,
		ContractVersion: cfg.ContractVersion,
		AgentVersion:    cfg.AgentVersion,
		Nonce:           nonce,
		SignedAt:        time.Now().UTC().Unix(),
		Signature:       "local-dev-signature",
	}
	return c.postJSON(ctx, "/internal/v1/nodes/heartbeat", reqBody, nil)
}

func (c *client) postJSON(ctx context.Context, path string, in any, out any) error {
	payload, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}
