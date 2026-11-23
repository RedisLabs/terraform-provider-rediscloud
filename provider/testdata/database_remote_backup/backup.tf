variable "active_active" {
  type    = bool
  default = false
}

variable "name" {
  type = string
}

variable "team" {
  type = string
}

variable "owner" {
  type = string
}

variable "backup_principal" {
  type = string

  validation {
    condition     = can(regex("^arn:aws:iam::[0-9]{12}:(root|role/[A-Za-z0-9_+=,.@/-]+)$", var.backup_principal))
    error_message = "Use an AWS account root ARN or a specific AWS IAM role ARN."
  }
}

variable "interval" {
  type    = string
  default = null
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
  bucket        = "${lower(var.name)}-backup"
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
        Effect    = "Allow"
        Principal = { AWS = var.backup_principal }
        Action    = ["s3:PutObject", "s3:GetObject", "s3:DeleteObject"]
        Resource  = "${aws_s3_bucket.backup.arn}/backup/*"
      },
      {
        Effect    = "Allow"
        Principal = { AWS = var.backup_principal }
        Action    = ["s3:ListBucket", "s3:GetBucketLocation"]
        Resource  = aws_s3_bucket.backup.arn
      },
    ]
  })
}

locals {
  backup_path = "s3://${aws_s3_bucket.backup.id}/backup"
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}
