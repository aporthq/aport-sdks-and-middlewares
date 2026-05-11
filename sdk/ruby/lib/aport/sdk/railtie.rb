module APort
  module SDK
    class Railtie < ::Rails::Railtie
      initializer "aport.sdk.configure" do
        APort::SDK.configure do |config|
          config.base_url = ENV.fetch("AGENT_PASSPORT_BASE_URL", DEFAULT_BASE_URL)
          config.api_key = ENV["AGENT_PASSPORT_API_KEY"]
        end
      end

      initializer "aport.sdk.controller_helpers" do
        ActiveSupport.on_load(:action_controller) do
          include APort::SDK::Rails::ControllerHelpers
        end
      end
    end
  end
end
