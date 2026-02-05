locals {
  card_type         = "__CARD_TYPE__"
  last_four_numbers = "__LAST_FOUR_NUMBERS__"
}

data "rediscloud_payment_method" "card" {
  card_type         = local.card_type
  last_four_numbers = local.last_four_numbers
}
