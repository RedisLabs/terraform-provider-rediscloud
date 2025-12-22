locals {
  subscription_name     = "__SUBSCRIPTION_NAME__"
  subscription_cidr     = "__SUBSCRIPTION_CIDR__"
  peering_source_region = "__PEERING_SOURCE_REGION__"
  peering_dest_region   = "__PEERING_DEST_REGION__"
  aws_account_id        = "__AWS_ACCOUNT_ID__"
  vpc_id                = "__VPC_ID__"
  vpc_cidr              = "__VPC_CIDR__"
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}

resource "rediscloud_active_active_subscription" "example" {
  name              = local.subscription_name
  payment_method_id = data.rediscloud_payment_method.card.id
  cloud_provider    = "AWS"

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
      region                      = local.peering_source_region
      networking_deployment_cidr  = local.subscription_cidr
      write_operations_per_second = 1000
      read_operations_per_second  = 1000
    }
  }
}

resource "rediscloud_active_active_subscription_peering" "test" {
  subscription_id    = rediscloud_active_active_subscription.example.id
  provider_name      = "AWS"
  source_region      = local.peering_source_region
  destination_region = local.peering_dest_region
  aws_account_id     = local.aws_account_id
  vpc_id             = local.vpc_id
  vpc_cidr           = local.vpc_cidr
}
