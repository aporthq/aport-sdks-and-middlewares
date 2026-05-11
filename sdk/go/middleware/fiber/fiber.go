package fibermiddleware

import (
	"github.com/gofiber/fiber/v2"

	"github.com/aporthq/aport-sdks-and-middlewares/sdk/go/middleware/common"
)

type Options = common.Options

func Middleware(opts Options) fiber.Handler {
	return func(c *fiber.Ctx) error {
		body := map[string]any{}
		if len(c.Body()) > 0 {
			_ = c.BodyParser(&body)
		}

		headers := map[string]string{}
		c.Request().Header.VisitAll(func(key, value []byte) {
			headers[string(key)] = string(value)
		})

		result, errResponse := common.Evaluate(opts, common.RequestData{
			Path:           c.Path(),
			Headers:        headers,
			Body:           body,
			RequestContext: c.UserContext(),
		})
		if errResponse != nil {
			return c.Status(errResponse.Status).JSON(errResponse.Body())
		}
		if result == nil || result.Skipped {
			return c.Next()
		}

		c.Locals("aport_agent_id", result.AgentID)
		c.Locals("aport_decision", result.Decision)
		c.Locals("aport_passport", result.Passport)
		c.Locals("aport_passport_view", result.PassportView)
		return c.Next()
	}
}

func RequirePolicy(policyID string, opts Options) fiber.Handler {
	opts.PolicyID = policyID
	return Middleware(opts)
}

func RequireRefundPolicy(opts Options) fiber.Handler {
	return RequirePolicy("finance.payment.refund.v1", opts)
}

func RequireDataExportPolicy(opts Options) fiber.Handler {
	return RequirePolicy("data.export.create.v1", opts)
}

func RequireMessagingPolicy(opts Options) fiber.Handler {
	return RequirePolicy("messaging.message.send.v1", opts)
}

func RequireRepositoryPolicy(opts Options) fiber.Handler {
	return RequirePolicy("code.repository.merge.v1", opts)
}
