module APort
  module SDK
    class PolicyVerifier
      def initialize(client = SDK.client)
        @client = client
      end

      def verify_refund(agent_id:, context:, idempotency_key: nil)
        @client.verify_policy(
          agent_id: agent_id,
          policy_id: "finance.payment.refund.v1",
          context: context,
          idempotency_key: idempotency_key
        )
      end

      def verify_release(agent_id:, context:, idempotency_key: nil)
        @client.verify_policy(
          agent_id: agent_id,
          policy_id: "code.release.publish.v1",
          context: context,
          idempotency_key: idempotency_key
        )
      end

      def verify_data_export(agent_id:, context:, idempotency_key: nil)
        @client.verify_policy(
          agent_id: agent_id,
          policy_id: "data.export.create.v1",
          context: context,
          idempotency_key: idempotency_key
        )
      end
    end
  end
end
