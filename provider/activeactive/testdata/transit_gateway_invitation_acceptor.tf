locals {
  cloud_account_name        = "%s"
  subscription_name         = "%s"
  aws_region                = "%s"
  rediscloud_aws_account_id = "%s"
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}

data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type            = "AWS"
  name                     = local.cloud_account_name
}

data "rediscloud_regions" "aws" {
  provider_name = "AWS"
}

resource "rediscloud_active_active_subscription" "test" {
  name              = local.subscription_name
  payment_method_id = data.rediscloud_payment_method.card.id
  cloud_provider    = "AWS"

  creation_plan {
    dataset_size_in_gb = 1
    quantity           = 1
    region {
      region                       = local.aws_region
      networking_deployment_cidr   = "192.168.0.0/24"
      write_operations_per_second  = 1000
      read_operations_per_second   = 1000
    }
    region {
      region                       = "us-east-2"
      networking_deployment_cidr   = "10.0.1.0/24"
      write_operations_per_second  = 1000
      read_operations_per_second   = 1000
    }
  }
}

locals {
  region_id = one([
    for r in data.rediscloud_regions.aws.regions :
    r.region_id if r.name == local.aws_region
  ])
}

resource "aws_ec2_transit_gateway" "test" {
  description = local.subscription_name
  tags = {
    Name = local.subscription_name
  }
}

resource "aws_ram_resource_share" "test" {
  name                      = local.subscription_name
  allow_external_principals = true
}

resource "aws_ram_resource_association" "test" {
  resource_share_arn = aws_ram_resource_share.test.arn
  resource_arn       = aws_ec2_transit_gateway.test.arn
}

resource "aws_ram_principal_association" "test" {
  resource_share_arn = aws_ram_resource_share.test.arn
  principal          = local.rediscloud_aws_account_id
}

resource "time_sleep" "wait_for_invitation" {
  depends_on      = [aws_ram_principal_association.test]
  create_duration = "120s"
}

data "rediscloud_active_active_transit_gateway_invitations" "test" {
  subscription_id = rediscloud_active_active_subscription.test.id
  region_id       = local.region_id

  depends_on = [time_sleep.wait_for_invitation]
}

locals {
  matching_invitation = one([
    for inv in data.rediscloud_active_active_transit_gateway_invitations.test.invitations :
    inv if inv.name == local.subscription_name
  ])
}

resource "rediscloud_active_active_transit_gateway_invitation_acceptor" "test" {
  subscription_id   = rediscloud_active_active_subscription.test.id
  region_id         = local.region_id
  tgw_invitation_id = local.matching_invitation.id
  action            = "accept"
}

resource "time_sleep" "wait_for_acceptance" {
  depends_on      = [rediscloud_active_active_transit_gateway_invitation_acceptor.test]
  create_duration = "30s"
}

data "rediscloud_active_active_transit_gateway" "test" {
  subscription_id = rediscloud_active_active_subscription.test.id
  region_id       = local.region_id
  aws_tgw_uid     = aws_ec2_transit_gateway.test.id

  depends_on = [time_sleep.wait_for_acceptance]
}

resource "rediscloud_active_active_transit_gateway_attachment" "test" {
  subscription_id = rediscloud_active_active_subscription.test.id
  region_id       = local.region_id
  tgw_id          = data.rediscloud_active_active_transit_gateway.test.tgw_id
}
