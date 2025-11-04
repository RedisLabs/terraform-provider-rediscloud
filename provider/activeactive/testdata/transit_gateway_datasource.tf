data "rediscloud_payment_method" "card" {
	card_type = "Visa"
	last_four_numbers = "5556"
}

data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type = "AWS"
  name = "%s"
}

resource "rediscloud_active_active_subscription" "test" {
  name = "%s"
  payment_method_id = data.rediscloud_payment_method.card.id
  cloud_provider = "AWS"

  creation_plan {
    dataset_size_in_gb = 1
    quantity = 1
    region {
      region = "us-east-1"
      networking_deployment_cidr = "192.168.0.0/24"
      write_operations_per_second = 1000
      read_operations_per_second = 1000
    }
    region {
      region = "us-east-2"
      networking_deployment_cidr = "10.0.1.0/24"
      write_operations_per_second = 1000
      read_operations_per_second = 1000
    }
  }
}

data "rediscloud_active_active_transit_gateway" "test" {
  subscription_id = rediscloud_active_active_subscription.test.id
  region_id = tolist(rediscloud_active_active_subscription.test.regions)[0].region_id
  aws_tgw_uid = "%s"
}
