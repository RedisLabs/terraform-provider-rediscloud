terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }
}

variable "subscription_name" {
  type = string
}

variable "aws_region" {
  type = string
}

variable "team" {
  type = string
}

variable "owner" {
  type = string
}

variable "lookup_aws_tgw_uid" {
  type    = string
  default = null
}

variable "wait_for_tgw_timeout" {
  type    = number
  default = 600
}

provider "aws" {
  region = var.aws_region
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

    provider         = "AWS"
    cloud_account_id = "1"
    region {
      region                     = var.aws_region
      networking_deployment_cidr = "10.0.0.0/24"
    }
  }

  creation_plan {
    memory_limit_in_gb           = 1
    quantity                     = 1
    replication                  = false
    support_oss_cluster_api      = false
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 10000
    modules                      = []
  }
}

resource "aws_ec2_transit_gateway" "test" {
  description = var.subscription_name
  tags = {
    Name = var.subscription_name
  }
}

resource "aws_ram_resource_share" "test" {
  name                      = var.subscription_name
  allow_external_principals = true
}

resource "aws_ram_resource_association" "test" {
  resource_share_arn = aws_ram_resource_share.test.arn
  resource_arn       = aws_ec2_transit_gateway.test.arn
}

resource "aws_ram_principal_association" "test" {
  resource_share_arn = aws_ram_resource_share.test.arn
  principal          = rediscloud_subscription.example.cloud_provider[0].aws_account_id
}

data "rediscloud_transit_gateway_invitations" "test" {
  subscription_id              = rediscloud_subscription.example.id
  wait_for_invitations_timeout = 600

  depends_on = [
    aws_ram_resource_association.test,
    aws_ram_principal_association.test,
  ]
}

locals {
  matching_invitation = one([
    for inv in data.rediscloud_transit_gateway_invitations.test.invitations :
    inv if inv.name == var.subscription_name
  ])
}

resource "rediscloud_transit_gateway_invitation_acceptor" "test" {
  subscription_id   = rediscloud_subscription.example.id
  tgw_invitation_id = local.matching_invitation.id
  action            = "accept"
}

data "rediscloud_transit_gateway" "test" {
  subscription_id      = rediscloud_subscription.example.id
  aws_tgw_uid          = coalesce(var.lookup_aws_tgw_uid, aws_ec2_transit_gateway.test.id)
  wait_for_tgw_timeout = var.wait_for_tgw_timeout

  depends_on = [rediscloud_transit_gateway_invitation_acceptor.test]
}
