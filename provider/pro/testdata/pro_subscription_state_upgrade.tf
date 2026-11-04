variable "team" {
  type = string
}

variable "owner" {
  type = string
}

locals {
  rediscloud_subscription_name = "%s"
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}

resource "rediscloud_subscription" "example" {
  name              = local.rediscloud_subscription_name
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage    = "ram"

  cloud_provider {
    resource_tags = {
      team  = var.team
      owner = var.owner
    }

    provider = "AWS"
    region {
      region                      = "eu-west-1"
      multiple_availability_zones = false
      networking_deployment_cidr  = "10.0.0.0/24"
    }
  }

  creation_plan {
    dataset_size_in_gb           = 1
    quantity                     = 1
    replication                  = false
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 1000
  }
}
