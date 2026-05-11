# APort Ruby SDK

Ruby and Rails package for APort policy verification.

## Install

```ruby
gem "aporthq-sdk-ruby"
```

```bash
bundle install
bin/rails generate aport:install
```

## Client

```ruby
require "aport/sdk"

client = APort::SDK::Client.new(
  base_url: "https://api.aport.io",
  api_key: ENV["AGENT_PASSPORT_API_KEY"],
  timeout: 0.8
)

decision = client.verify_policy(
  agent_id: "agent_123",
  policy_id: "finance.payment.refund.v1",
  context: {
    amount: 1000,
    currency: "USD",
    order_id: "order_123"
  },
  idempotency_key: "refund-order-123"
)

puts decision["decision_id"] if decision["allow"]
```

## Rails Middleware

Add middleware in `config/initializers/aport.rb`:

```ruby
Rails.application.config.middleware.use(
  APort::SDK::Middleware,
  policy_id: "finance.payment.refund.v1"
)
```

The middleware reads `X-Agent-Passport-Id`, `X-Agent-Id`, or `body.passport.agent_id`, sends request context to APort, and stores these values in the Rack/Rails environment:

- `env["aport.agent_id"]`
- `env["aport.decision"]`

Requests with `allow=false` return `403` with the APort denial reasons.

## Controller Helpers

Rails controllers get:

```ruby
aport_agent_id
aport_decision
verify_aport_policy!(
  "finance.payment.refund.v1",
  context: { amount: 1000, currency: "USD" }
)
```

## Inline Passport and Policy

The middleware and client support local passport and inline policy evaluation:

```ruby
client.verify_policy_with_policy_in_body(
  agent_or_passport: { "agent_id" => "agent_123", "claims" => { "role" => "buyer" } },
  policy: { "id" => "custom.policy.v1", "requires_capabilities" => ["payments.charge"] },
  context: { amount: 1000 }
)
```

## Development

```bash
bundle exec rake test
```
