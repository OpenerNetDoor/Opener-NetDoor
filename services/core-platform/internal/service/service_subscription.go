package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/model"
)

func (s *CoreService) GetUserSubscription(ctx context.Context, actor model.ActorPrincipal, q model.GetUserSubscriptionQuery) (model.UserSubscription, error) {
	q.TenantID = strings.TrimSpace(q.TenantID)
	q.UserID = strings.TrimSpace(q.UserID)
	q.Format = strings.ToLower(strings.TrimSpace(q.Format))
	if q.Format == "" {
		q.Format = "plain"
	}

	if q.TenantID == "" && actor.TenantID != "" {
		q.TenantID = actor.TenantID
	}
	if q.TenantID == "" || q.UserID == "" {
		return model.UserSubscription{}, &AppError{Status: 400, Code: "validation_error", Message: "tenant_id and user_id are required"}
	}
	if !actor.CanAccessTenant(q.TenantID) {
		return model.UserSubscription{}, &AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}

	user, err := s.store.GetUserByID(ctx, q.TenantID, q.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.UserSubscription{}, &AppError{Status: 404, Code: "user_not_found", Message: "user not found", Err: err}
		}
		return model.UserSubscription{}, mapStoreError("subscription_resolve_failed", err)
	}
	if strings.EqualFold(strings.TrimSpace(user.Status), "blocked") {
		return model.UserSubscription{}, &AppError{Status: 403, Code: "user_blocked", Message: "user is blocked"}
	}

	keys, err := s.store.ListAccessKeys(ctx, model.ListAccessKeysQuery{
		ListQuery: model.ListQuery{Limit: 10000, Offset: 0, Status: "active"},
		TenantID:  q.TenantID,
		UserID:    q.UserID,
	})
	if err != nil {
		return model.UserSubscription{}, mapStoreError("subscription_resolve_failed", err)
	}

	nodes, err := s.store.ListNodes(ctx, model.ListNodesQuery{
		ListQuery: model.ListQuery{Limit: 1000, Offset: 0},
		TenantID:  q.TenantID,
	})
	if err != nil {
		return model.UserSubscription{}, mapStoreError("subscription_resolve_failed", err)
	}

	runtimeByNode := make(map[string]model.NodeRuntime, len(nodes))
	rs, ok := s.store.(runtimeStore)
	if ok {
		for _, node := range nodes {
			runtime, runtimeErr := rs.GetNodeRuntime(ctx, q.TenantID, node.ID)
			if runtimeErr != nil {
				continue
			}
			runtimeByNode[node.ID] = runtime
		}
	}

	configs := make([]model.SubscriptionConfig, 0)
	seen := make(map[string]struct{})

	for _, key := range keys {
		keyType := strings.ToLower(strings.TrimSpace(key.KeyType))
		uuid := strings.TrimSpace(key.SecretRef)

		if keyType == "vless" && uuidRe.MatchString(uuid) {
			for _, node := range nodes {
				status := strings.ToLower(strings.TrimSpace(node.Status))
				if status == "revoked" || status == "offline" {
					continue
				}
				runtime, hasRuntime := runtimeByNode[node.ID]
				if !hasRuntime {
					continue
				}
				if strings.TrimSpace(runtime.RealityPublicKey) == "" || runtime.RealityPublicKey == "pending" {
					continue
				}
				if strings.ToLower(strings.TrimSpace(runtime.RuntimeProtocol)) != "vless_reality" {
					continue
				}
				if strings.ToLower(strings.TrimSpace(runtime.RuntimeStatus)) == "error" {
					continue
				}

				uri := buildVLESSRealityURI(node, runtime, uuid, key.ID)
				if _, exists := seen[uri]; exists {
					continue
				}
				seen[uri] = struct{}{}
				configs = append(configs, model.SubscriptionConfig{
					ServerID: node.ID,
					Hostname: node.Hostname,
					Region:   node.Region,
					Protocol: "vless_reality",
					Label:    subscriptionLabel(node, "VLESS Reality"),
					URI:      uri,
				})
			}
			continue
		}

		if strings.TrimSpace(key.ConnectionURI) != "" {
			uri := strings.TrimSpace(key.ConnectionURI)
			if _, exists := seen[uri]; exists {
				continue
			}
			seen[uri] = struct{}{}
			configs = append(configs, model.SubscriptionConfig{
				ServerID: "",
				Hostname: "",
				Region:   "",
				Protocol: keyType,
				Label:    strings.ToUpper(keyType),
				URI:      uri,
			})
		}
	}

	sort.SliceStable(configs, func(i, j int) bool {
		if configs[i].Label == configs[j].Label {
			return configs[i].URI < configs[j].URI
		}
		return configs[i].Label < configs[j].Label
	})

	payload := ""
	switch q.Format {
	case "json":
		blob, marshalErr := json.Marshal(configs)
		if marshalErr != nil {
			return model.UserSubscription{}, &AppError{Status: 500, Code: "subscription_resolve_failed", Message: "failed to encode subscription payload", Err: marshalErr}
		}
		payload = string(blob)
	case "plain":
		fallthrough
	default:
		lines := make([]string, 0, len(configs))
		for _, cfg := range configs {
			lines = append(lines, cfg.URI)
		}
		payload = strings.Join(lines, "\n")
		q.Format = "plain"
	}

	display := subscriptionDisplayName(user)
	subscriptionURL := fmt.Sprintf("%s/%s/%s/#%s", strings.TrimRight(s.opts.PublicBaseURL, "/"), s.opts.SubscriptionAccessSecret, q.UserID, url.QueryEscape(display))

	return model.UserSubscription{
		TenantID:        q.TenantID,
		UserID:          q.UserID,
		GeneratedAt:     time.Now().UTC(),
		Format:          q.Format,
		SubscriptionURL: subscriptionURL,
		Payload:         payload,
		ConfigCount:     len(configs),
		Configs:         configs,
	}, nil
}

func subscriptionDisplayName(user model.User) string {
	noteName := ""
	if strings.TrimSpace(user.Note) != "" {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(user.Note), &parsed); err == nil {
			if v, ok := parsed["display_name"].(string); ok {
				noteName = strings.TrimSpace(v)
			}
		}
	}
	if noteName != "" {
		return noteName
	}
	email := strings.TrimSpace(user.Email)
	if email == "" {
		return user.ID
	}
	parts := strings.Split(email, "@")
	if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
		return strings.TrimSpace(parts[0])
	}
	return email
}

func subscriptionLabel(node model.Node, protocol string) string {
	server := strings.TrimSpace(node.Region)
	if server == "" {
		server = strings.TrimSpace(node.Hostname)
	}
	if server == "" {
		server = strings.TrimSpace(node.ID)
	}
	return fmt.Sprintf("%s • %s", server, protocol)
}
