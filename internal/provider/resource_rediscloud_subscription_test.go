package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceRedisCloudSubscription(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-test")

	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudSubscription, name, 1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("rediscloud_subscription.example", "name", name),
					resource.TestCheckResourceAttr("rediscloud_subscription.example", "database.#", "1"),
				),
			},
			//{
			//	Config: fmt.Sprintf(testAccResourceRedisCloudSubscription, name, 2),
			//	Check: resource.ComposeTestCheckFunc(
			//		resource.TestCheckResourceAttr("rediscloud_subscription.example", "name", name),
			//		resource.TestCheckResourceAttr("rediscloud_subscription.example", "database.#", "1"),
			//	),
			//},
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

  cloud_providers {
    provider = "AWS"
    cloud_account_id = 16566
    regions {
      region = "eu-west-1"
      networking_deployment_cidr = "10.0.0.0/24"
    }
  }

  database {
    name = "tf-example-database"
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
