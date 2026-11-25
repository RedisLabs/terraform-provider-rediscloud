resource "rediscloud_iris_identity_api_key" "key" {
  name        = "invalid-key"
  grants_json = "[]"
}
