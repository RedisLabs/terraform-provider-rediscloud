locals {
  rediscloud_subscription_name = "%s"
  rediscloud_cloud_account = "%s"
  rediscloud_private_link_share_name = "%s"
  rediscloud_database_password = "%s"
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}


resource "rediscloud_active_active_subscription" "aa_subscription" {
  name              = local.rediscloud_subscription_name
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

resource "rediscloud_active_active_subscription_database" "aa_database" {
  subscription_id         = rediscloud_active_active_subscription.aa_subscription.id
  name                    = "db"
  memory_limit_in_gb      = 1
  global_data_persistence = "aof-every-1-second"
  global_password         = local.rediscloud_database_password
}

data "rediscloud_active_active_subscription_regions" "aa_regions_info" {
  subscription_name = rediscloud_active_active_subscription.aa_subscription.name
  depends_on        = [rediscloud_active_active_subscription_database.aa_database]
}


resource "rediscloud_active_active_private_link" "aa_private_link" {
  subscription_id = rediscloud_active_active_subscription.aa_subscription.id
  region_id       = data.rediscloud_active_active_subscription_regions.aa_regions_info.regions[0].region_id
  share_name      = "private_link testing"

  principal {
    principal       = "123456789012"
    principal_type  = "aws_account"
    principal_alias = "terraform test aws account"
  }
  principal {
    principal       = "688576139039"
    principal_type  = "aws_account"
    principal_alias = "terraform test aws account 2"
  }
}

data "rediscloud_active_active_private_link" "aa_private_link" {
  subscription_id = rediscloud_active_active_private_link.aa_private_link.subscription_id
  region_id       = data.rediscloud_active_active_subscription_regions.aa_regions_info.regions[0].region_id
}

# data "rediscloud_private_link_endpoint_script" "endpoint_script" {
#   subscription_id = rediscloud_private_link.private_link.subscription_id
# }
