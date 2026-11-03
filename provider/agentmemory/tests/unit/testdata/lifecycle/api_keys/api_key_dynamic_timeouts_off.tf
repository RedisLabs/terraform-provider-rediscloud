resource "rediscloud_agent_memory_api_key" "runtime" {
  store_id = "store-1"
  name     = "runtime"
  dynamic "timeouts" {
    for_each = terraform_data.flag.output ? [true] : []
    content { read = "7m" }
  }
}
resource "terraform_data" "flag" {
  input            = false
  triggers_replace = 1
}
