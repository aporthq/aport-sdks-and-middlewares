package echomiddleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/aporthq/aport-sdks-and-middlewares/sdk/go/middleware/common"
)

type Options = common.Options

func Middleware(opts Options) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			body := map[string]any{}
			if c.Request().Body != nil && c.Request().ContentLength != 0 {
				raw, _ := io.ReadAll(c.Request().Body)
				c.Request().Body = io.NopCloser(bytes.NewReader(raw))
				_ = json.Unmarshal(raw, &body)
			}

			result, errResponse := common.Evaluate(opts, common.RequestData{
				Path:           c.Request().URL.Path,
				Headers:        headersFromHTTP(c.Request().Header),
				Body:           body,
				RequestContext: c.Request().Context(),
			})
			if errResponse != nil {
				return c.JSON(errResponse.Status, errResponse.Body())
			}
			if result == nil || result.Skipped {
				return next(c)
			}

			c.Set("aport_agent_id", result.AgentID)
			c.Set("aport_decision", result.Decision)
			c.Set("aport_passport", result.Passport)
			c.Set("aport_passport_view", result.PassportView)
			return next(c)
		}
	}
}

func RequirePolicy(policyID string, opts Options) echo.MiddlewareFunc {
	opts.PolicyID = policyID
	return Middleware(opts)
}

func RequireRefundPolicy(opts Options) echo.MiddlewareFunc {
	return RequirePolicy("finance.payment.refund.v1", opts)
}

func RequireDataExportPolicy(opts Options) echo.MiddlewareFunc {
	return RequirePolicy("data.export.create.v1", opts)
}

func RequireMessagingPolicy(opts Options) echo.MiddlewareFunc {
	return RequirePolicy("messaging.message.send.v1", opts)
}

func RequireRepositoryPolicy(opts Options) echo.MiddlewareFunc {
	return RequirePolicy("code.repository.merge.v1", opts)
}

func headersFromHTTP(headers http.Header) map[string]string {
	out := make(map[string]string, len(headers))
	for key, values := range headers {
		if len(values) > 0 {
			out[key] = values[0]
		}
	}
	return out
}
