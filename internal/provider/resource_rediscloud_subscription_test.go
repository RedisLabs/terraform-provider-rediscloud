package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceRedisCloudSubscription(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceRedisCloudSubsctiption,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("rediscloud_subscription.example", "name", "Example Subscription"),
					resource.TestCheckResourceAttr("rediscloud_subscription.example", "cloud_providers.0.provider", "AWS"),
					resource.TestCheckResourceAttr("rediscloud_subscription.example", "cloud_providers.0.cloud_account_id", "1"),
					resource.TestCheckResourceAttr("rediscloud_subscription.example", "cloud_providers.0.regions.0.region", "eu-west-1"),
					resource.TestCheckResourceAttr("rediscloud_subscription.example", "cloud_providers.0.regions.0.multiple_availability_zones", "false"),
					resource.TestCheckResourceAttr("rediscloud_subscription.example", "cloud_providers.0.networking.0.deployment_cidr", "192.168.0.0/24"),
				),
			},
		},
	})
}

const testAccResourceRedisCloudSubsctiption = `
resource "rediscloud_subscription" "example" {

	name = "TF Example Subscription"
	dry_run = false
	payment_method_id = 16971
	memory_storage = "ram"
	persistent_storage_encryption = false

	cloud_providers {
		provider = "AWS"
		cloud_account_id = 16566
		regions {
			region = "eu-west-1"
			networking_deployment_cidr = "10.0.0.0/24"
		}
	}

	databases {
		name = "tf-example-database"
		protocol = "redis"
		memory_limit_in_gb = 1
		support_oss_cluster_api = true
		data_persistence = "none"
		replication = false
		throughput_measurement_by = "operations-per-second"
		throughput_measurement_value = 10000
		quantity = 1
	}
}
`
