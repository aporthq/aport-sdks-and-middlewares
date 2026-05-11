require "json"
require "stringio"

module APort
  module SDK
    class Middleware
      DEFAULT_SKIP_PATHS = ["/health", "/metrics", "/status"].freeze

      def initialize(app, client: SDK.client, policy_id: nil, agent_id: nil, fail_closed: true, skip_paths: DEFAULT_SKIP_PATHS)
        @app = app
        @client = client
        @policy_id = policy_id
        @agent_id = agent_id
        @fail_closed = fail_closed
        @skip_paths = skip_paths
      end

      def call(env)
        return @app.call(env) if skip?(env)

        body = json_body(env)
        passport = body["passport"].is_a?(Hash) ? body["passport"] : nil
        policy = body["policy"].is_a?(Hash) ? body["policy"] : nil
        agent_id = @agent_id || passport && passport["agent_id"] || env["HTTP_X_AGENT_PASSPORT_ID"] || env["HTTP_X_AGENT_ID"]

        unless agent_id || passport
          return json_response(401, "missing_agent_id", "Agent ID is required. Provide X-Agent-Passport-Id, X-Agent-Id, or body.passport.") if @fail_closed

          return @app.call(env)
        end

        context = body.reject { |key, _value| key == "passport" || key == "policy" }
        decision = verify(agent_id, passport, policy, context, env["HTTP_IDEMPOTENCY_KEY"])
        if decision && decision["allow"] == false
          return json_response(
            403,
            "policy_violation",
            "Policy violation",
            "agent_id" => agent_id,
            "policy_id" => @policy_id || policy && policy["id"],
            "decision_id" => decision["decision_id"],
            "reasons" => decision["reasons"]
          )
        end

        env["aport.agent_id"] = agent_id || passport["agent_id"]
        env["aport.decision"] = decision if decision
        @app.call(env)
      rescue Error => error
        json_response(error.status.zero? ? 502 : error.status, "api_error", error.message, "reasons" => error.reasons)
      end

      private

      def verify(agent_id, passport, policy, context, idempotency_key)
        if policy && passport
          @client.verify_policy_with_policy_in_body(
            agent_or_passport: passport,
            policy: policy,
            context: context,
            idempotency_key: idempotency_key
          )
        elsif policy
          @client.verify_policy_with_policy_in_body(
            agent_or_passport: agent_id,
            policy: policy,
            context: context,
            idempotency_key: idempotency_key
          )
        elsif passport
          @client.verify_policy_with_passport(
            passport: passport,
            policy_id: @policy_id,
            context: context,
            idempotency_key: idempotency_key
          )
        elsif @policy_id
          @client.verify_policy(
            agent_id: agent_id,
            policy_id: @policy_id,
            context: context,
            idempotency_key: idempotency_key
          )
        else
          @client.get_passport_view(agent_id: agent_id)
          nil
        end
      end

      def skip?(env)
        path = env["PATH_INFO"] || env["REQUEST_PATH"] || ""
        @skip_paths.any? { |skip_path| path.start_with?(skip_path) }
      end

      def json_body(env)
        input = env["rack.input"]
        return {} unless input

        raw = input.read
        input.rewind if input.respond_to?(:rewind)
        return {} if raw.nil? || raw.empty?
        return {} unless (env["CONTENT_TYPE"] || "").include?("json")

        JSON.parse(raw)
      rescue JSON::ParserError
        {}
      end

      def json_response(status, error, message, extra = {})
        body = JSON.generate({ "error" => error, "message" => message }.merge(extra.compact))
        [status, { "Content-Type" => "application/json" }, [body]]
      end
    end
  end
end
