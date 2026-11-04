variable "subscription_name" {
  type = string
}

variable "share_name" {
  type = string
}

variable "team" {
  type = string
}

variable "owner" {
  type = string
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}

resource "rediscloud_active_active_subscription" "aa_subscription" {
  resource_tags = {
    team  = var.team
    owner = var.owner
  }

  name              = var.subscription_name
  payment_method    = "credit-card"
  payment_method_id = data.rediscloud_payment_method.card.id
  cloud_provider    = "AWS"

  creation_plan {
    memory_limit_in_gb = 1
    quantity           = 1
    region {
      region                      = "eu-west-1"
      networking_deployment_cidr  = "192.168.0.0/24"
      write_operations_per_second = 1000
      read_operations_per_second  = 1000
    }
    region {
      region                      = "eu-west-2"
      networking_deployment_cidr  = "10.0.1.0/24"
      write_operations_per_second = 1000
      read_operations_per_second  = 1000
    }
  }
}

data "rediscloud_active_active_subscription_regions" "aa_regions_info" {
  subscription_name = rediscloud_active_active_subscription.aa_subscription.name
}

resource "rediscloud_active_active_private_link" "aa_private_link" {
  subscription_id = rediscloud_active_active_subscription.aa_subscription.id
  region_id       = data.rediscloud_active_active_subscription_regions.aa_regions_info.regions[0].region_id
  share_name      = var.share_name

  principal {
    principal       = "123456789012"
    principal_type  = "aws_account"
    principal_alias = "terraform test aws account"
  }
}

data "rediscloud_active_active_private_link" "aa_private_link" {
  subscription_id = rediscloud_active_active_private_link.aa_private_link.subscription_id
  region_id       = rediscloud_active_active_private_link.aa_private_link.region_id
}
