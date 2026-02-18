/**
 * Shared types for SDK-Server communication
 * These types are used by both the SDK and the API endpoints
 */

import type { PassportData } from "./passport";

/** Minimal OAP policy pack shape when supplying policy in body (pack_id = IN_BODY). */
export interface PolicyPack {
  id: string;
  requires_capabilities: string[];
  [key: string]: any;
}

/**
 * Request body for POST /api/verify/policy/{pack_id}.
 * - context is required and must contain agent_id (or provide passport for local mode).
 * - When pack_id is IN_BODY, policy is required.
 */
export interface PolicyVerificationRequestBody {
  context: {
    agent_id?: string;
    policy_id?: string;
    idempotency_key?: string;
    [key: string]: any;
  };
  passport?: PassportData;
  policy?: PolicyPack;
}

/** Convenience shape: agent_id + context. SDK builds body.context from this. */
export interface PolicyVerificationRequest {
  agent_id: string;
  idempotency_key?: string;
  context: Record<string, any>;
  /** Passport in body (local mode). When set, agent_id can be omitted from context. */
  passport?: PassportData;
  /** Policy pack in body (use pack_id IN_BODY). When set, path is /api/verify/policy/IN_BODY. */
  policy?: PolicyPack;
}

export interface PolicyVerificationResponse {
  decision_id: string;
  allow: boolean;
  reasons?: Array<{
    code: string;
    message: string;
    severity?: "info" | "warning" | "error";
  }>;
  assurance_level?: "L0" | "L1" | "L2" | "L3" | "L4";
  expires_in?: number; // for decision token mode
  passport_digest?: string;
  signature?: string; // HMAC/JWT
  created_at?: string;
  _meta?: {
    serverTiming?: string;
  };
}

// Legacy types for backward compatibility
export interface Decision extends PolicyVerificationResponse {}

export interface DecisionReason {
  code: string;
  message: string;
  severity: "info" | "warning" | "error";
}

export interface VerificationContext {
  agent_id: string;
  policy_id: string;
  context?: Record<string, any>;
  idempotency_key?: string;
}

// JWKS support for local token validation
export interface Jwks {
  keys: Array<{
    kty: string;
    use: string;
    kid: string;
    x5t: string;
    n: string;
    e: string;
    x5c: string[];
  }>;
}
