package common

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	aport "github.com/aporthq/aport-sdks-and-middlewares/sdk/go"
)

type Verifier interface {
	VerifyPolicy(ctx context.Context, agentID, policyID string, contextFields map[string]any, idempotencyKey string) (*aport.Decision, error)
	VerifyPolicyWithPassport(ctx context.Context, passport aport.PassportData, policyID string, contextFields map[string]any, idempotencyKey string) (*aport.Decision, error)
	VerifyPolicyWithPolicyInBody(ctx context.Context, agentIDOrPassport any, policy aport.PolicyPack, contextFields map[string]any, idempotencyKey string) (*aport.Decision, error)
	GetPassportView(ctx context.Context, agentID string) (map[string]any, error)
}

type Options struct {
	Client           Verifier
	BaseURL          string
	APIKey           string
	Timeout          time.Duration
	FailClosed       bool
	SkipPaths        []string
	PolicyID         string
	AgentID          string
	Context          map[string]any
	PassportFromBody bool
	PolicyFromBody   bool
}

type RequestData struct {
	Path           string
	Headers        map[string]string
	Body           map[string]any
	RequestContext context.Context
	IdempotencyKey string
}

type Result struct {
	AgentID      string
	Decision     *aport.Decision
	Passport     *aport.PassportData
	PassportView map[string]any
	Skipped      bool
}

type ErrorResponse struct {
	Status     int
	Error      string
	Message    string
	AgentID    string
	PolicyID   string
	DecisionID string
	Reasons    []aport.Reason
}

func (e *ErrorResponse) Body() map[string]any {
	body := map[string]any{
		"error":   e.Error,
		"message": e.Message,
	}
	if e.AgentID != "" {
		body["agent_id"] = e.AgentID
	}
	if e.PolicyID != "" {
		body["policy_id"] = e.PolicyID
	}
	if e.DecisionID != "" {
		body["decision_id"] = e.DecisionID
	}
	if len(e.Reasons) > 0 {
		body["reasons"] = e.Reasons
	}
	return body
}

func Evaluate(opts Options, data RequestData) (*Result, *ErrorResponse) {
	opts = normalizeOptions(opts)
	if shouldSkip(data.Path, opts.SkipPaths) {
		return &Result{Skipped: true}, nil
	}

	ctx := data.RequestContext
	if ctx == nil {
		ctx = context.Background()
	}

	body := data.Body
	if body == nil {
		body = map[string]any{}
	}

	bodyPassport := (*aport.PassportData)(nil)
	if opts.PassportFromBody {
		bodyPassport = parsePassport(body["passport"])
	}

	bodyPolicy := (*aport.PolicyPack)(nil)
	if opts.PolicyFromBody {
		bodyPolicy = parsePolicy(body["policy"])
	}

	agentID := opts.AgentID
	if agentID == "" && bodyPassport != nil {
		agentID = bodyPassport.AgentID
	}
	if agentID == "" {
		agentID = header(data.Headers, "x-agent-passport-id")
	}
	if agentID == "" {
		agentID = header(data.Headers, "x-agent-id")
	}

	if agentID == "" && bodyPassport == nil {
		if opts.FailClosed {
			return nil, &ErrorResponse{
				Status:  http.StatusUnauthorized,
				Error:   "missing_agent_id",
				Message: "Agent ID is required. Provide X-Agent-Passport-Id header, X-Agent-Id header, or body.passport.",
			}
		}
		return &Result{Skipped: true}, nil
	}

	contextFields := buildContext(body, opts.Context)
	idempotencyKey := data.IdempotencyKey
	if idempotencyKey == "" {
		idempotencyKey = header(data.Headers, "idempotency-key")
	}

	if opts.PolicyID == "" && bodyPolicy == nil {
		if bodyPassport != nil {
			return &Result{AgentID: bodyPassport.AgentID, Passport: bodyPassport}, nil
		}

		view, err := opts.Client.GetPassportView(ctx, agentID)
		if err != nil {
			return nil, apiError(err, "agent_verification_failed", agentID, "")
		}
		return &Result{AgentID: agentID, PassportView: view}, nil
	}

	var decision *aport.Decision
	var err error
	switch {
	case bodyPolicy != nil && bodyPassport != nil:
		decision, err = opts.Client.VerifyPolicyWithPolicyInBody(ctx, *bodyPassport, *bodyPolicy, contextFields, idempotencyKey)
	case bodyPolicy != nil:
		decision, err = opts.Client.VerifyPolicyWithPolicyInBody(ctx, agentID, *bodyPolicy, contextFields, idempotencyKey)
	case bodyPassport != nil:
		decision, err = opts.Client.VerifyPolicyWithPassport(ctx, *bodyPassport, opts.PolicyID, contextFields, idempotencyKey)
	default:
		decision, err = opts.Client.VerifyPolicy(ctx, agentID, opts.PolicyID, contextFields, idempotencyKey)
	}
	if err != nil {
		return nil, apiError(err, "api_error", agentID, effectivePolicyID(opts.PolicyID, bodyPolicy))
	}

	if !decision.Allow {
		return nil, &ErrorResponse{
			Status:     http.StatusForbidden,
			Error:      "policy_violation",
			Message:    "Policy violation",
			AgentID:    agentID,
			PolicyID:   effectivePolicyID(opts.PolicyID, bodyPolicy),
			DecisionID: decision.DecisionID,
			Reasons:    decision.Reasons,
		}
	}

	return &Result{AgentID: agentID, Decision: decision, Passport: bodyPassport}, nil
}

func normalizeOptions(opts Options) Options {
	if opts.Timeout == 0 {
		opts.Timeout = 5 * time.Second
	}
	if opts.SkipPaths == nil {
		opts.SkipPaths = []string{"/health", "/metrics", "/status"}
	}
	if !opts.FailClosed {
		opts.FailClosed = true
	}
	if !opts.PassportFromBody {
		opts.PassportFromBody = true
	}
	if !opts.PolicyFromBody {
		opts.PolicyFromBody = true
	}
	if opts.Client == nil {
		opts.Client = aport.NewClient(aport.Options{
			BaseURL: opts.BaseURL,
			APIKey:  opts.APIKey,
			Timeout: opts.Timeout,
		})
	}
	return opts
}

func shouldSkip(path string, skipPaths []string) bool {
	for _, skipPath := range skipPaths {
		if skipPath != "" && strings.HasPrefix(path, skipPath) {
			return true
		}
	}
	return false
}

func header(headers map[string]string, name string) string {
	for key, value := range headers {
		if strings.EqualFold(key, name) {
			return value
		}
	}
	return ""
}

func buildContext(body map[string]any, static map[string]any) map[string]any {
	contextFields := map[string]any{}
	for key, value := range body {
		if key == "passport" || key == "policy" {
			continue
		}
		contextFields[key] = value
	}
	for key, value := range static {
		contextFields[key] = value
	}
	return contextFields
}

func parsePassport(value any) *aport.PassportData {
	if value == nil {
		return nil
	}
	var passport aport.PassportData
	if !decode(value, &passport) || passport.AgentID == "" {
		return nil
	}
	if raw, ok := value.(map[string]any); ok {
		passport.Raw = raw
	}
	return &passport
}

func parsePolicy(value any) *aport.PolicyPack {
	if value == nil {
		return nil
	}
	var policy aport.PolicyPack
	if !decode(value, &policy) || policy.ID == "" {
		return nil
	}
	if raw, ok := value.(map[string]any); ok {
		policy.Raw = raw
	}
	return &policy
}

func decode(input any, output any) bool {
	raw, err := json.Marshal(input)
	if err != nil {
		return false
	}
	return json.Unmarshal(raw, output) == nil
}

func apiError(err error, code, agentID, policyID string) *ErrorResponse {
	response := &ErrorResponse{
		Status:   http.StatusInternalServerError,
		Error:    code,
		Message:  err.Error(),
		AgentID:  agentID,
		PolicyID: policyID,
	}
	if aportErr, ok := err.(*aport.AportError); ok {
		response.Status = aportErr.Status
		if response.Status == 0 {
			response.Status = http.StatusBadGateway
		}
		response.Reasons = aportErr.Reasons
		response.DecisionID = aportErr.DecisionID
	}
	return response
}

func effectivePolicyID(configured string, policy *aport.PolicyPack) string {
	if configured != "" {
		return configured
	}
	if policy != nil {
		return policy.ID
	}
	return ""
}
