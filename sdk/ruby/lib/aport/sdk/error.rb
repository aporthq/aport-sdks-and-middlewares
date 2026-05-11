module APort
  module SDK
    class Error < StandardError
      attr_reader :status, :reasons, :decision_id, :server_timing, :raw_response

      def initialize(status:, reasons: nil, decision_id: nil, server_timing: nil, raw_response: nil)
        @status = status
        @reasons = reasons || []
        @decision_id = decision_id
        @server_timing = server_timing
        @raw_response = raw_response
        super(build_message)
      end

      private

      def build_message
        return "APort request failed: #{status}" if reasons.empty?

        messages = reasons.map { |reason| reason["message"] || reason[:message] || reason["code"] || reason[:code] }.compact
        return "APort request failed: #{status}" if messages.empty?

        "APort request failed: #{status} #{messages.join(", ")}"
      end
    end

    AportError = Error
  end
end
