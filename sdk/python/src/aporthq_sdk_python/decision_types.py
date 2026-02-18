"""
Shared types for SDK-Server communication.
Align with POST /api/verify/policy/{pack_id}:
  body.context (required), body.passport (optional), body.policy (optional; required when pack_id is IN_BODY).
"""

from typing import Any, Dict, List, Optional, Union
from dataclasses import dataclass, field


# Minimal OAP policy pack when supplying policy in body (pack_id = IN_BODY)
PolicyPack = Dict[str, Any]  # must have "id" and "requires_capabilities"


@dataclass
class PolicyVerificationRequestBody:
    """Request body for POST /api/verify/policy/{pack_id}. API expects this shape."""

    context: Dict[str, Any]  # must include agent_id or provide passport
    passport: Optional[Dict[str, Any]] = None  # local mode
    policy: Optional[PolicyPack] = None  # required when pack_id is IN_BODY


# Convenience shape; SDK builds PolicyVerificationRequestBody from this
@dataclass
class PolicyVerificationRequest:
    """Convenience request shape. SDK builds body.context from agent_id, policy_id, context."""

    agent_id: str
    idempotency_key: Optional[str] = None
    context: Dict[str, Any] = field(default_factory=dict)
    passport: Optional[Dict[str, Any]] = None  # passport in body (local mode)
    policy: Optional[PolicyPack] = None  # policy in body (use path IN_BODY)


@dataclass
class PolicyVerificationResponse:
    """Canonical response shape for policy verification (inner decision object from API)."""

    decision_id: str
    allow: bool
    reasons: Optional[List[Dict[str, Any]]] = None
    assurance_level: Optional[str] = None  # "L0" | "L1" | "L2" | "L3" | "L4"
    expires_in: Optional[int] = None  # for decision token mode
    passport_digest: Optional[str] = None
    signature: Optional[str] = None  # HMAC/JWT
    created_at: Optional[str] = None
    _meta: Optional[Dict[str, Any]] = None  # Server-Timing, etc.

    @classmethod
    def from_api_response(cls, data: Dict[str, Any]) -> "PolicyVerificationResponse":
        """Build from API response; unwraps .decision when present."""
        decision = data.get("decision")
        payload = dict(decision) if decision is not None else dict(data)
        if "_meta" in data and "_meta" not in payload:
            payload["_meta"] = data["_meta"]
        fields = set(cls.__dataclass_fields__)
        kwargs = {k: payload[k] for k in fields if k in payload}
        return cls(**kwargs)


# Legacy types for backward compatibility
@dataclass
class DecisionReason:
    """Reason for a policy decision."""
    
    code: str
    message: str
    severity: str  # "info" | "warning" | "error"


@dataclass
class Decision(PolicyVerificationResponse):
    """Policy decision result (legacy compatibility)."""
    pass


@dataclass
class VerificationContext:
    """Context for policy verification (legacy compatibility)."""
    
    agent_id: str
    policy_id: str
    context: Optional[Dict[str, Any]] = None
    idempotency_key: Optional[str] = None


# JWKS support for local token validation
@dataclass
class JwksKey:
    """JSON Web Key."""
    
    kty: str
    use: str
    kid: str
    x5t: str
    n: str
    e: str
    x5c: List[str]


@dataclass
class Jwks:
    """JSON Web Key Set."""
    
    keys: List[JwksKey]