variable "card_type" {
  type = string
}

variable "last_four_numbers" {
  type = string
}

data "rediscloud_payment_method" "card" {
  card_type         = var.card_type
  last_four_numbers = var.last_four_numbers
}
