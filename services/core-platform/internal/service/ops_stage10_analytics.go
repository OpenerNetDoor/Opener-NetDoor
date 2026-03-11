package service

import (
	"context"
	"strings"
	"time"

	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/model"
)

func (s *CoreService) GetOpsAnalytics(ctx context.Context, actor model.ActorPrincipal, tenantID string) (model.OpsAnalytics, error) {
	tenantID = strings.TrimSpace(tenantID)
	if !actor.IsPlatformAdmin() {
		if tenantID == "" {
			tenantID = actor.TenantID
		}
		if !actor.CanAccessTenant(tenantID) {
			return model.OpsAnalytics{}, &AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
		}
	}
	if tenantID != "" && !actor.CanAccessTenant(tenantID) {
		return model.OpsAnalytics{}, &AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}

	now := time.Now().UTC()
	since24h := now.Add(-24 * time.Hour)
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	seriesStart := dayStart.AddDate(0, 0, -6)
	seriesUntil := dayStart.AddDate(0, 0, 1)

	totalUsers, err := s.store.CountUsersByStatus(ctx, tenantID, "")
	if err != nil {
		return model.OpsAnalytics{}, &AppError{Status: 500, Code: "ops_analytics_failed", Message: "failed to load total users", Err: err}
	}
	activeUsers, err := s.store.CountUsersByStatus(ctx, tenantID, "active")
	if err != nil {
		return model.OpsAnalytics{}, &AppError{Status: 500, Code: "ops_analytics_failed", Message: "failed to load active users", Err: err}
	}
	activeKeys, err := s.store.CountAccessKeysByStatus(ctx, tenantID, "active")
	if err != nil {
		return model.OpsAnalytics{}, &AppError{Status: 500, Code: "ops_analytics_failed", Message: "failed to load active keys", Err: err}
	}
	nodeStatus, err := s.store.ListNodeStatusCounts(ctx, tenantID)
	if err != nil {
		return model.OpsAnalytics{}, &AppError{Status: 500, Code: "ops_analytics_failed", Message: "failed to load server status counters", Err: err}
	}
	onlineServers := 0
	for _, item := range nodeStatus {
		if item.Status == "active" {
			onlineServers = item.Count
			break
		}
	}
	traffic24h, err := s.store.GetTrafficUsageTotalBetween(ctx, tenantID, since24h, now)
	if err != nil {
		return model.OpsAnalytics{}, &AppError{Status: 500, Code: "ops_analytics_failed", Message: "failed to load 24h traffic", Err: err}
	}

	trafficRaw, err := s.store.ListTrafficUsageSeries(ctx, tenantID, seriesStart, seriesUntil)
	if err != nil {
		return model.OpsAnalytics{}, &AppError{Status: 500, Code: "ops_analytics_failed", Message: "failed to load traffic history", Err: err}
	}
	trafficHistory := make([]model.OpsTrafficPoint, 0)
	if len(trafficRaw) > 0 {
		byDay := make(map[string]model.OpsTrafficPoint, len(trafficRaw))
		for _, point := range trafficRaw {
			k := point.TsHour.UTC().Format("2006-01-02")
			byDay[k] = point
		}
		for i := 0; i < 7; i++ {
			day := seriesStart.AddDate(0, 0, i)
			k := day.Format("2006-01-02")
			if point, ok := byDay[k]; ok {
				trafficHistory = append(trafficHistory, point)
				continue
			}
			trafficHistory = append(trafficHistory, model.OpsTrafficPoint{TsHour: day})
		}
	}

	growthRaw, err := s.store.ListUserGrowthByDay(ctx, tenantID, seriesStart, seriesUntil)
	if err != nil {
		return model.OpsAnalytics{}, &AppError{Status: 500, Code: "ops_analytics_failed", Message: "failed to load user growth history", Err: err}
	}
	growth := make([]model.OpsUserGrowthPoint, 0)
	if totalUsers > 0 || len(growthRaw) > 0 {
		baseline, baseErr := s.store.CountUsersCreatedBefore(ctx, tenantID, seriesStart)
		if baseErr != nil {
			return model.OpsAnalytics{}, &AppError{Status: 500, Code: "ops_analytics_failed", Message: "failed to load user growth baseline", Err: baseErr}
		}
		dayToNew := make(map[string]int, len(growthRaw))
		for _, point := range growthRaw {
			dayToNew[point.Day] = point.NewUsers
		}
		running := baseline
		for i := 0; i < 7; i++ {
			day := seriesStart.AddDate(0, 0, i).Format("2006-01-02")
			newUsers := dayToNew[day]
			running += newUsers
			growth = append(growth, model.OpsUserGrowthPoint{
				Day:        day,
				NewUsers:   newUsers,
				TotalUsers: running,
			})
		}
	}

	protocolUsage, err := s.store.ListProtocolUsageBetween(ctx, tenantID, since24h, now)
	if err != nil {
		return model.OpsAnalytics{}, &AppError{Status: 500, Code: "ops_analytics_failed", Message: "failed to load protocol usage", Err: err}
	}
	for i := range protocolUsage {
		protocolUsage[i].Protocol = strings.TrimSpace(protocolUsage[i].Protocol)
	}

	topServers, err := s.store.ListTopServersByTraffic(ctx, tenantID, since24h, now, 5)
	if err != nil {
		return model.OpsAnalytics{}, &AppError{Status: 500, Code: "ops_analytics_failed", Message: "failed to load top servers", Err: err}
	}
	var maxBytes int64
	for _, item := range topServers {
		if item.BytesTotal > maxBytes {
			maxBytes = item.BytesTotal
		}
	}
	if maxBytes > 0 {
		for i := range topServers {
			topServers[i].LoadPercent = int((topServers[i].BytesTotal * 100) / maxBytes)
			if topServers[i].LoadPercent == 0 && topServers[i].BytesTotal > 0 {
				topServers[i].LoadPercent = 1
			}
		}
	}

	return model.OpsAnalytics{
		TenantID:         tenantID,
		GeneratedAt:      now,
		TotalUsers:       totalUsers,
		ActiveUsers:      activeUsers,
		ActiveKeys:       activeKeys,
		OnlineServers:    onlineServers,
		TrafficBytes24h:  traffic24h,
		TrafficHistory7d: trafficHistory,
		UserGrowth7d:     growth,
		ProtocolUsage24h: protocolUsage,
		TopServersByLoad: topServers,
	}, nil
}
