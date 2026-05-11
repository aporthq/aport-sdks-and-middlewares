require "test_helper"
require "stringio"

class APortMiddlewareTest < Minitest::Test
  class FakeClient
    attr_reader :agent_id, :policy_id, :context

    def initialize(decision)
      @decision = decision
    end

    def verify_policy(agent_id:, policy_id:, context:, idempotency_key: nil)
      @agent_id = agent_id
      @policy_id = policy_id
      @context = context
      @decision
    end

    def verify_policy_with_passport(passport:, policy_id:, context:, idempotency_key: nil)
      verify_policy(agent_id: passport.fetch("agent_id"), policy_id: policy_id, context: context, idempotency_key: idempotency_key)
    end

    def verify_policy_with_policy_in_body(agent_or_passport:, policy:, context:, idempotency_key: nil)
      agent_id = agent_or_passport.is_a?(Hash) ? agent_or_passport.fetch("agent_id") : agent_or_passport
      verify_policy(agent_id: agent_id, policy_id: policy.fetch("id"), context: context, idempotency_key: idempotency_key)
    end
  end

  def test_allows_request_and_sets_env
    client = FakeClient.new("decision_id" => "dec_1", "allow" => true)
    app = lambda do |env|
      [200, { "Content-Type" => "application/json" }, [JSON.generate("agent_id" => env["aport.agent_id"], "decision_id" => env["aport.decision"]["decision_id"])]]
    end
    middleware = APort::SDK::Middleware.new(app, client: client, policy_id: "finance.payment.refund.v1")

    status, _headers, body = middleware.call(env("X-Agent-Passport-Id" => "agent_123", body: { amount: 20 }))

    assert_equal 200, status
    assert_includes body.join, "dec_1"
    assert_equal "agent_123", client.agent_id
    assert_equal "finance.payment.refund.v1", client.policy_id
    assert_equal 20, client.context["amount"]
  end

  def test_denies_policy_violation
    client = FakeClient.new("decision_id" => "dec_deny", "allow" => false, "reasons" => [{ "code" => "DENIED" }])
    middleware = APort::SDK::Middleware.new(lambda { |_env| [200, {}, ["ok"]] }, client: client, policy_id: "finance.payment.refund.v1")

    status, _headers, body = middleware.call(env("X-Agent-Id" => "agent_123"))

    assert_equal 403, status
    assert_includes body.join, "policy_violation"
    assert_includes body.join, "dec_deny"
  end

  def test_missing_agent_fails_closed
    client = FakeClient.new("decision_id" => "dec_1", "allow" => true)
    middleware = APort::SDK::Middleware.new(lambda { |_env| [200, {}, ["ok"]] }, client: client, policy_id: "finance.payment.refund.v1")

    status, _headers, body = middleware.call(env)

    assert_equal 401, status
    assert_includes body.join, "missing_agent_id"
  end

  private

  def env(headers = {})
    body_hash = headers.delete(:body) || {}
    raw_body = body_hash.empty? ? "" : JSON.generate(body_hash)
    {
      "PATH_INFO" => "/refunds",
      "REQUEST_PATH" => "/refunds",
      "CONTENT_TYPE" => raw_body.empty? ? nil : "application/json",
      "rack.input" => StringIO.new(raw_body)
    }.merge(headers.transform_keys { |key| "HTTP_#{key.upcase.tr("-", "_")}" })
  end
end
