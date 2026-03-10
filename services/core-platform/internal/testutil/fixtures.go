package testutil

import "github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/model"

func PlatformAdminActor() model.ActorPrincipal {
	return model.ActorPrincipal{
		Subject: "admin-platform-1",
		Scopes:  []string{"admin:read", "admin:write", "platform:admin"},
	}
}

func TenantActor(tenantID string) model.ActorPrincipal {
	return model.ActorPrincipal{
		Subject:  "admin-tenant-1",
		TenantID: tenantID,
		Scopes:   []string{"admin:read", "admin:write"},
	}
}
