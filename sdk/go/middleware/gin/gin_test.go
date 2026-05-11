package ginmiddleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	aport "github.com/aporthq/aport-sdks-and-middlewares/sdk/go"
)

type fakeVerifier struct{}

func (f fakeVerifier) VerifyPolicy(ctx context.Context, agentID, policyID string, contextFields map[string]any, idempotencyKey string) (*aport.Decision, error) {
	return &aport.Decision{DecisionID: "dec_gin", Allow: agentID == "agent_gin"}, nil
}

func (f fakeVerifier) VerifyPolicyWithPassport(ctx context.Context, passport aport.PassportData, policyID string, contextFields map[string]any, idempotencyKey string) (*aport.Decision, error) {
	return &aport.Decision{DecisionID: "dec_gin", Allow: passport.AgentID != ""}, nil
}

func (f fakeVerifier) VerifyPolicyWithPolicyInBody(ctx context.Context, agentIDOrPassport any, policy aport.PolicyPack, contextFields map[string]any, idempotencyKey string) (*aport.Decision, error) {
	return &aport.Decision{DecisionID: "dec_gin", Allow: true}, nil
}

func (f fakeVerifier) GetPassportView(ctx context.Context, agentID string) (map[string]any, error) {
	return map[string]any{"agent_id": agentID}, nil
}

func TestGinMiddlewareAllowsRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(Middleware(Options{Client: fakeVerifier{}, PolicyID: "finance.payment.refund.v1"}))
	router.POST("/refunds", func(c *gin.Context) {
		decisionValue, exists := c.Get("aport_decision")
		if !exists {
			t.Fatalf("decision missing from gin context")
		}
		decision := decisionValue.(*aport.Decision)
		c.JSON(http.StatusOK, gin.H{
			"agent_id":    c.GetString("aport_agent_id"),
			"decision_id": decision.DecisionID,
		})
	})

	req := httptest.NewRequest(http.MethodPost, "/refunds", strings.NewReader(`{"amount":20}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Passport-Id", "agent_gin")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "dec_gin") {
		t.Fatalf("response = %s", rec.Body.String())
	}
}
