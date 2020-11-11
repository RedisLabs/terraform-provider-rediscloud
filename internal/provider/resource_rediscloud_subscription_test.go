package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceRedisCloudSubscription(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudSubscription, name, 1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("rediscloud_subscription.example", "name", name),
					resource.TestCheckResourceAttr("rediscloud_subscription.example", "cloud_provider.0.provider", "AWS"),
					resource.TestCheckResourceAttr("rediscloud_subscription.example", "cloud_provider.0.region.0.preferred_availability_zones.#", "1"),
					resource.TestCheckResourceAttrSet("rediscloud_subscription.example", "cloud_provider.0.region.0.networking_subnet_id"),
					resource.TestCheckResourceAttr("rediscloud_subscription.example", "database.#", "1"),
					resource.TestMatchResourceAttr("rediscloud_subscription.example", "database.0.db_id", regexp.MustCompile("^[1-9][0-9]*$")),
					resource.TestCheckResourceAttrSet("rediscloud_subscription.example", "database.0.password"),
					resource.TestCheckResourceAttr("rediscloud_subscription.example", "database.0.name", "tf-database"),
					resource.TestCheckResourceAttr("rediscloud_subscription.example", "database.0.memory_limit_in_gb", "1"),
					// TODO use API to check that the subscription/database exist
				),
			},
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudSubscription, name, 2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("rediscloud_subscription.example", "name", name),
					resource.TestCheckResourceAttr("rediscloud_subscription.example", "database.#", "1"),
					resource.TestCheckResourceAttr("rediscloud_subscription.example", "database.0.memory_limit_in_gb", "2"),
					// TODO use API to check that the subscription/database exist
				),
			},
		},
	})
}

const testAccResourceRedisCloudSubscription = `
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

resource "rediscloud_subscription" "example" {

  name = "%s"
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage = "ram"
  persistent_storage_encryption = false

  cloud_provider {
    provider = "AWS"
    cloud_account_id = "16566"
    region {
      region = "eu-west-1"
      networking_deployment_cidr = "10.0.0.0/24"
    }
  }

  database {
    name = "tf-database"
    protocol = "redis"
    memory_limit_in_gb = %d
    support_oss_cluster_api = true
    data_persistence = "none"
    replication = false
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 10000
  }
}
`
