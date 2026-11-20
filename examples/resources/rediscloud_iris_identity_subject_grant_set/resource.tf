resource "rediscloud_iris_identity_subject_grant_set" "example" {
  tenant  = "your-tenant"
  issuer  = "https://issuer.example.com"
  subject = "user-123"
}
