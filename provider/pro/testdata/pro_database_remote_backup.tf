variable "team" {
  type = string
}

variable "owner" {
  type = string
}

variable "subscription_name" {
  type = string
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

variable "list_backups" {
  type    = bool
  default = false
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
  region = "eu-west-1"
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

resource "rediscloud_subscription" "example" {
  name              = var.subscription_name
  payment_method_id = data.rediscloud_payment_method.card.id

  cloud_provider {
    resource_tags = {
      team  = var.team
      owner = var.owner
    }

    provider = "AWS"
    region {
      region                     = "eu-west-1"
      networking_deployment_cidr = "10.0.0.0/24"
    }
  }

  creation_plan {
    dataset_size_in_gb           = 1
    quantity                     = 1
    replication                  = false
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 1000
  }
}

resource "rediscloud_subscription_database" "example" {
  subscription_id              = rediscloud_subscription.example.id
  name                         = var.subscription_name
  dataset_size_in_gb           = 1
  data_persistence             = "none"
  throughput_measurement_by    = "operations-per-second"
  throughput_measurement_value = 1000

  remote_backup {
    interval     = var.interval
    time_utc     = var.time_utc
    storage_type = "aws-s3"
    storage_path = "s3://${aws_s3_bucket.backup.id}/pro"
  }

  depends_on = [aws_s3_bucket_policy.backup]
}

data "aws_s3_objects" "backups" {
  count  = var.list_backups ? 1 : 0
  bucket = aws_s3_bucket.backup.id
  prefix = "pro"
}
