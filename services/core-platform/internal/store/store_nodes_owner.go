package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/model"
)

func (s *SQLStore) CreateNode(ctx context.Context, in model.CreateNodeRequest) (model.Node, error) {
	caps := in.Capabilities
	if caps == nil {
		caps = []string{}
	}
	capsJSON, err := json.Marshal(caps)
	if err != nil {
		return model.Node{}, fmt.Errorf("marshal node capabilities: %w", err)
	}

	nodePublicKey := fmt.Sprintf("owner-managed-%d", time.Now().UTC().UnixNano())
	digest := sha256.Sum256([]byte(nodePublicKey))
	identityFingerprint := hex.EncodeToString(digest[:])

	row := s.db.QueryRowContext(
		ctx,
		`INSERT INTO nodes (
		   tenant_id, region, hostname, node_key_id, node_public_key, contract_version, agent_version,
		   capabilities, identity_fingerprint, status, last_seen_at, last_heartbeat_at
		 ) VALUES (
		   $1, $2, $3,
		   'owner-' || gen_random_uuid()::text,
		   $4, $5, $6,
		   $7::jsonb, $8, 'pending', NOW(), NULL
		 )
		 RETURNING id::text, tenant_id::text, region, hostname, node_key_id, node_public_key, contract_version, agent_version,
		           capabilities, identity_fingerprint, status, last_seen_at, last_heartbeat_at, created_at`,
		in.TenantID,
		in.Region,
		in.Hostname,
		nodePublicKey,
		in.ContractVersion,
		in.AgentVersion,
		string(capsJSON),
		identityFingerprint,
	)

	item, scanErr := scanNode(row.Scan)
	if scanErr != nil {
		return model.Node{}, mapDBError(scanErr)
	}
	return item, nil
}
