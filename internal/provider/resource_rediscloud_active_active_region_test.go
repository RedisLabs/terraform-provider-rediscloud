package provider

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// Checks CRUDI (CREATE,READ,UPDATE,IMPORT) operations on the subscription resource.
func TestAccResourceRedisCloudActiveActiveRegion_CRUDI(t *testing.T) {

	name := acctest.RandomWithPrefix(testResourcePrefix)
	resourceName := "rediscloud_active_active_subscription.example"

	var subId int

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudActiveActiveRegion, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider", "AWS"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.memory_limit_in_gb", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.quantity", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.support_oss_cluster_api", "false"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.region.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.region.0.write_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.region.0.read_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.region.1.write_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.region.1.read_operations_per_second", "1000"),

					func(s *terraform.State) error {
						r := s.RootModule().Resources[resourceName]

						var err error
						subId, err = strconv.Atoi(r.Primary.ID)
						if err != nil {
							return err
						}

						client := testProvider.Meta().(*apiClient)
						sub, err := client.client.Subscription.Get(context.TODO(), subId)
						if err != nil {
							return err
						}

						if redis.StringValue(sub.Name) != name {
							return fmt.Errorf("unexpected name value: %s", redis.StringValue(sub.Name))
						}
						return nil
					},
				),
			},
		},
	})
}

// TF config for provisioning a new subscription.
const testAccResourceRedisCloudActiveActiveRegion = `
  
data "rediscloud_payment_method" "card" {
	card_type = "Visa"
}

resource "rediscloud_active_active_subscription" "example" {
	name = "%s"
	payment_method_id = data.rediscloud_payment_method.card.id
	cloud_provider = "AWS"

	creation_plan {
		memory_limit_in_gb = 1
		quantity = 1
		support_oss_cluster_api=false
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

resource "rediscloud_active_active_subscription_regions" "example" {
	subscription_id = rediscloud_active_active_subscription.example.id
	delete_regions = false
	region {
	  region = "us-east-1"
	  networking_deployment_cidr = "10.0.1.0/24" 
	  recreate_region = false
	  database {
		id = rediscloud_active_active_subscription_database.example_1.id
		database_name = "db_us_east_1_1"
		local_write_operations_per_second = 1000
		local_read_operations_per_second = 1000
	  }
	  database {
		id = rediscloud_active_active_subscription_database.example_2.id
		database_name = "db_us_east_1_2"
		local_write_operations_per_second = 1000
		local_read_operations_per_second = 1000
	  }
	}
	region {
	  region = "us-east-2"
	  networking_deployment_cidr = "10.0.1.0/24" 
	  recreate_region = false
	  database {
		id = rediscloud_active_active_subscription_database.example_1.id
		database_name = "db_us_east_2_1"
		local_write_operations_per_second = 1000
		local_read_operations_per_second = 1000
	  }
	  database {
		id = rediscloud_active_active_subscription_database.example_2.id
		database_name = "db_us_east_2_2"
		local_write_operations_per_second = 1000
		local_read_operations_per_second = 1000
	  }
	}
 }
 
`
