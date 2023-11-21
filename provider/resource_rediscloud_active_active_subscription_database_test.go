package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// Checks CRUDI (CREATE,READ,UPDATE,IMPORT) operations on the database resource.
func TestAccResourceRedisCloudActiveActiveSubscriptionDatabase_CRUDI(t *testing.T) {

	subscriptionName := acctest.RandomWithPrefix(testResourcePrefix) + "-subscription"
	name := acctest.RandomWithPrefix(testResourcePrefix) + "-database"
	password := acctest.RandString(20)
	resourceName := "rediscloud_active_active_subscription_database.example"
	subscriptionResourceName := "rediscloud_active_active_subscription.example"

	var subId int

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			// Test database creation
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudActiveActiveSubscriptionDatabase, subscriptionName, name, password),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "memory_limit_in_gb", "3"),
					resource.TestCheckResourceAttr(resourceName, "support_oss_cluster_api", "false"),
					resource.TestCheckResourceAttr(resourceName, "global_data_persistence", "none"),
					resource.TestCheckResourceAttr(resourceName, "external_endpoint_for_oss_cluster_api", "false"),
					resource.TestCheckResourceAttr(resourceName, "global_password", password),
					resource.TestCheckResourceAttr(resourceName, "enable_tls", "false"),
					resource.TestCheckResourceAttr(resourceName, "data_eviction", "volatile-lru"),
					resource.TestCheckResourceAttr(resourceName, "global_alert.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "global_alert.0.name", "dataset-size"),
					resource.TestCheckResourceAttr(resourceName, "global_alert.0.value", "40"),
					resource.TestCheckResourceAttr(resourceName, "global_source_ips.#", "2"),

					resource.TestCheckResourceAttr(resourceName, "override_region.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "override_region.0.name", "us-east-1"),
					resource.TestCheckResourceAttr(resourceName, "override_region.0.override_global_data_persistence", "aof-every-write"),
					resource.TestCheckResourceAttr(resourceName, "override_region.0.override_global_password", "region-specific-password"),
					// check override region alert block
					resource.TestCheckResourceAttr(resourceName, "override_region.0.override_global_alert.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "override_region.0.override_global_alert.0.name", "dataset-size"),
					resource.TestCheckResourceAttr(resourceName, "override_region.0.override_global_alert.0.value", "42"),
					resource.TestCheckResourceAttr(resourceName, "override_region.0.override_global_source_ips.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "override_region.0.override_global_source_ips.0", "192.175.0.0/16"),

					// Check that global values are used for the second region where no override is set
					resource.TestCheckResourceAttr(resourceName, "override_region.1.name", "us-east-2"),
					resource.TestCheckResourceAttr(resourceName, "override_region.1.override_global_data_persistence", ""),
					resource.TestCheckResourceAttr(resourceName, "override_region.1.override_global_password", ""),
					resource.TestCheckResourceAttr(resourceName, "override_region.1.override_global_alert.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "override_region.1.override_source_ips.#", "0"),

					// Test databases exist
					func(s *terraform.State) error {
						r := s.RootModule().Resources[subscriptionResourceName]

						var err error
						subId, err = strconv.Atoi(r.Primary.ID)
						if err != nil {
							return fmt.Errorf("couldn't parse the subscription ID: %s", redis.StringValue(&r.Primary.ID))
						}

						client := testProvider.Meta().(*apiClient)
						sub, err := client.client.Subscription.Get(context.TODO(), subId)
						if err != nil {
							return err
						}

						if redis.StringValue(sub.Name) != subscriptionName {
							return fmt.Errorf("unexpected name value: %s", redis.StringValue(sub.Name))
						}

						listDb := client.client.Database.List(context.TODO(), subId)
						if listDb.Next() != true {
							return fmt.Errorf("no database found: %s", listDb.Err())
						}
						if listDb.Err() != nil {
							return listDb.Err()
						}

						return nil
					},
				),
			},
			// Test database is updated successfully, including updates to both global and local alerts
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudActiveActiveSubscriptionDatabaseUpdate, subscriptionName, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "memory_limit_in_gb", "1"),
					resource.TestCheckResourceAttr(resourceName, "support_oss_cluster_api", "true"),
					resource.TestCheckResourceAttr(resourceName, "external_endpoint_for_oss_cluster_api", "true"),
					resource.TestCheckResourceAttr(resourceName, "global_data_persistence", "aof-every-1-second"),
					resource.TestCheckResourceAttr(resourceName, "global_password", "updated-password"),
					resource.TestCheckResourceAttr(resourceName, "global_alert.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "global_alert.0.name", "dataset-size"),
					resource.TestCheckResourceAttr(resourceName, "global_alert.0.value", "60"),

					resource.TestCheckResourceAttr(resourceName, "override_region.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "override_region.0.name", "us-east-1"),
					resource.TestCheckResourceAttr(resourceName, "override_region.0.override_global_data_persistence", "none"),
					resource.TestCheckResourceAttr(resourceName, "override_region.0.override_global_password", "password-updated"),
					resource.TestCheckResourceAttr(resourceName, "override_region.0.override_global_alert.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "override_region.0.override_global_alert.0.name", "dataset-size"),
					resource.TestCheckResourceAttr(resourceName, "override_region.0.override_global_alert.0.value", "41"),
					resource.TestCheckResourceAttr(resourceName, "override_region.0.override_global_source_ips.#", "0"),
				),
			},
			// Test database is updated, including deletion of global and local alerts
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudActiveActiveSubscriptionDatabaseUpdateNoAlerts, subscriptionName, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "memory_limit_in_gb", "1"),
					resource.TestCheckResourceAttr(resourceName, "support_oss_cluster_api", "true"),
					resource.TestCheckResourceAttr(resourceName, "external_endpoint_for_oss_cluster_api", "true"),
					resource.TestCheckResourceAttr(resourceName, "global_data_persistence", "aof-every-1-second"),
					resource.TestCheckResourceAttr(resourceName, "global_password", "updated-password"),
					resource.TestCheckResourceAttr(resourceName, "global_alert.#", "0"),

					resource.TestCheckResourceAttr(resourceName, "override_region.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "override_region.0.name", "us-east-1"),
					resource.TestCheckResourceAttr(resourceName, "override_region.0.override_global_data_persistence", "none"),
					resource.TestCheckResourceAttr(resourceName, "override_region.0.override_global_password", "password-updated"),
					resource.TestCheckResourceAttr(resourceName, "override_region.0.override_global_alert.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "override_region.0.override_global_source_ips.#", "0"),
				),
			},
			// Test that that database is imported successfully
			{
				Config:            fmt.Sprintf(testAccResourceRedisCloudActiveActiveSubscriptionDatabaseImport, subscriptionName, name),
				ResourceName:      "rediscloud_active_active_subscription_database.example",
				ImportState:       true,
				ImportStateVerify: true,
				// global and override attributes not supported as part of import
				ImportStateVerifyIgnore: []string{
					"global_data_persistence",
					"global_password",
					"global_source_ips.#",
					"global_source_ips.0",
					"override_region.#",
					"override_region.0.%",
					"override_region.0.name",
					"override_region.0.override_global_alert.#",
					"override_region.0.override_global_alert.0.%",
					"override_region.0.override_global_alert.0.name",
					"override_region.0.override_global_alert.0.value",
					"override_region.0.override_global_data_persistence",
					"override_region.0.override_global_password",
				},
			},
		},
	})
}

func TestAccResourceRedisCloudActiveActiveSubscriptionDatabase_optionalAttributes(t *testing.T) {
	// Test that attributes can be optional, either by setting them or not having them set when compared to CRUDI test
	subscriptionName := acctest.RandomWithPrefix(testResourcePrefix) + "-subscription"
	name := acctest.RandomWithPrefix(testResourcePrefix) + "-database"
	password := acctest.RandString(20)
	resourceName := "rediscloud_active_active_subscription_database.example"
	portNumber := 10101

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudActiveActiveSubscriptionDatabaseOptionalAttributes, subscriptionName, name, password, portNumber),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "port", strconv.Itoa(portNumber)),
				),
			},
		},
	})
}

func TestAccResourceRedisCloudActiveActiveSubscriptionDatabase_timeUtcRequiresValidInterval(t *testing.T) {
	name := acctest.RandomWithPrefix(testResourcePrefix)
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")
	password := acctest.RandString(20)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccResourceRedisCloudActiveActiveSubscriptionDatabaseInvalidTimeUtc, testCloudAccountName, name, password),
				ExpectError: regexp.MustCompile("unexpected value at override_region\\.\\d*\\.remote_backup\\.0\\.time_utc - time_utc can only be set when interval is either every-24-hours or every-12-hours"),
			},
		},
	})
}

const activeActiveSubscriptionBoilerplate = `
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

// Create and Read tests
// TF config for provisioning a new database
const testAccResourceRedisCloudActiveActiveSubscriptionDatabase = activeActiveSubscriptionBoilerplate + `
resource "rediscloud_active_active_subscription_database" "example" {
    subscription_id = rediscloud_active_active_subscription.example.id
    name = "%s"
    memory_limit_in_gb = 3
    support_oss_cluster_api = false 
    external_endpoint_for_oss_cluster_api = false
	enable_tls = false
    
    global_data_persistence = "none"
    global_password = "%s" 
    global_source_ips = ["192.168.0.0/16", "192.170.0.0/16"]
    global_alert {
		name = "dataset-size"
		value = 40
	}
	override_region {
		name = "us-east-1"
		override_global_data_persistence = "aof-every-write"
		override_global_source_ips = ["192.175.0.0/16"]
		override_global_password = "region-specific-password"
		override_global_alert {
			name = "dataset-size"
			value = 42
		}
	}
	override_region {
		name = "us-east-2"
	}

} 
`

// TF config for updating a database
const testAccResourceRedisCloudActiveActiveSubscriptionDatabaseUpdate = activeActiveSubscriptionBoilerplate + `
resource "rediscloud_active_active_subscription_database" "example" {
    subscription_id = rediscloud_active_active_subscription.example.id
    name = "%s"
    memory_limit_in_gb = 1
    support_oss_cluster_api = true 
    external_endpoint_for_oss_cluster_api = true
    
    global_data_persistence = "aof-every-1-second"
    global_password = "updated-password" 
    global_source_ips = ["192.170.0.0/16"]
	global_alert {
		name = "dataset-size"
		value = 60
	}

	override_region {
		name = "us-east-1"
		override_global_data_persistence = "none"
		override_global_password = "password-updated"
		override_global_alert {
			name = "dataset-size"
			value = 41
		}
	}
}
`

const testAccResourceRedisCloudActiveActiveSubscriptionDatabaseUpdateNoAlerts = activeActiveSubscriptionBoilerplate + `
resource "rediscloud_active_active_subscription_database" "example" {
    subscription_id = rediscloud_active_active_subscription.example.id
    name = "%s"
    memory_limit_in_gb = 1
    support_oss_cluster_api = true 
    external_endpoint_for_oss_cluster_api = true
    
    global_data_persistence = "aof-every-1-second"
    global_password = "updated-password" 
    global_source_ips = ["192.170.0.0/16"]

	override_region {
		name = "us-east-1"
		override_global_data_persistence = "none"
		override_global_password = "password-updated"
	}
}
`

// TF config for updating a database
const testAccResourceRedisCloudActiveActiveSubscriptionDatabaseImport = activeActiveSubscriptionBoilerplate + `
resource "rediscloud_active_active_subscription_database" "example" {
    subscription_id = rediscloud_active_active_subscription.example.id
    name = "%s"
    memory_limit_in_gb = 1
}
`

const testAccResourceRedisCloudActiveActiveSubscriptionDatabaseOptionalAttributes = `
data "rediscloud_payment_method" "card" {
	card_type = "Visa"
}

resource "rediscloud_active_active_subscription" "example" {
	name = "%s"
	payment_method_id = data.rediscloud_payment_method.card.id
	cloud_provider = "AWS"
	redis_version = "latest"

	creation_plan {
		memory_limit_in_gb = 1
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

resource "rediscloud_active_active_subscription_database" "example" {
	subscription_id = rediscloud_active_active_subscription.example.id
	name = "%s"
	memory_limit_in_gb = 3
	support_oss_cluster_api = false 
	external_endpoint_for_oss_cluster_api = false
	enable_tls = false
	global_resp_version = "resp3"

	global_data_persistence = "none"
	global_password = "%s" 
	global_source_ips = ["192.168.0.0/16", "192.170.0.0/16"]
	global_alert {
		name = "dataset-size"
		value = 40
	}
	override_region {
		name = "us-east-1"
		override_global_data_persistence = "aof-every-write"
		override_global_source_ips = ["192.175.0.0/16"]
		override_global_password = "region-specific-password"
		override_global_alert {
			name = "dataset-size"
			value = 42
		}
	}
	override_region {
		name = "us-east-2"
	}
	port = %d
} 
`

const testAccResourceRedisCloudActiveActiveSubscriptionDatabaseInvalidTimeUtc = activeActiveSubscriptionBoilerplate + `
resource "rediscloud_active_active_subscription_database" "example" {
	subscription_id = rediscloud_active_active_subscription.example.id
	name = "%s"
	memory_limit_in_gb = 3
	support_oss_cluster_api = false 
	external_endpoint_for_oss_cluster_api = false
	enable_tls = false

	global_data_persistence = "none"
	global_password = "%s" 
	global_source_ips = ["192.168.0.0/16", "192.170.0.0/16"]
	global_alert {
		name = "dataset-size"
		value = 40
	}
	override_region {
		name = "us-east-1"
		override_global_data_persistence = "aof-every-write"
		override_global_source_ips = ["192.175.0.0/16"]
		override_global_password = "region-specific-password"
		override_global_alert {
			name = "dataset-size"
			value = 42
		}
		remote_backup {
			interval = "every-6-hours"
			time_utc = "16:00"
			storage_type = "aws-s3"
			storage_path = "uri://interval.not.12.or.24.hours.test"
		}
	}
	override_region {
		name = "us-east-2"
	}
} 
`
