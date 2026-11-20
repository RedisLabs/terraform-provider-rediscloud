resource "rediscloud_iris_identity_provider" "example" {
  tenant              = "your-tenant"
  name                = "Corporate"
  issuer              = "https://issuer.example.com"
  audience            = "api://redis-iris"
  jwks_uri            = "https://issuer.example.com/.well-known/jwks.json"
  required_algorithms = ["RS256"]
  token_exchange_json = jsonencode({
    grants        = "SUBJECT_GRANTS"
    irisAudiences = ["api://redis-iris"]
  })
}
