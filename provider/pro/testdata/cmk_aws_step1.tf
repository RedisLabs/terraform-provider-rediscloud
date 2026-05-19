variable "name" {
  type = string
}

provider "aws" {
  region = "us-east-1"
}

data "aws_caller_identity" "current" {}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}

# Create KMS key in step 1
resource "aws_kms_key" "cmk" {
  description             = "rediscloud-cmk-test-${var.name}"
  deletion_window_in_days = 7
  enable_key_rotation     = false

  tags = {
    managed-by = "terraform"
    purpose    = "rediscloud-cmk-test"
  }
}

# Create subscription (enters encryption_key_pending state)
# No customer_managed_key blocks yet, so no circular dependency on KMS key ARN
resource "rediscloud_subscription" "example" {
  name                         = var.name
  payment_method               = "credit-card"
  payment_method_id            = data.rediscloud_payment_method.card.id
  memory_storage               = "ram"
  customer_managed_key_enabled = true

  cloud_provider {
    provider = "AWS"
    region {
      region                     = "us-east-1"
      networking_deployment_cidr = "10.0.1.0/24"
    }
  }

  creation_plan {
    dataset_size_in_gb           = 1
    quantity                     = 1
    replication                  = false
    support_oss_cluster_api      = false
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 10000
  }
}

# Grant the Redis IAM role access to the KMS key via key policy.
# Policy is a separate resource so it can reference the role ARN
# returned by the subscription, avoiding the inline-policy ordering deadlock.
resource "aws_kms_key_policy" "cmk" {
  key_id = aws_kms_key.cmk.id

  policy = jsonencode({
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
        Principal = { AWS = rediscloud_subscription.example.customer_managed_key_aws_role_arn }
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
