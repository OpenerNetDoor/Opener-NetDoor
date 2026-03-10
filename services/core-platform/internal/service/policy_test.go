package service

import (
	"testing"

	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/model"
)

func TestValidatePolicyValues(t *testing.T) {
	negQuota := int64(-1)
	if err := validatePolicyValues(&negQuota, nil, nil); err == nil {
		t.Fatal("expected error for negative quota")
	}

	zeroLimit := 0
	if err := validatePolicyValues(nil, &zeroLimit, nil); err == nil {
		t.Fatal("expected error for zero device limit")
	}

	zeroTTL := 0
	if err := validatePolicyValues(nil, nil, &zeroTTL); err == nil {
		t.Fatal("expected error for zero ttl")
	}

	quota := int64(1024)
	limit := 2
	ttl := 3600
	if err := validatePolicyValues(&quota, &limit, &ttl); err != nil {
		t.Fatalf("expected valid values, got %v", err)
	}
}

func TestEnforceQuota(t *testing.T) {
	svc := &CoreService{}
	quota := int64(100)
	if err := svc.enforceQuota(model.EffectivePolicy{TrafficQuotaBytes: &quota, UsageBytes: 100}); err == nil {
		t.Fatal("expected quota exceeded error")
	}
	if err := svc.enforceQuota(model.EffectivePolicy{TrafficQuotaBytes: &quota, UsageBytes: 99}); err != nil {
		t.Fatalf("expected no quota error below limit, got %v", err)
	}
	if err := svc.enforceQuota(model.EffectivePolicy{TrafficQuotaBytes: nil, UsageBytes: 999999}); err != nil {
		t.Fatalf("expected unlimited policy, got %v", err)
	}
}
