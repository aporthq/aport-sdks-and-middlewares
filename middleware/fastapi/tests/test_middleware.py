"""Tests for the FastAPI middleware."""

import pytest
from unittest.mock import Mock, patch, AsyncMock
from fastapi import FastAPI, Request, Depends
from fastapi.testclient import TestClient
from httpx import AsyncClient

from aporthq_middleware_fastapi import (
    AgentPassportMiddleware,
    agent_passport_middleware,
    require_policy,
    require_refund_policy,
    require_data_export_policy,
    AgentPassportMiddlewareOptions,
    require_policy_dependency_with_context,
    require_policy_with_context,
)
from aporthq_sdk_python import AgentPassport, AportError
from aporthq_sdk_python.decision_types import PolicyVerificationResponse
from aporthq_middleware_fastapi.middleware import _response_to_dict


class TestAgentPassportMiddleware:
    """Test cases for AgentPassportMiddleware."""
    
    def setup_method(self):
        """Set up test fixtures."""
        self.app = FastAPI()
        self.mock_agent = AgentPassport(
            agent_id='ap_a2d10232c6534523812423eec8a1425c4567890abcdef',
            slug='test-agent',
            name='Test Agent',
            owner='test-owner',
            controller_type='org',
            claimed=True,
            role='Test Role',
            description='Test Description',
            status='active',
            verification_status='verified',
            permissions=['read:data'],
            limits={},
            regions=['us-east-1'],
            contact='test@example.com',
            links={},
            source='admin',
            created_at='2024-01-15T10:30:00Z',
            updated_at='2024-01-15T10:30:00Z',
            version='1.0.0',
        )

    @patch('aporthq_middleware_fastapi.middleware.create_client')
    def test_agent_passport_middleware_success(self, mock_create_client):
        """Test successful agent passport verification."""
        # Mock the SDK client
        mock_client = Mock()
        mock_client.get_passport_view = AsyncMock(return_value={
            'agent_id': 'ap_a2d10232c6534523812423eec8a1425c4567890abcdef',
            'slug': 'test-agent',
            'name': 'Test Agent',
            'status': 'active',
        })
        mock_create_client.return_value = mock_client

        # Add middleware to app
        self.app.add_middleware(AgentPassportMiddleware)
        
        @self.app.get("/test")
        async def test_endpoint(request: Request):
            return {"agent": getattr(request.state, 'agent', None)}

        # Test the endpoint
        with TestClient(self.app) as client:
            response = client.get(
                "/test",
                headers={"X-Agent-Passport-Id": "ap_a2d10232c6534523812423eec8a1425c4567890abcdef"}
            )
            
            assert response.status_code == 200
            assert response.json()["agent"]["agent_id"] == "ap_a2d10232c6534523812423eec8a1425c4567890abcdef"

    def test_agent_passport_middleware_no_agent_id(self):
        """Test middleware behavior when no agent ID is provided."""
        # Add middleware to app
        self.app.add_middleware(AgentPassportMiddleware)
        
        @self.app.get("/test")
        async def test_endpoint(request: Request):
            return {"agent": getattr(request.state, 'agent', None)}

        # Test the endpoint without agent ID
        with TestClient(self.app) as client:
            response = client.get("/test")
            
            assert response.status_code == 401
            assert response.json()["error"] == "missing_agent_id"

    def test_agent_passport_middleware_skip_paths(self):
        """Test middleware skips certain paths."""
        # Add middleware to app with skip paths
        self.app.add_middleware(AgentPassportMiddleware, skip_paths=["/health"])
        
        @self.app.get("/health")
        async def health_endpoint():
            return {"status": "ok"}

        # Test the health endpoint
        with TestClient(self.app) as client:
            response = client.get("/health")
            
            assert response.status_code == 200
            assert response.json()["status"] == "ok"

    @patch('aporthq_middleware_fastapi.middleware.create_client')
    def test_agent_passport_middleware_with_policy(self, mock_create_client):
        """Test middleware with policy enforcement."""
        # Mock the SDK client
        mock_client = Mock()
        mock_client.verify_policy = AsyncMock(return_value={
            'decision_id': 'dec_123',
            'allow': True,
            'reasons': []
        })
        mock_create_client.return_value = mock_client

        # Add middleware to app with policy
        self.app.add_middleware(
            AgentPassportMiddleware, 
            options=AgentPassportMiddlewareOptions(policy_id="finance.payment.refund.v1")
        )
        
        @self.app.post("/refund")
        async def refund_endpoint(request: Request):
            return {
                "agent": getattr(request.state, 'agent', None),
                "policy_result": getattr(request.state, 'policy_result', None)
            }

        # Test the endpoint
        with TestClient(self.app) as client:
            response = client.post(
                "/refund",
                headers={"X-Agent-Passport-Id": "ap_test123"},
                json={"amount": 100, "currency": "USD"}
            )
            
            assert response.status_code == 200
            assert response.json()["agent"]["agent_id"] == "ap_test123"
            assert response.json()["policy_result"]["allow"] is True

    @patch('aporthq_middleware_fastapi.middleware.create_client')
    def test_agent_passport_middleware_policy_failure(self, mock_create_client):
        """Test middleware with policy enforcement failure."""
        # Mock the SDK client
        mock_client = Mock()
        mock_client.verify_policy = AsyncMock(return_value={
            'decision_id': 'dec_123',
            'allow': False,
            'reasons': [{"code": "INSUFFICIENT_PERMISSIONS", "message": "Access denied"}]
        })
        mock_create_client.return_value = mock_client

        # Add middleware to app with policy
        self.app.add_middleware(
            AgentPassportMiddleware, 
            options=AgentPassportMiddlewareOptions(policy_id="finance.payment.refund.v1")
        )
        
        @self.app.post("/refund")
        async def refund_endpoint(request: Request):
            return {"success": True}

        # Test the endpoint
        with TestClient(self.app) as client:
            response = client.post(
                "/refund",
                headers={"X-Agent-Passport-Id": "ap_test123"},
                json={"amount": 100, "currency": "USD"}
            )
            
            assert response.status_code == 403
            assert response.json()["error"] == "policy_violation"

    def test_response_to_dict_preserves_decision_metadata(self):
        """Decision helpers should not discard verification evidence fields."""
        response = PolicyVerificationResponse(
            decision_id="dec_123",
            allow=True,
            reasons=[],
            provenance="ci_time",
            policy_hash="sha256:abc",
            github={"repository": "aporthq/agent-passport"},
            signature_status={"signature_valid": True},
        )

        result = _response_to_dict(response)

        assert result["decision_id"] == "dec_123"
        assert result["allow"] is True
        assert result["provenance"] == "ci_time"
        assert result["policy_hash"] == "sha256:abc"
        assert result["github"]["repository"] == "aporthq/agent-passport"
        assert result["signature_status"]["signature_valid"] is True


class TestRequirePolicy:
    """Test cases for require_policy decorator."""
    
    def setup_method(self):
        """Set up test fixtures."""
        self.app = FastAPI()

    @patch('aporthq_middleware_fastapi.middleware.create_client')
    def test_require_policy_success(self, mock_create_client):
        """Test require_policy dependency with successful verification."""
        # Mock the SDK client
        mock_client = Mock()
        mock_client.verify_policy = AsyncMock(return_value={
            'decision_id': 'dec_123',
            'allow': True,
            'reasons': []
        })
        mock_create_client.return_value = mock_client

        @self.app.post("/refund")
        async def refund_endpoint(
            request: Request,
            policy_data: dict = Depends(require_policy("finance.payment.refund.v1"))
        ):
            return {"success": True, "policy_data": policy_data}

        # Test the endpoint
        with TestClient(self.app) as client:
            response = client.post(
                "/refund",
                headers={"X-Agent-Passport-Id": "ap_test123"},
                json={"amount": 100, "currency": "USD"}
            )
            
            assert response.status_code == 200
            assert response.json()["success"] is True
            assert response.json()["policy_data"]["agent"]["agent_id"] == "ap_test123"

    @patch('aporthq_middleware_fastapi.middleware.create_client')
    def test_require_policy_uses_configured_agent_id(self, mock_create_client):
        """Configured hosted passport IDs should not require a request header."""
        mock_client = Mock()
        mock_client.verify_policy = AsyncMock(return_value={
            'decision_id': 'dec_123',
            'allow': True,
            'reasons': []
        })
        mock_create_client.return_value = mock_client

        @self.app.post("/read-file")
        async def read_file_endpoint(
            policy_data: dict = Depends(require_policy("data.file.read.v1", "ap_configured123"))
        ):
            return {"success": True, "policy_data": policy_data}

        with TestClient(self.app) as client:
            response = client.post(
                "/read-file",
                json={"file_path": "/tmp/report.txt"}
            )

            assert response.status_code == 200
            assert response.json()["policy_data"]["agent"]["agent_id"] == "ap_configured123"
            mock_client.verify_policy.assert_awaited_once()
            assert mock_client.verify_policy.await_args.args[0] == "ap_configured123"

    @patch('aporthq_middleware_fastapi.middleware.create_client')
    def test_require_policy_sets_attribute_accessible_agent_state(self, mock_create_client):
        """Route examples can use request.state.agent.agent_id after verification."""
        mock_client = Mock()
        mock_client.verify_policy = AsyncMock(return_value={
            'decision_id': 'dec_123',
            'allow': True,
            'reasons': []
        })
        mock_create_client.return_value = mock_client

        @self.app.post("/read-file")
        async def read_file_endpoint(
            request: Request,
            policy_data: dict = Depends(require_policy("data.file.read.v1", "ap_configured123"))
        ):
            return {
                "agent_id_from_state": request.state.agent.agent_id,
                "agent_id_from_dependency": policy_data["agent"]["agent_id"],
            }

        with TestClient(self.app) as client:
            response = client.post("/read-file", json={"file_path": "/tmp/report.txt"})

            assert response.status_code == 200
            assert response.json()["agent_id_from_state"] == "ap_configured123"
            assert response.json()["agent_id_from_dependency"] == "ap_configured123"

    @patch('aporthq_middleware_fastapi.middleware.create_client')
    def test_require_policy_dependency_with_context_is_dependency(self, mock_create_client):
        """Context helper must be dependency-shaped for FastAPI Depends."""
        mock_client = Mock()
        mock_client.verify_policy = AsyncMock(return_value={
            'decision_id': 'dec_123',
            'allow': True,
            'reasons': []
        })
        mock_create_client.return_value = mock_client

        @self.app.post("/repo/pr")
        async def create_pr_endpoint(
            request: Request,
            policy_data: dict = Depends(
                require_policy_dependency_with_context(
                    "code.repository.merge.v1",
                    {"action": "pr.create", "repository": "aporthq/agent-passport"},
                    "ap_configured123",
                )
            )
        ):
            return {
                "agent_id": request.state.agent.agent_id,
                "decision_id": policy_data["policy_result"]["decision_id"],
            }

        with TestClient(self.app) as client:
            response = client.post("/repo/pr", json={"title": "Add guardrail"})

            assert response.status_code == 200
            assert response.json() == {"agent_id": "ap_configured123", "decision_id": "dec_123"}
            mock_client.verify_policy.assert_awaited_once_with(
                "ap_configured123",
                "code.repository.merge.v1",
                {
                    "title": "Add guardrail",
                    "action": "pr.create",
                    "repository": "aporthq/agent-passport",
                },
            )

    @patch('aporthq_middleware_fastapi.middleware.create_client')
    def test_require_policy_with_context_preserves_middleware_contract(self, mock_create_client):
        """Existing app.middleware("http") users must keep working after upgrade."""
        mock_client = Mock()
        mock_client.verify_policy = AsyncMock(return_value={
            'decision_id': 'dec_456',
            'allow': True,
            'reasons': []
        })
        mock_create_client.return_value = mock_client

        self.app.middleware("http")(
            require_policy_with_context(
                "code.repository.merge.v1",
                {"action": "pr.create", "repository": "aporthq/agent-passport"},
                "ap_configured123",
            )
        )

        @self.app.post("/repo/pr")
        async def create_pr_endpoint(request: Request):
            return {
                "agent_id": request.state.agent.agent_id,
                "decision_id": request.state.policy_result["decision_id"],
            }

        with TestClient(self.app) as client:
            response = client.post("/repo/pr", json={"title": "Add guardrail"})

            assert response.status_code == 200
            assert response.json() == {"agent_id": "ap_configured123", "decision_id": "dec_456"}
            mock_client.verify_policy.assert_awaited_once_with(
                "ap_configured123",
                "code.repository.merge.v1",
                {
                    "title": "Add guardrail",
                    "action": "pr.create",
                    "repository": "aporthq/agent-passport",
                },
            )

    @patch('aporthq_middleware_fastapi.middleware.create_client')
    def test_require_policy_failure(self, mock_create_client):
        """Test require_policy dependency with failed verification."""
        # Mock the SDK client
        mock_client = Mock()
        mock_client.verify_policy = AsyncMock(return_value={
            'decision_id': 'dec_123',
            'allow': False,
            'reasons': [{"code": "INSUFFICIENT_PERMISSIONS", "message": "Access denied"}]
        })
        mock_create_client.return_value = mock_client

        @self.app.post("/refund")
        async def refund_endpoint(
            request: Request,
            policy_data: dict = Depends(require_policy("finance.payment.refund.v1"))
        ):
            return {"success": True}

        # Test the endpoint
        with TestClient(self.app) as client:
            response = client.post(
                "/refund",
                headers={"X-Agent-Passport-Id": "ap_test123"},
                json={"amount": 100, "currency": "USD"}
            )
            
            assert response.status_code == 403
            assert response.json()["detail"]["error"] == "policy_violation"


class TestRequireRefundPolicy:
    """Test cases for require_refund_policy convenience function."""
    
    def setup_method(self):
        """Set up test fixtures."""
        self.app = FastAPI()

    @patch('aporthq_middleware_fastapi.middleware.create_client')
    def test_require_refund_policy(self, mock_create_client):
        """Test require_refund_policy convenience function."""
        # Mock the SDK client
        mock_client = Mock()
        mock_client.verify_policy = AsyncMock(return_value={
            'decision_id': 'dec_123',
            'allow': True,
            'reasons': []
        })
        mock_create_client.return_value = mock_client

        @self.app.post("/refund")
        async def refund_endpoint(
            request: Request,
            policy_data: dict = Depends(require_refund_policy())
        ):
            return {"success": True, "policy_data": policy_data}

        # Test the endpoint
        with TestClient(self.app) as client:
            response = client.post(
                "/refund",
                headers={"X-Agent-Passport-Id": "ap_test123"},
                json={"amount": 100, "currency": "USD", "order_id": "order_123"}
            )
            
            assert response.status_code == 200
            assert response.json()["success"] is True
            assert response.json()["policy_data"]["agent"]["agent_id"] == "ap_test123"


class TestRequireDataExportPolicy:
    """Test cases for require_data_export_policy convenience function."""
    
    def setup_method(self):
        """Set up test fixtures."""
        self.app = FastAPI()

    @patch('aporthq_middleware_fastapi.middleware.create_client')
    def test_require_data_export_policy(self, mock_create_client):
        """Test require_data_export_policy convenience function."""
        # Mock the SDK client
        mock_client = Mock()
        mock_client.verify_policy = AsyncMock(return_value={
            'decision_id': 'dec_123',
            'allow': True,
            'reasons': []
        })
        mock_create_client.return_value = mock_client

        @self.app.post("/data-export")
        async def data_export_endpoint(
            request: Request,
            policy_data: dict = Depends(require_data_export_policy())
        ):
            return {"success": True, "policy_data": policy_data}

        # Test the endpoint
        with TestClient(self.app) as client:
            response = client.post(
                "/data-export",
                headers={"X-Agent-Passport-Id": "ap_test123"},
                json={"data_types": ["user_data"], "destination": "s3://bucket"}
            )
            
            assert response.status_code == 200
            assert response.json()["success"] is True
            assert response.json()["policy_data"]["agent"]["agent_id"] == "ap_test123"
