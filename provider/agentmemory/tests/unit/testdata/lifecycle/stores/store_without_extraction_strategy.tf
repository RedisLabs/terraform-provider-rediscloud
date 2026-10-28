variable "name" {
  type = string
}

resource "rediscloud_agent_memory" "example" {
  name        = var.name
  database_id = 123

}
