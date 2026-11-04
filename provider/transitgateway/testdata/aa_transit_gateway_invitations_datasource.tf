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

variable "database_name" {
  type = string
}

variable "database_password" {
  type      = string
  sensitive = true
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

provider "aws" {
  region = var.aws_region
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}

data "rediscloud_regions" "aws" {
  provider_name = "AWS"
}

resource "rediscloud_active_active_subscription" "test" {
  resource_tags = {
    team  = var.team
    owner = var.owner
  }

  name              = var.subscription_name
  payment_method_id = data.rediscloud_payment_method.card.id
  cloud_provider    = "AWS"

  creation_plan {
    dataset_size_in_gb = 1
    quantity           = 1
    region {
      region                      = var.aws_region
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
}

resource "rediscloud_active_active_subscription_database" "test" {
  subscription_id         = rediscloud_active_active_subscription.test.id
  name                    = var.database_name
  dataset_size_in_gb      = 1
  global_data_persistence = "none"
  global_password         = var.database_password

  override_region {
    name = var.aws_region
  }

  override_region {
    name = "us-east-2"
  }
}

locals {
  region_id = one([
    for r in data.rediscloud_regions.aws.regions :
    r.region_id if r.name == var.aws_region
  ])
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
  principal          = rediscloud_active_active_subscription.test.aws_account_id
}

data "rediscloud_active_active_transit_gateway_invitations" "test" {
  subscription_id              = rediscloud_active_active_subscription.test.id
  region_id                    = local.region_id
  wait_for_invitations_timeout = 600

  depends_on = [
    aws_ram_resource_association.test,
    aws_ram_principal_association.test,
    rediscloud_active_active_subscription_database.test,
  ]
}
