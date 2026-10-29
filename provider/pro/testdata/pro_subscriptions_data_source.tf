variable "team" {
  type = string
}

variable "owner" {
  type = string
}

variable "subscription_name" {
  type = string
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}

resource "rediscloud_subscription" "example" {
  name              = var.subscription_name
  payment_method    = "credit-card"
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage    = "ram"

  cloud_provider {
    resource_tags = {
      team  = var.team
      owner = var.owner
    }

    provider = "AWS"

    region {
      region                     = "eu-west-1"
      networking_deployment_cidr = "10.0.0.0/24"
    }
  }

  creation_plan {
    memory_limit_in_gb           = 1
    quantity                     = 1
    replication                  = false
    support_oss_cluster_api      = false
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 1000
    modules                      = []
  }
}

# Filtering by the resource's own name makes the data source depend on the subscription,
# so it reads back the just-created one in a single step (no depends_on needed).
data "rediscloud_subscriptions" "example" {
  filters {
    name = rediscloud_subscription.example.name
  }
}
