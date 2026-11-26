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
  count             = 2
  name              = "${var.subscription_name}-${count.index}"
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
      networking_deployment_cidr = "10.${count.index}.0.0/24"
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

data "rediscloud_subscriptions" "example" {
  depends_on = [rediscloud_subscription.example]
}

output "lists_expected_subscriptions" {
  value = alltrue([
    for expected in rediscloud_subscription.example :
    length([for sub in data.rediscloud_subscriptions.example.subscriptions : sub if sub.id == expected.id && sub.name == expected.name]) == 1
  ])
}

locals {
  subscription_ids = [for sub in data.rediscloud_subscriptions.example.subscriptions : tonumber(sub.id)]
}

output "subscription_ids_in_order" {
  value = alltrue([for i, id in local.subscription_ids : local.subscription_ids[i - 1] < id if i > 0])
}
