package fibermiddleware

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	aport "github.com/aporthq/aport-sdks-and-middlewares/sdk/go"
)

type fakeVerifier struct{}

func (f fakeVerifier) VerifyPolicy(ctx context.Context, agentID, policyID string, contextFields map[string]any, idempotencyKey string) (*aport.Decision, error) {
	return &aport.Decision{DecisionID: "dec_fiber", Allow: agentID == "agent_fiber"}, nil
}

func (f fakeVerifier) VerifyPolicyWithPassport(ctx context.Context, passport aport.PassportData, policyID string, contextFields map[string]any, idempotencyKey string) (*aport.Decision, error) {
	return &aport.Decision{DecisionID: "dec_fiber", Allow: passport.AgentID != ""}, nil
}

func (f fakeVerifier) VerifyPolicyWithPolicyInBody(ctx context.Context, agentIDOrPassport any, policy aport.PolicyPack, contextFields map[string]any, idempotencyKey string) (*aport.Decision, error) {
	return &aport.Decision{DecisionID: "dec_fiber", Allow: true}, nil
}

func (f fakeVerifier) GetPassportView(ctx context.Context, agentID string) (map[string]any, error) {
	return map[string]any{"agent_id": agentID}, nil
}

func TestFiberMiddlewareAllowsRequest(t *testing.T) {
	app := fiber.New()
	app.Use(Middleware(Options{Client: fakeVerifier{}, PolicyID: "finance.payment.refund.v1"}))
	app.Post("/refunds", func(c *fiber.Ctx) error {
		decision := c.Locals("aport_decision").(*aport.Decision)
		return c.JSON(fiber.Map{
			"agent_id":    c.Locals("aport_agent_id"),
			"decision_id": decision.DecisionID,
		})
	})

	req, err := http.NewRequest(http.MethodPost, "/refunds", strings.NewReader(`{"amount":20}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Passport-Id", "agent_fiber")

	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d body = %s", res.StatusCode, body)
	}
	if !strings.Contains(string(body), "dec_fiber") {
		t.Fatalf("response = %s", body)
	}
}
