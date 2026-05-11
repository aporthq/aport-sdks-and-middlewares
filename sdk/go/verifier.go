package aport

import "context"

type PolicyVerifier struct {
	client *Client
}

func NewPolicyVerifier(client *Client) *PolicyVerifier {
	return &PolicyVerifier{client: client}
}

func (v *PolicyVerifier) VerifyRefund(ctx context.Context, agentID string, contextFields map[string]any, idempotencyKey string) (*Decision, error) {
	return v.client.VerifyPolicy(ctx, agentID, "finance.payment.refund.v1", contextFields, idempotencyKey)
}

func (v *PolicyVerifier) VerifyRelease(ctx context.Context, agentID string, contextFields map[string]any, idempotencyKey string) (*Decision, error) {
	return v.client.VerifyPolicy(ctx, agentID, "code.release.publish.v1", contextFields, idempotencyKey)
}

func (v *PolicyVerifier) VerifyDataExport(ctx context.Context, agentID string, contextFields map[string]any, idempotencyKey string) (*Decision, error) {
	return v.client.VerifyPolicy(ctx, agentID, "data.export.create.v1", contextFields, idempotencyKey)
}

func (v *PolicyVerifier) VerifyMessaging(ctx context.Context, agentID string, contextFields map[string]any, idempotencyKey string) (*Decision, error) {
	return v.client.VerifyPolicy(ctx, agentID, "messaging.message.send.v1", contextFields, idempotencyKey)
}

func (v *PolicyVerifier) VerifyRepository(ctx context.Context, agentID string, contextFields map[string]any, idempotencyKey string) (*Decision, error) {
	return v.client.VerifyPolicy(ctx, agentID, "code.repository.merge.v1", contextFields, idempotencyKey)
}

func (v *PolicyVerifier) VerifyDeliverableTaskComplete(ctx context.Context, agentID string, contextFields map[string]any, idempotencyKey string) (*Decision, error) {
	return v.client.VerifyPolicy(ctx, agentID, "deliverable.task.complete.v1", contextFields, idempotencyKey)
}
