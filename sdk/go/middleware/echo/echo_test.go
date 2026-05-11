package echomiddleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	aport "github.com/aporthq/aport-sdks-and-middlewares/sdk/go"
)

type fakeVerifier struct{}

func (f fakeVerifier) VerifyPolicy(ctx context.Context, agentID, policyID string, contextFields map[string]any, idempotencyKey string) (*aport.Decision, error) {
	return &aport.Decision{DecisionID: "dec_echo", Allow: agentID == "agent_echo"}, nil
}

func (f fakeVerifier) VerifyPolicyWithPassport(ctx context.Context, passport aport.PassportData, policyID string, contextFields map[string]any, idempotencyKey string) (*aport.Decision, error) {
	return &aport.Decision{DecisionID: "dec_echo", Allow: passport.AgentID != ""}, nil
}

func (f fakeVerifier) VerifyPolicyWithPolicyInBody(ctx context.Context, agentIDOrPassport any, policy aport.PolicyPack, contextFields map[string]any, idempotencyKey string) (*aport.Decision, error) {
	return &aport.Decision{DecisionID: "dec_echo", Allow: true}, nil
}

func (f fakeVerifier) GetPassportView(ctx context.Context, agentID string) (map[string]any, error) {
	return map[string]any{"agent_id": agentID}, nil
}

func TestEchoMiddlewareAllowsRequest(t *testing.T) {
	e := echo.New()
	e.Use(Middleware(Options{Client: fakeVerifier{}, PolicyID: "finance.payment.refund.v1"}))
	e.POST("/refunds", func(c echo.Context) error {
		decision := c.Get("aport_decision").(*aport.Decision)
		return c.JSON(http.StatusOK, map[string]any{
			"agent_id":    c.Get("aport_agent_id"),
			"decision_id": decision.DecisionID,
		})
	})

	req := httptest.NewRequest(http.MethodPost, "/refunds", strings.NewReader(`{"amount":20}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Passport-Id", "agent_echo")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "dec_echo") {
		t.Fatalf("response = %s", rec.Body.String())
	}
}
