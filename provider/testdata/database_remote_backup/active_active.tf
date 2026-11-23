variable "password" {
  type      = string
  sensitive = true
  default   = null
}

resource "rediscloud_active_active_subscription" "example" {
  count = var.active_active ? 1 : 0

  name              = var.name
  payment_method_id = data.rediscloud_payment_method.card.id
  cloud_provider    = "AWS"
  resource_tags = {
    team  = var.team
    owner = var.owner
  }

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
  count = var.active_active ? 1 : 0

  subscription_id         = rediscloud_active_active_subscription.example[0].id
  name                    = var.name
  dataset_size_in_gb      = 1
  global_data_persistence = "none"
  global_password         = var.password

  override_region {
    name = "us-east-1"

    dynamic "remote_backup" {
      for_each = var.interval == null ? [] : [var.interval]

      content {
        interval     = remote_backup.value
        time_utc     = var.time_utc
        storage_type = "aws-s3"
        storage_path = local.backup_path
      }
    }
  }

  override_region {
    name = "us-east-2"
  }

  depends_on = [aws_s3_bucket_policy.backup]
}
