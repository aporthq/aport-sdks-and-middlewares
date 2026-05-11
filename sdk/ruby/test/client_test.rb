require "test_helper"
require "webrick"

class APortClientTest < Minitest::Test
  def with_server(&block)
    requests = []
    server = WEBrick::HTTPServer.new(
      Port: 0,
      Logger: WEBrick::Log.new(File::NULL),
      AccessLog: []
    )
    handler = nil
    server.mount_proc("/") do |request, response|
      requests << request
      handler.call(request, response)
    end

    thread = Thread.new { server.start }
    base_url = "http://127.0.0.1:#{server.config[:Port]}"
    sleep 0.01 until server.status == :Running
    block.call(base_url, requests, proc { |&new_handler| handler = new_handler })
  ensure
    server.shutdown if server
    thread.join if thread
  end

  def test_verify_policy_builds_request_and_returns_decision
    with_server do |base_url, requests, on_request|
      on_request.call do |request, response|
        assert_equal "/api/verify/policy/finance.payment.refund.v1", request.path
        assert_equal "Bearer test-key", request["authorization"]
        assert_equal "idem-1", request["idempotency-key"]

        body = JSON.parse(request.body)
        assert_equal "agent_123", body.dig("context", "agent_id")
        assert_equal "finance.payment.refund.v1", body.dig("context", "policy_id")
        assert_equal 1000, body.dig("context", "amount")

        response["server-timing"] = "app;dur=9"
        response.body = JSON.generate("decision" => { "decision_id" => "dec_123", "allow" => true })
      end

      client = APort::SDK::Client.new(base_url: base_url, api_key: "test-key")
      decision = client.verify_policy(
        agent_id: "agent_123",
        policy_id: "finance.payment.refund.v1",
        context: { amount: 1000 },
        idempotency_key: "idem-1"
      )

      assert_equal 1, requests.length
      assert_equal true, decision["allow"]
      assert_equal "dec_123", decision["decision_id"]
      assert_equal "app;dur=9", decision.dig("_meta", "serverTiming")
    end
  end

  def test_api_error_preserves_reasons
    with_server do |base_url, _requests, on_request|
      on_request.call do |_request, response|
        response.status = 403
        response.body = JSON.generate(
          "decision_id" => "dec_deny",
          "reasons" => [{ "code" => "DENIED", "message" => "not allowed" }]
        )
      end

      client = APort::SDK::Client.new(base_url: base_url)
      error = assert_raises(APort::SDK::Error) do
        client.verify_policy(agent_id: "agent_123", policy_id: "finance.payment.refund.v1")
      end

      assert_equal 403, error.status
      assert_equal "dec_deny", error.decision_id
      assert_equal "DENIED", error.reasons.first["code"]
    end
  end

  def test_decision_token_and_jwks_cache
    jwks_calls = 0
    with_server do |base_url, _requests, on_request|
      on_request.call do |request, response|
        case request.path
        when "/api/verify/token/finance.payment.refund.v1"
          response.body = JSON.generate("token" => "token-123")
        when "/api/verify/token/validate"
          response.body = JSON.generate("decision" => { "decision_id" => "dec_token", "allow" => true })
        when "/jwks.json"
          jwks_calls += 1
          response.body = JSON.generate("keys" => [{ "kty" => "RSA", "kid" => "kid-1" }])
        else
          response.status = 404
        end
      end

      client = APort::SDK::Client.new(base_url: base_url)
      token = client.get_decision_token(agent_id: "agent_123", policy_id: "finance.payment.refund.v1")
      assert_equal "token-123", token

      2.times do
        assert_equal true, client.validate_decision_token_local(token: token)["allow"]
      end
      assert_equal 1, jwks_calls
    end
  end
end
