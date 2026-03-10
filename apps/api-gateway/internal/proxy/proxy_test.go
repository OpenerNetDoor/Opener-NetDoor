package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestForwardSetsActorHeaders(t *testing.T) {
	var gotActorSub string
	var gotActorScopes string
	var gotActorTenant string
	var gotRequestID string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotActorSub = r.Header.Get("X-Actor-Sub")
		gotActorScopes = r.Header.Get("X-Actor-Scopes")
		gotActorTenant = r.Header.Get("X-Actor-Tenant-ID")
		gotRequestID = r.Header.Get("X-Request-ID")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	px, err := New(upstream.URL)
	if err != nil {
		t.Fatalf("new proxy: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/users?tenant_id=t1", nil)
	req.Header.Set("X-Request-ID", "rid-123")

	px.Forward(rr, req, "/internal/v1/users", "admin-1", []string{"admin:read", "admin:write"}, "tenant-1")

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if gotActorSub != "admin-1" {
		t.Fatalf("unexpected X-Actor-Sub: %q", gotActorSub)
	}
	if gotActorScopes != "admin:read,admin:write" {
		t.Fatalf("unexpected X-Actor-Scopes: %q", gotActorScopes)
	}
	if gotActorTenant != "tenant-1" {
		t.Fatalf("unexpected X-Actor-Tenant-ID: %q", gotActorTenant)
	}
	if gotRequestID != "rid-123" {
		t.Fatalf("unexpected X-Request-ID: %q", gotRequestID)
	}
}

func TestForwardSkipsUpstreamRequestID(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-ID", "upstream-rid")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	px, err := New(upstream.URL)
	if err != nil {
		t.Fatalf("new proxy: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/users?tenant_id=t1", nil)
	req.Header.Set("X-Request-ID", "gateway-rid")

	px.Forward(rr, req, "/internal/v1/users", "admin-1", []string{"admin:read"}, "tenant-1")

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if values := rr.Header()["X-Request-Id"]; len(values) != 0 {
		t.Fatalf("expected no upstream request id copied, got %v", values)
	}
	body, _ := io.ReadAll(rr.Body)
	if string(body) == "" {
		t.Fatal("expected body from upstream")
	}
}
