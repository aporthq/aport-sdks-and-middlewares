package aport

import (
	"fmt"
	"strings"
)

// AportError reports API, timeout, and network failures with structured
// decision denial details when the APort API returned them.
type AportError struct {
	Status       int
	Reasons      []Reason
	DecisionID   string
	ServerTiming string
	RawResponse  string
	Err          error
}

func (e *AportError) Error() string {
	if e == nil {
		return ""
	}
	if len(e.Reasons) == 0 {
		if e.Err != nil {
			return fmt.Sprintf("aport request failed: %d: %v", e.Status, e.Err)
		}
		return fmt.Sprintf("aport request failed: %d", e.Status)
	}

	messages := make([]string, 0, len(e.Reasons))
	for _, reason := range e.Reasons {
		if reason.Message != "" {
			messages = append(messages, reason.Message)
			continue
		}
		if reason.Code != "" {
			messages = append(messages, reason.Code)
		}
	}
	if len(messages) == 0 {
		return fmt.Sprintf("aport request failed: %d", e.Status)
	}
	return fmt.Sprintf("aport request failed: %d %s", e.Status, strings.Join(messages, ", "))
}

func (e *AportError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
