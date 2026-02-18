"""Type definitions for the FastAPI middleware."""

from typing import List, Optional, Dict, Any, Callable
from dataclasses import dataclass

from aporthq_sdk_python import PassportData

# Re-export the shared type
AgentPassport = PassportData


@dataclass
class PolicyResult:
    """Policy evaluation result"""
    allowed: bool
    evaluation: Optional[Dict[str, Any]] = None
    error: Optional[Dict[str, Any]] = None


@dataclass
class PolicyEvaluation:
    """Policy evaluation details"""
    decision_id: Optional[str] = None
    remaining_daily_cap: Optional[int] = None
    violations: List[str] = None
    warnings: List[str] = None
    
    def __post_init__(self):
        if self.violations is None:
            self.violations = []
        if self.warnings is None:
            self.warnings = []


@dataclass
class AgentPassportMiddlewareOptions:
    """Options for the Agent Passport middleware (legacy/alternate; main options in middleware.py)."""

    base_url: Optional[str] = None
    timeout: int = 5
    cache: bool = True
    fail_closed: bool = True
    allowed_regions: Optional[List[str]] = None
    skip_paths: Optional[List[str]] = None
    skip_methods: Optional[List[str]] = None
    policy_id: Optional[str] = None

    def __post_init__(self):
        if self.allowed_regions is None:
            self.allowed_regions = []
        if self.skip_paths is None:
            self.skip_paths = []
        if self.skip_methods is None:
            self.skip_methods = ["OPTIONS"]


# Type aliases for middleware functions
AgentPassportMiddleware = Callable[[Request, Response], None]
PolicyMiddleware = Callable[[Request, Response], None]
