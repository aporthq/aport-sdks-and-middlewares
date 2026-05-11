package aport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	DefaultBaseURL = "https://api.aport.io"
	DefaultTimeout = 800 * time.Millisecond
	userAgent      = "aport-sdk-go/0.1.0"
)

// Client is a thin APort API client. It does not evaluate policies locally; it
// only transports requests and returns the API's structured decisions.
type Client struct {
	baseURL    string
	apiKey     string
	timeout    time.Duration
	httpClient HTTPDoer

	mu              sync.Mutex
	jwksCache       *JWKS
	jwksCacheExpiry time.Time
}

// APortClient is an alias for compatibility with the naming used by other SDKs.
type APortClient = Client

func NewClient(opts Options) *Client {
	baseURL := strings.TrimRight(opts.BaseURL, "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	}

	return &Client{
		baseURL:    baseURL,
		apiKey:     opts.APIKey,
		timeout:    timeout,
		httpClient: httpClient,
	}
}

func NewAPortClient(opts Options) *Client {
	return NewClient(opts)
}

// VerifyPolicy verifies a policy against an agent in cloud mode, where APort
// fetches the agent passport from its registry.
func (c *Client) VerifyPolicy(ctx context.Context, agentID, policyID string, contextFields map[string]any, idempotencyKey string) (*Decision, error) {
	body := buildPolicyRequestBody(agentID, policyID, contextFields, idempotencyKey, nil, nil)
	raw, serverTiming, err := c.request(ctx, http.MethodPost, "/api/verify/policy/"+url.PathEscape(policyID), body, idempotencyKey)
	if err != nil {
		return nil, err
	}
	return decodeDecision(raw, serverTiming)
}

// VerifyPolicyWithPassport verifies a policy with the passport embedded in the
// request body, useful for local or dynamic passport flows.
func (c *Client) VerifyPolicyWithPassport(ctx context.Context, passport PassportData, policyID string, contextFields map[string]any, idempotencyKey string) (*Decision, error) {
	body := buildPolicyRequestBody(passport.AgentID, policyID, contextFields, idempotencyKey, passport, nil)
	raw, serverTiming, err := c.request(ctx, http.MethodPost, "/api/verify/policy/"+url.PathEscape(policyID), body, idempotencyKey)
	if err != nil {
		return nil, err
	}
	return decodeDecision(raw, serverTiming)
}

// VerifyPolicyWithPolicyInBody verifies against a policy pack included in the
// request body. agentIDOrPassport accepts string, PassportData, or *PassportData.
func (c *Client) VerifyPolicyWithPolicyInBody(ctx context.Context, agentIDOrPassport any, policy PolicyPack, contextFields map[string]any, idempotencyKey string) (*Decision, error) {
	var agentID string
	var passport any

	switch value := agentIDOrPassport.(type) {
	case string:
		agentID = value
	case PassportData:
		agentID = value.AgentID
		passport = value
	case *PassportData:
		if value != nil {
			agentID = value.AgentID
			passport = value
		}
	default:
		return nil, &AportError{
			Status: 400,
			Reasons: []Reason{{
				Code:    "INVALID_AGENT",
				Message: "agentIDOrPassport must be string, PassportData, or *PassportData",
			}},
		}
	}

	body := buildPolicyRequestBody(agentID, policy.ID, contextFields, idempotencyKey, passport, policy)
	raw, serverTiming, err := c.request(ctx, http.MethodPost, "/api/verify/policy/IN_BODY", body, idempotencyKey)
	if err != nil {
		return nil, err
	}
	return decodeDecision(raw, serverTiming)
}

func (c *Client) GetDecisionToken(ctx context.Context, agentID, policyID string, contextFields map[string]any) (string, error) {
	body := map[string]any{
		"agent_id": agentID,
		"context":  cleanContext(contextFields),
	}
	raw, _, err := c.request(ctx, http.MethodPost, "/api/verify/token/"+url.PathEscape(policyID), body, "")
	if err != nil {
		return "", err
	}

	var response DecisionTokenResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return "", err
	}
	return response.Token, nil
}

func (c *Client) ValidateDecisionToken(ctx context.Context, token string) (*Decision, error) {
	raw, serverTiming, err := c.request(ctx, http.MethodPost, "/api/verify/token/validate", map[string]any{"token": token}, "")
	if err != nil {
		return nil, err
	}
	return decodeDecision(raw, serverTiming)
}

// ValidateDecisionTokenLocal currently mirrors the Node and Python SDKs: it
// fetches JWKS to validate availability, then uses the server validation route.
func (c *Client) ValidateDecisionTokenLocal(ctx context.Context, token string) (*Decision, error) {
	if _, err := c.GetJWKS(ctx); err != nil {
		return nil, &AportError{
			Status: 401,
			Reasons: []Reason{{
				Code:    "INVALID_TOKEN",
				Message: "Token validation failed",
			}},
			Err: err,
		}
	}
	return c.ValidateDecisionToken(ctx, token)
}

func (c *Client) GetPassportView(ctx context.Context, agentID string) (map[string]any, error) {
	raw, _, err := c.request(ctx, http.MethodGet, "/api/passports/"+url.PathEscape(agentID)+"/verify_view", nil, "")
	if err != nil {
		return nil, err
	}

	var passport map[string]any
	if err := json.Unmarshal(raw, &passport); err != nil {
		return nil, err
	}
	return passport, nil
}

func (c *Client) GetJWKS(ctx context.Context) (*JWKS, error) {
	c.mu.Lock()
	if c.jwksCache != nil && time.Now().Before(c.jwksCacheExpiry) {
		cached := *c.jwksCache
		c.mu.Unlock()
		return &cached, nil
	}
	c.mu.Unlock()

	raw, _, err := c.request(ctx, http.MethodGet, "/jwks.json", nil, "")
	if err != nil {
		return nil, &AportError{
			Status: 500,
			Reasons: []Reason{{
				Code:    "JWKS_FETCH_FAILED",
				Message: "Failed to fetch JWKS",
			}},
			Err: err,
		}
	}

	var jwks JWKS
	if err := json.Unmarshal(raw, &jwks); err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.jwksCache = &jwks
	c.jwksCacheExpiry = time.Now().Add(5 * time.Minute)
	c.mu.Unlock()

	return &jwks, nil
}

func (c *Client) request(parent context.Context, method, path string, body any, idempotencyKey string) ([]byte, string, error) {
	if parent == nil {
		parent = context.Background()
	}

	ctx, cancel := context.WithTimeout(parent, c.timeout)
	defer cancel()

	var requestBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, "", err
		}
		requestBody = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.url(path), requestBody)
	if err != nil {
		return nil, "", err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, "", &AportError{
				Status: 408,
				Reasons: []Reason{{
					Code:    "TIMEOUT",
					Message: "Request timeout",
				}},
				Err: err,
			}
		}
		return nil, "", &AportError{
			Status: 0,
			Reasons: []Reason{{
				Code:    "NETWORK_ERROR",
				Message: err.Error(),
			}},
			Err: err,
		}
	}
	defer res.Body.Close()

	raw, readErr := io.ReadAll(res.Body)
	if readErr != nil {
		return nil, res.Header.Get("Server-Timing"), readErr
	}

	serverTiming := res.Header.Get("Server-Timing")
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, serverTiming, buildAPIError(res.StatusCode, raw, serverTiming)
	}

	return raw, serverTiming, nil
}

func (c *Client) url(path string) string {
	cleanPath := path
	if !strings.HasPrefix(cleanPath, "/") {
		cleanPath = "/" + cleanPath
	}
	return c.baseURL + cleanPath
}

func buildPolicyRequestBody(agentID, policyID string, contextFields map[string]any, idempotencyKey string, passport any, policy any) VerificationRequestBody {
	context := map[string]any{}
	if agentID != "" {
		context["agent_id"] = agentID
	}
	if policyID != "" {
		context["policy_id"] = policyID
	}
	if idempotencyKey != "" {
		context["idempotency_key"] = idempotencyKey
	}
	for key, value := range contextFields {
		context[key] = value
	}

	body := VerificationRequestBody{Context: context}
	if passport != nil {
		body.Passport = passport
	}
	if policy != nil {
		body.Policy = policy
	}
	return body
}

func cleanContext(contextFields map[string]any) map[string]any {
	if len(contextFields) == 0 {
		return map[string]any{}
	}
	context := make(map[string]any, len(contextFields))
	for key, value := range contextFields {
		context[key] = value
	}
	return context
}

func decodeDecision(raw []byte, serverTiming string) (*Decision, error) {
	var wrapped struct {
		Decision *Decision `json:"decision"`
	}
	if err := json.Unmarshal(raw, &wrapped); err == nil && wrapped.Decision != nil {
		attachServerTiming(wrapped.Decision, serverTiming)
		return wrapped.Decision, nil
	}

	var decision Decision
	if err := json.Unmarshal(raw, &decision); err != nil {
		return nil, err
	}
	attachServerTiming(&decision, serverTiming)
	return &decision, nil
}

func attachServerTiming(decision *Decision, serverTiming string) {
	if decision == nil || serverTiming == "" {
		return
	}
	if decision.Meta == nil {
		decision.Meta = map[string]any{}
	}
	decision.Meta["serverTiming"] = serverTiming
}

func buildAPIError(status int, raw []byte, serverTiming string) error {
	payload := struct {
		Reasons    []Reason `json:"reasons"`
		DecisionID string   `json:"decision_id"`
		Error      string   `json:"error"`
		Message    string   `json:"message"`
	}{}
	_ = json.Unmarshal(raw, &payload)

	reasons := payload.Reasons
	if len(reasons) == 0 {
		message := payload.Message
		if message == "" {
			message = payload.Error
		}
		if message == "" {
			message = fmt.Sprintf("HTTP %d", status)
		}
		reasons = []Reason{{
			Code:    "API_ERROR",
			Message: message,
		}}
	}

	return &AportError{
		Status:       status,
		Reasons:      reasons,
		DecisionID:   payload.DecisionID,
		ServerTiming: serverTiming,
		RawResponse:  string(raw),
	}
}
