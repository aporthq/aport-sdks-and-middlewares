Gem::Specification.new do |spec|
  spec.name = "aporthq-sdk-ruby"
  spec.version = "0.1.0"
  spec.authors = ["APort"]
  spec.email = ["support@aport.io"]

  spec.summary = "Ruby and Rails SDK for APort policy verification"
  spec.description = "Thin APort API client with Rack/Rails middleware, helpers, and install generator."
  spec.homepage = "https://aport.io"
  spec.license = "MIT"

  spec.required_ruby_version = ">= 2.6.0"
  spec.files = Dir["lib/**/*", "README.md", "LICENSE"]
  spec.require_paths = ["lib"]

  spec.add_development_dependency "rake", "~> 12.0"
  spec.add_development_dependency "minitest", "~> 5.0"
end
