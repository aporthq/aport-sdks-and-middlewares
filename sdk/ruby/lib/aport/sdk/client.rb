require "json"
require "net/http"
require "uri"

module APort
  module SDK
    class Client
      USER_AGENT = "aport-sdk-ruby/0.1.0"

      def initialize(base_url: ENV.fetch("AGENT_PASSPORT_BASE_URL", DEFAULT_BASE_URL), api_key: ENV["AGENT_PASSPORT_API_KEY"], timeout: DEFAULT_TIMEOUT, http: nil)
        @base_url = base_url.to_s.sub(%r{/+\z}, "")
        @api_key = api_key
        @timeout = timeout
        @http = http
        @jwks_cache = nil
        @jwks_cache_expires_at = nil
      end

      def verify_policy(agent_id:, policy_id:, context: {}, idempotency_key: nil, passport: nil, policy: nil)
        path = policy ? "/api/verify/policy/IN_BODY" : "/api/verify/policy/#{escape_path(policy_id)}"
        body = build_policy_request_body(
          agent_id: agent_id,
          policy_id: policy_id,
          context: context,
          idempotency_key: idempotency_key,
          passport: passport,
          policy: policy
        )
        normalize_decision(post(path, body, idempotency_key: idempotency_key))
      end

      def verify_policy_with_passport(passport:, policy_id:, context: {}, idempotency_key: nil)
        verify_policy(
          agent_id: passport.fetch("agent_id"),
          policy_id: policy_id,
          context: context,
          idempotency_key: idempotency_key,
          passport: passport
        )
      end

      def verify_policy_with_policy_in_body(agent_or_passport:, policy:, context: {}, idempotency_key: nil)
        agent_id = agent_or_passport.is_a?(Hash) ? agent_or_passport.fetch("agent_id") : agent_or_passport
        verify_policy(
          agent_id: agent_id,
          policy_id: policy.fetch("id"),
          context: context,
          idempotency_key: idempotency_key,
          passport: agent_or_passport.is_a?(Hash) ? agent_or_passport : nil,
          policy: policy
        )
      end

      def get_decision_token(agent_id:, policy_id:, context: {})
        response = post("/api/verify/token/#{escape_path(policy_id)}", {
          "agent_id" => agent_id,
          "context" => stringify_keys(context)
        })
        response.fetch("token")
      end

      def validate_decision_token(token:)
        normalize_decision(post("/api/verify/token/validate", { "token" => token }))
      end

      def validate_decision_token_local(token:)
        get_jwks
        validate_decision_token(token: token)
      rescue Error => error
        raise Error.new(
          status: 401,
          reasons: [{ "code" => "INVALID_TOKEN", "message" => "Token validation failed" }],
          raw_response: error.raw_response
        )
      end

      def get_passport_view(agent_id:)
        get("/api/passports/#{escape_path(agent_id)}/verify_view")
      end

      def get_jwks
        if @jwks_cache && @jwks_cache_expires_at && Time.now < @jwks_cache_expires_at
          return @jwks_cache
        end

        @jwks_cache = get("/jwks.json")
        @jwks_cache_expires_at = Time.now + 300
        @jwks_cache
      rescue Error => error
        raise Error.new(
          status: 500,
          reasons: [{ "code" => "JWKS_FETCH_FAILED", "message" => "Failed to fetch JWKS" }],
          raw_response: error.raw_response
        )
      end

      private

      def get(path)
        request(Net::HTTP::Get, path)
      end

      def post(path, body, idempotency_key: nil)
        request(Net::HTTP::Post, path, body: body, idempotency_key: idempotency_key)
      end

      def request(klass, path, body: nil, idempotency_key: nil)
        uri = URI.parse("#{@base_url}#{path}")
        request = klass.new(uri)
        request["Accept"] = "application/json"
        request["User-Agent"] = USER_AGENT
        request["Authorization"] = "Bearer #{@api_key}" if @api_key && !@api_key.empty?
        request["Idempotency-Key"] = idempotency_key if idempotency_key

        if body
          request["Content-Type"] = "application/json"
          request.body = JSON.generate(body)
        end

        response = perform_request(uri, request)
        parsed = parse_json(response.body)
        server_timing = response["server-timing"]

        unless response.code.to_i.between?(200, 299)
          raise Error.new(
            status: response.code.to_i,
            reasons: parsed["reasons"],
            decision_id: parsed["decision_id"],
            server_timing: server_timing,
            raw_response: response.body
          )
        end

        parsed["_meta"] = { "serverTiming" => server_timing } if server_timing
        parsed
      rescue Timeout::Error => error
        raise Error.new(status: 408, reasons: [{ "code" => "TIMEOUT", "message" => error.message }])
      rescue SystemCallError, SocketError => error
        raise Error.new(status: 0, reasons: [{ "code" => "NETWORK_ERROR", "message" => error.message }])
      end

      def perform_request(uri, request)
        return @http.request(uri, request) if @http

        Net::HTTP.start(
          uri.host,
          uri.port,
          use_ssl: uri.scheme == "https",
          open_timeout: @timeout,
          read_timeout: @timeout
        ) do |http|
          http.request(request)
        end
      end

      def build_policy_request_body(agent_id:, policy_id:, context:, idempotency_key:, passport:, policy:)
        request_context = {}
        request_context["agent_id"] = agent_id if agent_id
        request_context["policy_id"] = policy_id if policy_id
        request_context["idempotency_key"] = idempotency_key if idempotency_key
        stringify_keys(context).each { |key, value| request_context[key] = value }

        body = { "context" => request_context }
        body["passport"] = passport if passport
        body["policy"] = policy if policy
        body
      end

      def normalize_decision(response)
        decision = response["decision"] || response
        if response["_meta"] && !decision["_meta"]
          decision["_meta"] = response["_meta"]
        end
        decision
      end

      def stringify_keys(hash)
        (hash || {}).each_with_object({}) do |(key, value), output|
          output[key.to_s] = value
        end
      end

      def parse_json(body)
        return {} if body.nil? || body.empty?

        JSON.parse(body)
      rescue JSON::ParserError
        {}
      end

      def escape_path(value)
        URI.encode_www_form_component(value.to_s)
      end
    end
  end
end
