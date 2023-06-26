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

func TestAccResourceRedisCloudActiveActiveSubscriptionRegions_CRUDI(t *testing.T) {
	subName := acctest.RandomWithPrefix(testResourcePrefix) + "-regions-test"
	dbName := acctest.RandomWithPrefix(testResourcePrefix) + "-regions" + "-db"
	dbPass := acctest.RandString(20)
	resourceName := "rediscloud_active_active_subscription_regions.example"

	var subId int

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: providerFactories,
		CheckDestroy:             testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudCreateActiveActiveRegion, subName, dbName, dbPass),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "region.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "region.2.region", "eu-west-2"),
					resource.TestCheckResourceAttr(resourceName, "region.2.networking_deployment_cidr", "10.2.0.0/24"),
					resource.TestCheckResourceAttr(resourceName, "region.2.database.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "region.2.database.0.database_name", dbName),
					resource.TestCheckResourceAttr(resourceName, "region.2.database.0.local_write_operations_per_second", "1500"),
					resource.TestCheckResourceAttr(resourceName, "region.2.database.0.local_read_operations_per_second", "1500"),

					func(s *terraform.State) error {
						r := s.RootModule().Resources[resourceName]

						var err error
						subId, err = strconv.Atoi(r.Primary.ID)
						if err != nil {
							return err
						}

						client := sdkProvider.Meta().(*apiClient)
						sub, err := client.client.Subscription.Get(context.TODO(), subId)
						if err != nil {
							return err
						}

						if redis.StringValue(sub.Name) != subName {
							return fmt.Errorf("unexpected name value: %s", redis.StringValue(sub.Name))
						}
						return nil
					},
				),
			},
			{
				// Checks region re-created correctly
				Config: fmt.Sprintf(testAccResourceRedisCloudReCreateActiveActiveRegion, subName, dbName, dbPass),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "region.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "region.2.region", "eu-west-2"),
					resource.TestCheckResourceAttr(resourceName, "region.2.networking_deployment_cidr", "10.3.0.0/24"),
					resource.TestCheckResourceAttr(resourceName, "region.2.database.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "region.2.database.0.database_name", dbName),
					resource.TestCheckResourceAttr(resourceName, "region.2.database.0.local_write_operations_per_second", "1500"),
					resource.TestCheckResourceAttr(resourceName, "region.2.database.0.local_read_operations_per_second", "1500"),
				),
			},
			{
				// Checks region DB updated correctly
				Config: fmt.Sprintf(testAccResourceRedisCloudUpdateDBActiveActiveRegion, subName, dbName, dbPass),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "region.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "region.2.region", "eu-west-2"),
					resource.TestCheckResourceAttr(resourceName, "region.2.networking_deployment_cidr", "10.3.0.0/24"),
					resource.TestCheckResourceAttr(resourceName, "region.2.database.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "region.2.database.0.database_name", dbName),
					resource.TestCheckResourceAttr(resourceName, "region.2.database.0.local_write_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(resourceName, "region.2.database.0.local_read_operations_per_second", "1000"),
				),
			},
			{
				// Checks region deleted correctly
				Config: fmt.Sprintf(testAccResourceRedisCloudDeleteActiveActiveRegion, subName, dbName, dbPass),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "region.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "region.0.region", "us-east-1"),
					resource.TestCheckResourceAttr(resourceName, "region.1.region", "us-east-2"),
				),
			},
			{
				// Checks region re-created correctly
				Config: fmt.Sprintf(testAccResourceRedisCloudRemoveAndCreateSameTimeActiveActiveRegion, subName, dbName, dbPass),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "region.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "region.0.region", "us-east-1"),
					resource.TestCheckResourceAttr(resourceName, "region.0.networking_deployment_cidr", "10.0.0.0/24"),
					resource.TestCheckResourceAttr(resourceName, "region.1.region", "eu-west-1"),
					resource.TestCheckResourceAttr(resourceName, "region.1.networking_deployment_cidr", "10.2.0.0/24"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"delete_regions"},
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
		region {
			region = "us-east-1"
			networking_deployment_cidr = "10.0.0.0/24"
			write_operations_per_second = 1000
			read_operations_per_second = 1000
		}
		region {
			region = "us-east-2"
			networking_deployment_cidr = "10.1.0.0/24"
			write_operations_per_second = 1000
			read_operations_per_second = 1000
		}
	}
}

resource "rediscloud_active_active_subscription_database" "example" {
    subscription_id = rediscloud_active_active_subscription.example.id
    name = "%s"
    memory_limit_in_gb = 3
    support_oss_cluster_api = false 
    external_endpoint_for_oss_cluster_api = false
    
    // OPTIONAL
    global_data_persistence = "none"
    global_password = "%s" 
    global_alert {
		name = "dataset-size"
		value = 40
	}
} 

`

// TF config for provisioning a new region.
const testAccResourceRedisCloudCreateActiveActiveRegion = testAARegionsBoilerplate + `

resource "rediscloud_active_active_subscription_regions" "example" {
	subscription_id = rediscloud_active_active_subscription.example.id
	delete_regions = false
	region {
	  region = "us-east-1"
	  networking_deployment_cidr = "10.0.0.0/24"
	  recreate_region = false
	  database {
		database_id = rediscloud_active_active_subscription_database.example.db_id
		database_name = rediscloud_active_active_subscription_database.example.name
		local_write_operations_per_second = 1000
		local_read_operations_per_second = 1000
	  }
	}
	region {
	  region = "us-east-2"
	  networking_deployment_cidr = "10.1.0.0/24"
	  recreate_region = false
	  database {
		database_id = rediscloud_active_active_subscription_database.example.db_id
		database_name = rediscloud_active_active_subscription_database.example.name
		local_write_operations_per_second = 1000
		local_read_operations_per_second = 1000
	  }
	}
	region {
		region = "eu-west-2"
		networking_deployment_cidr = "10.2.0.0/24" 
		recreate_region = false
		database {
		  database_id = rediscloud_active_active_subscription_database.example.db_id
		  database_name = rediscloud_active_active_subscription_database.example.name
		  local_write_operations_per_second = 1500
		  local_read_operations_per_second = 1500
		}
	  }
 }
 
`

// TF config for re-creating a region
const testAccResourceRedisCloudReCreateActiveActiveRegion = testAARegionsBoilerplate + `

resource "rediscloud_active_active_subscription_regions" "example" {
	subscription_id = rediscloud_active_active_subscription.example.id
	delete_regions = true
	region {
	  region = "us-east-1"
	  networking_deployment_cidr = "10.0.0.0/24"
	  recreate_region = false
	  database {
		database_id = rediscloud_active_active_subscription_database.example.db_id
		database_name = rediscloud_active_active_subscription_database.example.name
		local_write_operations_per_second = 1000
		local_read_operations_per_second = 1000
	  }
	}
	region {
	  region = "us-east-2"
	  networking_deployment_cidr = "10.1.0.0/24"
	  recreate_region = false
	  database {
		database_id = rediscloud_active_active_subscription_database.example.db_id
		database_name = rediscloud_active_active_subscription_database.example.name
		local_write_operations_per_second = 1000
		local_read_operations_per_second = 1000
	  }
	}
	region {
		region = "eu-west-2"
		networking_deployment_cidr = "10.3.0.0/24" 
		recreate_region = true
		database {
		  database_id = rediscloud_active_active_subscription_database.example.db_id
		  database_name = rediscloud_active_active_subscription_database.example.name
		  local_write_operations_per_second = 1500
		  local_read_operations_per_second = 1500
		}
	  }
 }
 
`

// TF config for updating DB of a region
const testAccResourceRedisCloudUpdateDBActiveActiveRegion = testAARegionsBoilerplate + `

resource "rediscloud_active_active_subscription_regions" "example" {
	subscription_id = rediscloud_active_active_subscription.example.id
	delete_regions = false
	region {
	  region = "us-east-1"
	  networking_deployment_cidr = "10.0.0.0/24"
	  recreate_region = false
	  database {
		database_id = rediscloud_active_active_subscription_database.example.db_id
		database_name = rediscloud_active_active_subscription_database.example.name
		local_write_operations_per_second = 1000
		local_read_operations_per_second = 1000
	  }
	}
	region {
	  region = "us-east-2"
	  networking_deployment_cidr = "10.1.0.0/24"
	  recreate_region = false
	  database {
		database_id = rediscloud_active_active_subscription_database.example.db_id
		database_name = rediscloud_active_active_subscription_database.example.name
		local_write_operations_per_second = 1000
		local_read_operations_per_second = 1000
	  }
	}
	region {
		region = "eu-west-2"
		networking_deployment_cidr = "10.3.0.0/24" 
		recreate_region = false
		database {
		  database_id = rediscloud_active_active_subscription_database.example.db_id
		  database_name = rediscloud_active_active_subscription_database.example.name
		  local_write_operations_per_second = 1000
		  local_read_operations_per_second = 1000
		}
	  }
 }
 
`

// TF config for deleting a region
const testAccResourceRedisCloudDeleteActiveActiveRegion = testAARegionsBoilerplate + `

resource "rediscloud_active_active_subscription_regions" "example" {
	subscription_id = rediscloud_active_active_subscription.example.id
	delete_regions = true
	region {
	  region = "us-east-1"
	  networking_deployment_cidr = "10.0.0.0/24"
	  recreate_region = false
	  database {
		database_id = rediscloud_active_active_subscription_database.example.db_id
		database_name = rediscloud_active_active_subscription_database.example.name
		local_write_operations_per_second = 1000
		local_read_operations_per_second = 1000
	  }
	}
	region {
	  region = "us-east-2"
	  networking_deployment_cidr = "10.1.0.0/24"
	  recreate_region = false
	  database {
		database_id = rediscloud_active_active_subscription_database.example.db_id
		database_name = rediscloud_active_active_subscription_database.example.name
		local_write_operations_per_second = 1000
		local_read_operations_per_second = 1000
	  }
	}
 }
 
`

// TF config for deleting a region
const testAccResourceRedisCloudRemoveAndCreateSameTimeActiveActiveRegion = testAARegionsBoilerplate + `

resource "rediscloud_active_active_subscription_regions" "example" {
	subscription_id = rediscloud_active_active_subscription.example.id
	delete_regions = true
	region {
	  region = "us-east-1"
	  networking_deployment_cidr = "10.0.0.0/24"
	  recreate_region = false
	  database {
		database_id = rediscloud_active_active_subscription_database.example.db_id
		database_name = rediscloud_active_active_subscription_database.example.name
		local_write_operations_per_second = 1000
		local_read_operations_per_second = 1000
	  }
	}
	region {
	  region = "eu-west-1"
	  networking_deployment_cidr = "10.2.0.0/24"
	  recreate_region = false
	  database {
		database_id = rediscloud_active_active_subscription_database.example.db_id
		database_name = rediscloud_active_active_subscription_database.example.name
		local_write_operations_per_second = 1000
		local_read_operations_per_second = 1000
	  }
	}
 }
 
`
