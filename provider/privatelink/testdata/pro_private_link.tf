variable "subscription_name" {
  type = string
}

variable "cloud_account_name" {
  type = string
}

variable "share_name" {
  type = string
}

variable "database_password" {
  type = string
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}

data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type            = "AWS"
  name                     = var.cloud_account_name
}

resource "rediscloud_subscription" "pro_subscription" {
  name              = var.subscription_name
  payment_method_id = data.rediscloud_payment_method.card.id

  cloud_provider {
    provider         = data.rediscloud_cloud_account.account.provider_type
    cloud_account_id = data.rediscloud_cloud_account.account.id
    region {
      region                       = "eu-west-1"
      networking_deployment_cidr   = "10.0.0.0/24"
      preferred_availability_zones = ["eu-west-1a"]
    }
  }

  creation_plan {
    dataset_size_in_gb           = 1
    quantity                     = 1
    replication                  = true
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 20000
  }
}

## this will give some connections and databases in the output
resource "rediscloud_subscription_database" "pro_database" {
  subscription_id              = rediscloud_subscription.pro_subscription.id
  name                         = "db"
  memory_limit_in_gb           = 1
  password                     = var.database_password
  protocol                     = "redis"
  data_persistence             = "none"
  throughput_measurement_by    = "operations-per-second"
  throughput_measurement_value = 10000
}


resource "rediscloud_private_link" "pro_private_link" {
  subscription_id = rediscloud_subscription.pro_subscription.id
  share_name      = var.share_name

  principal {
    principal       = "123456789012"
    principal_type  = "aws_account"
    principal_alias = "principal 1"
  }

  principal {
    principal       = "234567890123"
    principal_type  = "aws_account"
    principal_alias = "principal 2"
  }

  depends_on = [rediscloud_subscription_database.pro_database]
}

data "rediscloud_private_link" "pro_private_link" {
  subscription_id = rediscloud_private_link.pro_private_link.subscription_id
}
