package main

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"

	aport "github.com/aporthq/aport-sdks-and-middlewares/sdk/go"
	ginaport "github.com/aporthq/aport-sdks-and-middlewares/sdk/go/middleware/gin"
)

type demoClient struct{}

func (demoClient) VerifyPolicy(ctx context.Context, agentID, policyID string, contextFields map[string]any, idempotencyKey string) (*aport.Decision, error) {
	return &aport.Decision{DecisionID: "dec_demo", Allow: agentID != ""}, nil
}

func (demoClient) VerifyPolicyWithPassport(ctx context.Context, passport aport.PassportData, policyID string, contextFields map[string]any, idempotencyKey string) (*aport.Decision, error) {
	return &aport.Decision{DecisionID: "dec_demo", Allow: passport.AgentID != ""}, nil
}

func (demoClient) VerifyPolicyWithPolicyInBody(ctx context.Context, agentIDOrPassport any, policy aport.PolicyPack, contextFields map[string]any, idempotencyKey string) (*aport.Decision, error) {
	return &aport.Decision{DecisionID: "dec_demo", Allow: true}, nil
}

func (demoClient) GetPassportView(ctx context.Context, agentID string) (map[string]any, error) {
	return map[string]any{"agent_id": agentID}, nil
}

func main() {
	router := gin.Default()
	router.Use(ginaport.RequireRefundPolicy(ginaport.Options{Client: demoClient{}}))
	router.POST("/refunds", func(c *gin.Context) {
		decision := c.MustGet("aport_decision").(*aport.Decision)
		c.JSON(200, gin.H{
			"agent_id":    c.GetString("aport_agent_id"),
			"decision_id": decision.DecisionID,
		})
	})

	log.Fatal(router.Run(":8080"))
}
