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

resource "rediscloud_subscription" "example" {
  name              = local.subscription_name
  payment_method    = "credit-card"
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage    = "ram"

  allowlist {
    cidrs              = ["192.168.0.0/16"]
    security_group_ids = []
  }

  cloud_provider {
    provider         = data.rediscloud_cloud_account.account.provider_type
    cloud_account_id = data.rediscloud_cloud_account.account.id
    region {
      region                       = local.aws_region
      networking_deployment_cidr   = "10.0.0.0/24"
      preferred_availability_zones = ["${local.aws_region}a"]
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
  create_duration = "60s"
}

data "rediscloud_transit_gateway_invitations" "test" {
  subscription_id = rediscloud_subscription.example.id

  depends_on = [time_sleep.wait_for_invitation]
}

locals {
  matching_invitation = one([
    for inv in data.rediscloud_transit_gateway_invitations.test.invitations :
    inv if inv.name == local.subscription_name
  ])
}

resource "rediscloud_transit_gateway_invitation_acceptor" "test" {
  subscription_id   = rediscloud_subscription.example.id
  tgw_invitation_id = local.matching_invitation.id
  action            = "accept"
}

resource "time_sleep" "wait_for_acceptance" {
  depends_on      = [rediscloud_transit_gateway_invitation_acceptor.test]
  create_duration = "30s"
}

data "rediscloud_transit_gateway" "test" {
  subscription_id = rediscloud_subscription.example.id
  aws_tgw_uid     = aws_ec2_transit_gateway.test.id

  depends_on = [time_sleep.wait_for_acceptance]
}

resource "rediscloud_transit_gateway_attachment" "test" {
  subscription_id = rediscloud_subscription.example.id
  tgw_id          = data.rediscloud_transit_gateway.test.tgw_id
  cidrs           = ["10.10.20.0/24"]
}
