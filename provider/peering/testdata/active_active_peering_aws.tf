locals {
  subscription_name = "__SUBSCRIPTION_NAME__"
  aws_region        = "__AWS_REGION__"
}

provider "aws" {
  region = local.aws_region
}

data "aws_caller_identity" "current" {}

resource "aws_vpc" "peering_target" {
  cidr_block           = "172.31.0.0/24" # use a CIDR that won't overlap with the subscription
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name = "${local.subscription_name}-peering-vpc"
  }
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
      region                      = local.aws_region
      networking_deployment_cidr  = "10.0.0.0/24"
      write_operations_per_second = 1000
      read_operations_per_second  = 1000
    }
  }
}

resource "rediscloud_active_active_subscription_peering" "test" {
  subscription_id    = rediscloud_active_active_subscription.example.id
  provider_name      = "AWS"
  source_region      = local.aws_region
  destination_region = local.aws_region
  aws_account_id     = data.aws_caller_identity.current.account_id
  vpc_id             = aws_vpc.peering_target.id
  vpc_cidr           = aws_vpc.peering_target.cidr_block
}
