variable "subscription_name" {
  type = string
}

variable "cloud_provider" {
  type = string
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}

resource "rediscloud_active_active_subscription" "example" {
  name              = var.subscription_name
  payment_method_id = data.rediscloud_payment_method.card.id

  maintenance_windows {
    mode = "automatic"
  }

  # NOTE: cloud_provider is placed after the nested blocks on purpose.
  # terraform-plugin-testing's HasProviderBlock detects top-level provider
  # configuration via a regex that scans for the substring "provider" followed
  # by whitespace, an identifier, more whitespace, and an opening brace. If
  # cloud_provider sits immediately above a nested block, the trailing
  # "provider" of cloud_provider plus the following block opener satisfies the
  # regex and the framework thinks the file declares a provider configuration,
  # failing the step with "Providers must only be specified either at the
  # TestCase or TestStep level". Keeping cloud_provider last (with no opening
  # brace afterwards) sidesteps the false positive.
  cloud_provider = var.cloud_provider
}

data "rediscloud_active_active_subscription" "example" {
  name = rediscloud_active_active_subscription.example.name
}
