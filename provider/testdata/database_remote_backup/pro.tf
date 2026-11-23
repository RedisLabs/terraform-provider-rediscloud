resource "rediscloud_subscription" "example" {
  count = var.active_active ? 0 : 1

  name              = var.name
  payment_method_id = data.rediscloud_payment_method.card.id

  cloud_provider {
    provider = "AWS"
    resource_tags = {
      team  = var.team
      owner = var.owner
    }
    region {
      region                     = "us-east-1"
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
  count = var.active_active ? 0 : 1

  subscription_id              = rediscloud_subscription.example[0].id
  name                         = var.name
  dataset_size_in_gb           = 1
  data_persistence             = "none"
  throughput_measurement_by    = "operations-per-second"
  throughput_measurement_value = 1000

  remote_backup {
    interval     = var.interval
    time_utc     = var.time_utc
    storage_type = "aws-s3"
    storage_path = local.backup_path
  }

  depends_on = [aws_s3_bucket_policy.backup]
}
