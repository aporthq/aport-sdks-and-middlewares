require "aport/sdk/version"
require "aport/sdk/error"
require "aport/sdk/client"
require "aport/sdk/middleware"
require "aport/sdk/policy_verifier"
require "aport/sdk/rails" if defined?(Rails)

module APort
  module SDK
    DEFAULT_BASE_URL = "https://api.aport.io"
    DEFAULT_TIMEOUT = 0.8

    class << self
      attr_writer :client

      def client
        @client ||= Client.new
      end

      def configure
        yield(configuration)
        @client = Client.new(
          base_url: configuration.base_url,
          api_key: configuration.api_key,
          timeout: configuration.timeout
        )
      end

      def configuration
        @configuration ||= Configuration.new
      end
    end

    class Configuration
      attr_accessor :base_url, :api_key, :timeout

      def initialize
        @base_url = ENV.fetch("AGENT_PASSPORT_BASE_URL", DEFAULT_BASE_URL)
        @api_key = ENV["AGENT_PASSPORT_API_KEY"]
        @timeout = DEFAULT_TIMEOUT
      end
    end
  end
end

require "aport/sdk/railtie" if defined?(Rails::Railtie)
