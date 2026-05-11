package ginmiddleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/aporthq/aport-sdks-and-middlewares/sdk/go/middleware/common"
)

type Options = common.Options

func Middleware(opts Options) gin.HandlerFunc {
	return func(c *gin.Context) {
		body := map[string]any{}
		if c.Request.Body != nil && c.Request.ContentLength != 0 {
			_ = c.ShouldBindBodyWith(&body, binding.JSON)
		}

		result, errResponse := common.Evaluate(opts, common.RequestData{
			Path:           c.Request.URL.Path,
			Headers:        headersFromHTTP(c.Request.Header),
			Body:           body,
			RequestContext: c.Request.Context(),
		})
		if errResponse != nil {
			c.AbortWithStatusJSON(errResponse.Status, errResponse.Body())
			return
		}
		if result == nil || result.Skipped {
			c.Next()
			return
		}

		c.Set("aport_agent_id", result.AgentID)
		c.Set("aport_decision", result.Decision)
		c.Set("aport_passport", result.Passport)
		c.Set("aport_passport_view", result.PassportView)
		c.Next()
	}
}

func RequirePolicy(policyID string, opts Options) gin.HandlerFunc {
	opts.PolicyID = policyID
	return Middleware(opts)
}

func RequireRefundPolicy(opts Options) gin.HandlerFunc {
	return RequirePolicy("finance.payment.refund.v1", opts)
}

func RequireDataExportPolicy(opts Options) gin.HandlerFunc {
	return RequirePolicy("data.export.create.v1", opts)
}

func RequireMessagingPolicy(opts Options) gin.HandlerFunc {
	return RequirePolicy("messaging.message.send.v1", opts)
}

func RequireRepositoryPolicy(opts Options) gin.HandlerFunc {
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
