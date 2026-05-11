package aport

import (
	"net/http"
	"time"
)

type Options struct {
	BaseURL    string
	APIKey     string
	Timeout    time.Duration
	HTTPClient HTTPDoer
}

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type Reason struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Severity string `json:"severity,omitempty"`
}

type Decision struct {
	DecisionID     string         `json:"decision_id,omitempty"`
	Allow          bool           `json:"allow"`
	Reasons        []Reason       `json:"reasons,omitempty"`
	AssuranceLevel string         `json:"assurance_level,omitempty"`
	ExpiresIn      int            `json:"expires_in,omitempty"`
	PassportDigest string         `json:"passport_digest,omitempty"`
	Signature      string         `json:"signature,omitempty"`
	CreatedAt      string         `json:"created_at,omitempty"`
	Meta           map[string]any `json:"_meta,omitempty"`
}

type PassportData struct {
	AgentID string         `json:"agent_id"`
	Claims  map[string]any `json:"claims,omitempty"`
	Limits  map[string]any `json:"limits,omitempty"`
	Status  string         `json:"status,omitempty"`
	Raw     map[string]any `json:"-"`
}

type PolicyPack struct {
	ID                   string         `json:"id"`
	RequiresCapabilities []string       `json:"requires_capabilities,omitempty"`
	Raw                  map[string]any `json:"-"`
}

type VerificationRequestBody struct {
	Context  map[string]any `json:"context"`
	Passport any            `json:"passport,omitempty"`
	Policy   any            `json:"policy,omitempty"`
}

type DecisionTokenResponse struct {
	Token string `json:"token"`
}

type JWKS struct {
	Keys []JWK `json:"keys"`
}

type JWK struct {
	Kty string   `json:"kty"`
	Use string   `json:"use,omitempty"`
	Kid string   `json:"kid"`
	X5t string   `json:"x5t,omitempty"`
	N   string   `json:"n,omitempty"`
	E   string   `json:"e,omitempty"`
	X5c []string `json:"x5c,omitempty"`
}
