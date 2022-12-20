package provider

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourceRedisCloudActiveActiveRegion_CRUDI(t *testing.T) {

	name := "test-sub-1" //acctest.RandomWithPrefix(testResourcePrefix)
	resourceName := "rediscloud_active_active_subscription_regions.example"

	var subId int

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudCrateActiveActiveRegion),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "subscription_id", "151945"),
					resource.TestCheckResourceAttr(resourceName, "region.#", "3"),

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
			{
				// Checks region re-created correctly
				Config: fmt.Sprintf(testAccResourceRedisCloudUpdateActiveActiveRegion),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "subscription_id", "151945"),
					resource.TestCheckResourceAttr(resourceName, "region.#", "2"),
				),
			},
		},
	})
}

const testAARegionsBoilerplate = `
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

`

// TF config for provisioning a new subscription.
const testAccResourceRedisCloudCrateActiveActiveRegion = testAARegionsBoilerplate + `

resource "rediscloud_active_active_subscription_regions" "example" {
	subscription_id = rediscloud_subscription.example.id
	delete_regions = false
	region {
	  region = "us-east-1"
	  networking_deployment_cidr = "10.0.0.0/24" 
	  recreate_region = false
	  database {
		id = "7839"
		database_name = "test-db-1"
		local_write_operations_per_second = 1000
		local_read_operations_per_second = 1000
	  }
	}
	region {
	  region = "eu-west-1"
	  networking_deployment_cidr = "10.1.0.0/24" 
	  recreate_region = false
	  database {
		id = "7839"
		database_name = "test-db-1"
		local_write_operations_per_second = 1000
		local_read_operations_per_second = 1000
	  }
	}
	region {
		region = "eu-west-2"
		networking_deployment_cidr = "10.2.0.0/24" 
		recreate_region = false
		database {
		  id = "7839"
		  database_name = "test-db-1"
		  local_write_operations_per_second = 1500
		  local_read_operations_per_second = 1500
		}
	  }
 }
 
`

// TF config for provisioning a new subscription.
const testAccResourceRedisCloudUpdateActiveActiveRegion = `

resource "rediscloud_active_active_subscription_regions" "example" {
	subscription_id = 151945
	delete_regions = true
	region {
	  region = "us-east-1"
	  networking_deployment_cidr = "10.0.0.0/24" 
	  recreate_region = false
	  database {
		id = "7839"
		database_name = "test-db-1"
		local_write_operations_per_second = 1000
		local_read_operations_per_second = 1000
	  }
	}
	region {
	  region = "eu-west-1"
	  networking_deployment_cidr = "10.1.0.0/24" 
	  recreate_region = false
	  database {
		id = "7839"
		database_name = "test-db-1"
		local_write_operations_per_second = 1000
		local_read_operations_per_second = 1000
	  }
	}
	region {
		region = "eu-west-2"
		networking_deployment_cidr = "10.3.0.0/24" 
		recreate_region = true
		database {
		  id = "7839"
		  database_name = "test-db-1"
		  local_write_operations_per_second = 1500
		  local_read_operations_per_second = 1500
		}
	  }
 }
 
`
