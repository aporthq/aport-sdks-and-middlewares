module APort
  module SDK
    module Rails
      module ControllerHelpers
        def aport_agent_id
          request.env["aport.agent_id"]
        end

        def aport_decision
          request.env["aport.decision"]
        end

        def verify_aport_policy!(policy_id, context: {}, agent_id: nil, idempotency_key: nil)
          resolved_agent_id = agent_id || aport_agent_id || request.headers["X-Agent-Passport-Id"] || request.headers["X-Agent-Id"]
          SDK.client.verify_policy(
            agent_id: resolved_agent_id,
            policy_id: policy_id,
            context: context,
            idempotency_key: idempotency_key
          ).tap do |decision|
            request.env["aport.decision"] = decision
            raise Error.new(status: 403, reasons: decision["reasons"], decision_id: decision["decision_id"]) unless decision["allow"]
          end
        end
      end
    end
  end
end
