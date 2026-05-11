package aport

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVerifyPolicyBuildsCanonicalRequest(t *testing.T) {
	var received map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/verify/policy/finance.payment.refund.v1" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("missing authorization header")
		}
		if r.Header.Get("Idempotency-Key") != "idem-1" {
			t.Fatalf("missing idempotency header")
		}
		if r.Header.Get("User-Agent") != userAgent {
			t.Fatalf("user agent = %s", r.Header.Get("User-Agent"))
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatal(err)
		}

		w.Header().Set("Server-Timing", "app;dur=12")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"decision": map[string]any{
				"decision_id": "dec_123",
				"allow":       true,
			},
		})
	}))
	defer server.Close()

	client := NewClient(Options{BaseURL: server.URL, APIKey: "test-key"})
	decision, err := client.VerifyPolicy(
		context.Background(),
		"agent_123",
		"finance.payment.refund.v1",
		map[string]any{"amount": 1000, "currency": "USD"},
		"idem-1",
	)
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Allow || decision.DecisionID != "dec_123" {
		t.Fatalf("unexpected decision: %+v", decision)
	}
	if decision.Meta["serverTiming"] != "app;dur=12" {
		t.Fatalf("server timing was not attached: %+v", decision.Meta)
	}

	contextMap := received["context"].(map[string]any)
	if contextMap["agent_id"] != "agent_123" {
		t.Fatalf("agent_id = %v", contextMap["agent_id"])
	}
	if contextMap["policy_id"] != "finance.payment.refund.v1" {
		t.Fatalf("policy_id = %v", contextMap["policy_id"])
	}
	if contextMap["idempotency_key"] != "idem-1" {
		t.Fatalf("idempotency_key = %v", contextMap["idempotency_key"])
	}
	if contextMap["amount"].(float64) != 1000 {
		t.Fatalf("amount = %v", contextMap["amount"])
	}
}

func TestVerifyPolicyWithPolicyInBody(t *testing.T) {
	var received VerificationRequestBody

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/verify/policy/IN_BODY" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(Decision{DecisionID: "dec_inline", Allow: true})
	}))
	defer server.Close()

	client := NewClient(Options{BaseURL: server.URL})
	passport := PassportData{AgentID: "agent_passport", Claims: map[string]any{"role": "buyer"}}
	policy := PolicyPack{ID: "custom.policy.v1", RequiresCapabilities: []string{"payments.charge"}}

	decision, err := client.VerifyPolicyWithPolicyInBody(context.Background(), passport, policy, map[string]any{"amount": 42}, "")
	if err != nil {
		t.Fatal(err)
	}
	if decision.DecisionID != "dec_inline" {
		t.Fatalf("decision_id = %s", decision.DecisionID)
	}
	if received.Context["agent_id"] != "agent_passport" {
		t.Fatalf("agent_id = %v", received.Context["agent_id"])
	}
	if received.Context["policy_id"] != "custom.policy.v1" {
		t.Fatalf("policy_id = %v", received.Context["policy_id"])
	}
	if received.Passport == nil {
		t.Fatalf("passport was not included")
	}
	if received.Policy == nil {
		t.Fatalf("policy was not included")
	}
}

func TestStructuredAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"decision_id": "dec_deny",
			"reasons": []map[string]any{{
				"code":    "LIMIT_EXCEEDED",
				"message": "amount is too high",
			}},
		})
	}))
	defer server.Close()

	client := NewClient(Options{BaseURL: server.URL})
	_, err := client.VerifyPolicy(context.Background(), "agent_123", "finance.payment.refund.v1", nil, "")

	var aportErr *AportError
	if !errors.As(err, &aportErr) {
		t.Fatalf("expected AportError, got %T", err)
	}
	if aportErr.Status != http.StatusForbidden {
		t.Fatalf("status = %d", aportErr.Status)
	}
	if aportErr.DecisionID != "dec_deny" {
		t.Fatalf("decision_id = %s", aportErr.DecisionID)
	}
	if aportErr.Reasons[0].Code != "LIMIT_EXCEEDED" {
		t.Fatalf("reason = %+v", aportErr.Reasons[0])
	}
}

func TestDecisionTokenAndJWKSCaching(t *testing.T) {
	jwksCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/verify/token/finance.payment.refund.v1":
			_ = json.NewEncoder(w).Encode(map[string]string{"token": "token-123"})
		case "/api/verify/token/validate":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"decision": map[string]any{"decision_id": "dec_token", "allow": true},
			})
		case "/jwks.json":
			jwksCalls++
			_ = json.NewEncoder(w).Encode(JWKS{Keys: []JWK{{Kty: "RSA", Kid: "kid-1"}}})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(Options{BaseURL: server.URL})
	token, err := client.GetDecisionToken(context.Background(), "agent_123", "finance.payment.refund.v1", map[string]any{"amount": 20})
	if err != nil {
		t.Fatal(err)
	}
	if token != "token-123" {
		t.Fatalf("token = %s", token)
	}

	for i := 0; i < 2; i++ {
		decision, err := client.ValidateDecisionTokenLocal(context.Background(), token)
		if err != nil {
			t.Fatal(err)
		}
		if !decision.Allow {
			t.Fatalf("decision denied")
		}
	}
	if jwksCalls != 1 {
		t.Fatalf("jwks calls = %d", jwksCalls)
	}
}
