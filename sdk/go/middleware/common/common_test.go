package common

import (
	"context"
	"net/http"
	"testing"

	aport "github.com/aporthq/aport-sdks-and-middlewares/sdk/go"
)

type fakeVerifier struct {
	decision *aport.Decision
	agentID  string
	policyID string
	context  map[string]any
}

func (f *fakeVerifier) VerifyPolicy(ctx context.Context, agentID, policyID string, contextFields map[string]any, idempotencyKey string) (*aport.Decision, error) {
	f.agentID = agentID
	f.policyID = policyID
	f.context = contextFields
	return f.decision, nil
}

func (f *fakeVerifier) VerifyPolicyWithPassport(ctx context.Context, passport aport.PassportData, policyID string, contextFields map[string]any, idempotencyKey string) (*aport.Decision, error) {
	f.agentID = passport.AgentID
	f.policyID = policyID
	f.context = contextFields
	return f.decision, nil
}

func (f *fakeVerifier) VerifyPolicyWithPolicyInBody(ctx context.Context, agentIDOrPassport any, policy aport.PolicyPack, contextFields map[string]any, idempotencyKey string) (*aport.Decision, error) {
	f.policyID = policy.ID
	f.context = contextFields
	switch value := agentIDOrPassport.(type) {
	case string:
		f.agentID = value
	case aport.PassportData:
		f.agentID = value.AgentID
	}
	return f.decision, nil
}

func (f *fakeVerifier) GetPassportView(ctx context.Context, agentID string) (map[string]any, error) {
	f.agentID = agentID
	return map[string]any{"agent_id": agentID}, nil
}

func TestEvaluateAllowsAndAddsContext(t *testing.T) {
	fake := &fakeVerifier{decision: &aport.Decision{DecisionID: "dec_1", Allow: true}}

	result, errResponse := Evaluate(Options{
		Client:   fake,
		PolicyID: "finance.payment.refund.v1",
		Context:  map[string]any{"route": "refunds"},
	}, RequestData{
		Path: "/refunds",
		Headers: map[string]string{
			"X-Agent-Passport-Id": "agent_123",
		},
		Body: map[string]any{"amount": 12},
	})
	if errResponse != nil {
		t.Fatalf("unexpected error: %+v", errResponse)
	}
	if result.AgentID != "agent_123" {
		t.Fatalf("agent_id = %s", result.AgentID)
	}
	if fake.context["amount"].(int) != 12 {
		t.Fatalf("amount = %v", fake.context["amount"])
	}
	if fake.context["route"] != "refunds" {
		t.Fatalf("route context missing")
	}
}

func TestEvaluateDeniesPolicyViolation(t *testing.T) {
	fake := &fakeVerifier{decision: &aport.Decision{
		DecisionID: "dec_deny",
		Allow:      false,
		Reasons:    []aport.Reason{{Code: "DENIED", Message: "no"}},
	}}

	_, errResponse := Evaluate(Options{
		Client:   fake,
		PolicyID: "finance.payment.refund.v1",
	}, RequestData{
		Path:    "/refunds",
		Headers: map[string]string{"X-Agent-Id": "agent_123"},
	})
	if errResponse == nil {
		t.Fatalf("expected error response")
	}
	if errResponse.Status != http.StatusForbidden {
		t.Fatalf("status = %d", errResponse.Status)
	}
	if errResponse.DecisionID != "dec_deny" {
		t.Fatalf("decision_id = %s", errResponse.DecisionID)
	}
}
