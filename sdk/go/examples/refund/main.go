package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	aport "github.com/aporthq/aport-sdks-and-middlewares/sdk/go"
)

func main() {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"decision": map[string]any{
				"decision_id": "dec_example",
				"allow":       true,
			},
		})
	}))
	defer api.Close()

	client := aport.NewClient(aport.Options{BaseURL: api.URL})
	decision, err := client.VerifyPolicy(
		context.Background(),
		"agent_demo",
		"finance.payment.refund.v1",
		map[string]any{
			"amount":   1000,
			"currency": "USD",
			"order_id": "order_123",
		},
		"refund-order-123",
	)
	if err != nil {
		panic(err)
	}

	fmt.Printf("allowed=%t decision_id=%s\n", decision.Allow, decision.DecisionID)
}
