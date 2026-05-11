locals {
  name = "__NAME__"
}

provider "aws" {
  region = "us-east-1"
}

provider "aws" {
  alias  = "use2"
  region = "us-east-2"
}

data "aws_caller_identity" "current" {}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}

resource "aws_kms_key" "cmk_primary" {
  description             = "rediscloud-cmk-test-${local.name}"
  multi_region            = true
  deletion_window_in_days = 7
  enable_key_rotation     = false

  tags = {
    managed-by = "terraform"
    purpose    = "rediscloud-cmk-test"
  }
}

resource "aws_kms_replica_key" "cmk_replica" {
  provider                = aws.use2
  description             = "rediscloud-cmk-test-${local.name}-replica"
  primary_key_arn         = aws_kms_key.cmk_primary.arn
  deletion_window_in_days = 7

  tags = {
    managed-by = "terraform"
    purpose    = "rediscloud-cmk-test"
  }
}

locals {
  cmk_policy = jsonencode({
    Version = "2012-10-17"
    Id      = "rediscloud-cmk-key-policy"
    Statement = [
      {
        Sid       = "EnableIAMUserPermissions"
        Effect    = "Allow"
        Principal = { AWS = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:root" }
        Action    = "kms:*"
        Resource  = "*"
      },
      {
        Sid       = "AllowRedisCloudCMK"
        Effect    = "Allow"
        Principal = { AWS = rediscloud_active_active_subscription.example.customer_managed_key_aws_role_arn }
        Action = [
          "kms:Encrypt",
          "kms:Decrypt",
          "kms:GenerateDataKey*",
          "kms:DescribeKey",
          "kms:ReEncrypt*",
          "kms:CreateGrant",
        ]
        Resource = "*"
      },
    ]
  })
}

resource "aws_kms_key_policy" "cmk_primary" {
  key_id = aws_kms_key.cmk_primary.id
  policy = local.cmk_policy
}

resource "aws_kms_key_policy" "cmk_replica" {
  provider = aws.use2
  key_id   = aws_kms_replica_key.cmk_replica.id
  policy   = local.cmk_policy
}

# Step 2: subscription now references both KMS key ARNs (per-region),
# transitioning out of encryption_key_pending.
resource "rediscloud_active_active_subscription" "example" {
  name                         = local.name
  payment_method               = "credit-card"
  payment_method_id            = data.rediscloud_payment_method.card.id
  customer_managed_key_enabled = true
  cloud_provider               = "AWS"

  customer_managed_key {
    resource_name = aws_kms_key.cmk_primary.arn
    region        = "us-east-1"
  }

  customer_managed_key {
    resource_name = aws_kms_replica_key.cmk_replica.arn
    region        = "us-east-2"
  }

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
      region                      = "us-east-2"
      networking_deployment_cidr  = "10.0.1.0/24"
      write_operations_per_second = 1000
      read_operations_per_second  = 1000
    }
  }
}
