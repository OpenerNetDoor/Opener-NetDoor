package client

import "time"

type Config struct {
	BaseURL   string
	Token     string
	Timeout   time.Duration
	TenantID  string
	UserAgent string
}

type HealthResponse struct {
	Status string `json:"status"`
}

type AuditLogRecord struct {
	ID         string         `json:"id"`
	TenantID   string         `json:"tenant_id,omitempty"`
	ActorType  string         `json:"actor_type"`
	Action     string         `json:"action"`
	TargetType string         `json:"target_type,omitempty"`
	TargetID   string         `json:"target_id,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	CreatedAt  string         `json:"created_at"`
}

type AuditLogListResponse struct {
	Items  []AuditLogRecord `json:"items"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
}

type OpsNodeStatusCount struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

type OpsSnapshot struct {
	TenantID             string               `json:"tenant_id,omitempty"`
	GeneratedAt          string               `json:"generated_at"`
	NodeStatus           []OpsNodeStatusCount `json:"node_status"`
	ActiveCertificates   int                  `json:"active_certificates"`
	ExpiringCertificates int                  `json:"expiring_certificates_24h"`
	TrafficBytes24h      int64                `json:"traffic_bytes_24h"`
	ReplayRejected24h    int                  `json:"replay_rejected_24h"`
	InvalidSignature24h  int                  `json:"invalid_signature_24h"`
}
