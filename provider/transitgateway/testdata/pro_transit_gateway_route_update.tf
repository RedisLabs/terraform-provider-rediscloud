locals {
  subscription_name = "%s"
  aws_region        = "%s"
}

provider "aws" {
  region = local.aws_region
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}

resource "rediscloud_subscription" "example" {
  name              = local.subscription_name
  payment_method    = "credit-card"
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage    = "ram"

  cloud_provider {
    provider         = "AWS"
    cloud_account_id = "1"
    region {
      region                     = local.aws_region
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
  principal          = rediscloud_subscription.example.cloud_provider[0].aws_account_id
}

resource "time_sleep" "wait_for_invitation" {
  depends_on      = [aws_ram_principal_association.test]
  create_duration = "120s"
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
}

resource "time_sleep" "wait_for_attachment" {
  depends_on      = [rediscloud_transit_gateway_attachment.test]
  create_duration = "120s"
}

resource "aws_ec2_transit_gateway_vpc_attachment_accepter" "test" {
  transit_gateway_attachment_id = rediscloud_transit_gateway_attachment.test.attachment_uid

  tags = {
    Name = local.subscription_name
  }

  depends_on = [time_sleep.wait_for_attachment]
}

resource "rediscloud_transit_gateway_route" "test" {
  subscription_id = rediscloud_subscription.example.id
  tgw_id          = data.rediscloud_transit_gateway.test.tgw_id
  cidrs           = ["10.10.20.0/24", "10.10.21.0/24"]

  depends_on = [aws_ec2_transit_gateway_vpc_attachment_accepter.test]
}
