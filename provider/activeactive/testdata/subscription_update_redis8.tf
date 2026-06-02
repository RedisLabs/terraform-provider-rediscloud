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

  creation_plan {
    memory_limit_in_gb = 1
    quantity           = 1
    region {
      region                      = "us-east-1"
      networking_deployment_cidr  = "192.168.0.0/24"
      write_operations_per_second = 1000
      read_operations_per_second  = 1000
    }
    region {
      region                      = "us-east-2"
      networking_deployment_cidr  = "10.0.1.0/24"
      write_operations_per_second = 1000
      read_operations_per_second  = 1000
    }
  }

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

resource "rediscloud_active_active_subscription_database" "example" {
  subscription_id         = rediscloud_active_active_subscription.example.id
  name                    = var.subscription_name
  redis_version           = "8.2"
  dataset_size_in_gb      = 1
  global_data_persistence = "aof-every-1-second"
  global_password         = "some-random-pass-2"
  global_source_ips       = ["192.168.0.0/16"]
  global_alert {
    name  = "dataset-size"
    value = 40
  }

  global_enable_default_user = false

  override_region {
    name                             = "us-east-1"
    override_global_data_persistence = "none"
    override_global_password         = "region-specific-password"
    override_global_alert {
      name  = "dataset-size"
      value = 60
    }
  }

  override_region {
    name                       = "us-east-2"
    enable_default_user        = false
    override_global_source_ips = ["192.10.0.0/16"]
  }

  tags = {
    "environment" = "production"
    "cost_center" = "0700"
  }
}
