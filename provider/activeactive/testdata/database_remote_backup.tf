variable "team" {
  type = string
}

variable "owner" {
  type = string
}

variable "subscription_name" {
  type = string
}

variable "database_name" {
  type = string
}

variable "password" {
  type      = string
  sensitive = true
}

variable "bucket_name" {
  type = string
}

variable "backup_principal" {
  type = string
}

variable "interval" {
  type = string
}

variable "time_utc" {
  type    = string
  default = null
}

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }
}

provider "aws" {
  region = "us-east-1"
}

resource "aws_s3_bucket" "backup" {
  bucket        = var.bucket_name
  force_destroy = true

  tags = {
    team  = var.team
    owner = var.owner
  }
}

resource "aws_s3_bucket_policy" "backup" {
  bucket = aws_s3_bucket.backup.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "RedisCloudBackupsAccess"
        Effect    = "Allow"
        Principal = { AWS = var.backup_principal }
        Action    = ["s3:PutObject", "s3:GetObject", "s3:DeleteObject"]
        Resource  = "${aws_s3_bucket.backup.arn}/*"
      },
      {
        Sid       = "RedisCloudBackupsList"
        Effect    = "Allow"
        Principal = { AWS = var.backup_principal }
        Action    = ["s3:ListBucket", "s3:GetBucketLocation"]
        Resource  = aws_s3_bucket.backup.arn
      },
    ]
  })
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}

resource "rediscloud_active_active_subscription" "example" {
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

resource "rediscloud_active_active_subscription_database" "example" {
  subscription_id         = rediscloud_active_active_subscription.example.id
  name                    = var.database_name
  dataset_size_in_gb      = 1
  global_data_persistence = "none"
  global_password         = var.password

  override_region {
    name = "us-east-1"

    remote_backup {
      interval     = var.interval
      time_utc     = var.time_utc
      storage_type = "aws-s3"
      storage_path = "s3://${aws_s3_bucket.backup.id}/us-east-1"
    }
  }

  override_region {
    name = "us-east-2"
  }

  depends_on = [aws_s3_bucket_policy.backup]
}
