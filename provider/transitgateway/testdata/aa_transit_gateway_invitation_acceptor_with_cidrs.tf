locals {
  subscription_name = "%s"
  database_name     = "%s"
  database_password = "%s"
  aws_region        = "%s"
}

provider "aws" {
  region = local.aws_region
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
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

resource "rediscloud_active_active_subscription_database" "test" {
  subscription_id         = rediscloud_active_active_subscription.test.id
  name                    = local.database_name
  dataset_size_in_gb      = 1
  global_data_persistence = "none"
  global_password         = local.database_password

  override_region {
    name = local.aws_region
  }

  override_region {
    name = "us-east-2"
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
  principal          = rediscloud_active_active_subscription.test.aws_account_id
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

data "aws_ec2_transit_gateway_vpc_attachments" "pending" {
  filter {
    name   = "state"
    values = ["pending-acceptance"]
  }

  filter {
    name   = "transit-gateway-id"
    values = [aws_ec2_transit_gateway.test.id]
  }

  depends_on = [data.rediscloud_active_active_transit_gateway.test]
}

resource "aws_ec2_transit_gateway_vpc_attachment_accepter" "test" {
  transit_gateway_attachment_id = data.aws_ec2_transit_gateway_vpc_attachments.pending.ids[0]

  tags = {
    Name = local.subscription_name
  }
}

resource "rediscloud_active_active_transit_gateway_attachment" "test" {
  subscription_id = rediscloud_active_active_subscription.test.id
  region_id       = local.region_id
  tgw_id          = data.rediscloud_active_active_transit_gateway.test.tgw_id
  cidrs           = ["10.10.20.0/24"]

  depends_on = [aws_ec2_transit_gateway_vpc_attachment_accepter.test]
}
