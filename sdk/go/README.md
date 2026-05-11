# APort Go SDK

Official Go SDK for the APort API with Gin, Echo, and Fiber middleware.

## Install

```bash
go get github.com/aporthq/aport-sdks-and-middlewares/sdk/go
go get github.com/aporthq/aport-sdks-and-middlewares/sdk/go/middleware/gin
go get github.com/aporthq/aport-sdks-and-middlewares/sdk/go/middleware/echo
go get github.com/aporthq/aport-sdks-and-middlewares/sdk/go/middleware/fiber
```

## Client

```go
package main

import (
	"context"
	"log"
	"time"

	aport "github.com/aporthq/aport-sdks-and-middlewares/sdk/go"
)

func main() {
	client := aport.NewClient(aport.Options{
		BaseURL: "https://api.aport.io",
		APIKey:  "your-api-key",
		Timeout: 800 * time.Millisecond,
	})

	decision, err := client.VerifyPolicy(
		context.Background(),
		"your-agent-id",
		"finance.payment.refund.v1",
		map[string]any{
			"amount":   1000,
			"currency": "USD",
			"order_id": "order_123",
		},
		"refund-order-123",
	)
	if err != nil {
		log.Fatal(err)
	}
	if decision.Allow {
		log.Printf("allowed: %s", decision.DecisionID)
	}
}
```

The client supports:

- `VerifyPolicy`
- `VerifyPolicyWithPassport`
- `VerifyPolicyWithPolicyInBody`
- `GetDecisionToken`
- `ValidateDecisionToken`
- `ValidateDecisionTokenLocal`
- `GetPassportView`
- `GetJWKS`

Errors are returned as `*aport.AportError` with HTTP status, decision ID, denial reasons, server timing, and raw response text when available.

## Gin Middleware

```go
router := gin.Default()
router.Use(ginaport.RequireRefundPolicy(ginaport.Options{
	APIKey: "your-api-key",
}))
```

## Echo Middleware

```go
e := echo.New()
e.Use(echoaport.RequireRefundPolicy(echoaport.Options{APIKey: "your-api-key"}))
```

## Fiber Middleware

```go
app := fiber.New()
app.Use(fiberaport.RequireRefundPolicy(fiberaport.Options{APIKey: "your-api-key"}))
```

All middleware packages read the agent ID from `X-Agent-Passport-Id`, `X-Agent-Id`, or `body.passport.agent_id`. They attach the policy result to the framework context as `aport_decision` and deny requests with `403` when APort returns `allow=false`.

## Local Passport and Inline Policy

Send `passport` and/or `policy` in a JSON request body to use local passport mode or `IN_BODY` policy evaluation:

```json
{
  "passport": {
    "agent_id": "agent_demo",
    "claims": { "role": "buyer" }
  },
  "policy": {
    "id": "custom.policy.v1",
    "requires_capabilities": ["payments.charge"]
  },
  "amount": 1000,
  "currency": "USD"
}
```

## Examples

```bash
go run ./examples/refund
go run ./examples/gin
```

## Development

```bash
go mod tidy
go test ./...
```
