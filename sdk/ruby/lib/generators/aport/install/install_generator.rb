require "rails/generators"

module Aport
  module Generators
    class InstallGenerator < Rails::Generators::Base
      desc "Creates an APort initializer for Rails applications"

      def create_initializer
        create_file "config/initializers/aport.rb", <<~RUBY
          require "aport/sdk"

          APort::SDK.configure do |config|
            config.base_url = ENV.fetch("AGENT_PASSPORT_BASE_URL", "https://api.aport.io")
            config.api_key = ENV["AGENT_PASSPORT_API_KEY"]
            config.timeout = 0.8
          end

          # Protect all routes with a policy:
          # Rails.application.config.middleware.use(
          #   APort::SDK::Middleware,
          #   policy_id: "finance.payment.refund.v1"
          # )
        RUBY
      end
    end
  end
end
