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
func TestAccResourceRedisCloudActiveActiveDatabase_CRUDI(t *testing.T) {

	subscriptionName := acctest.RandomWithPrefix(testResourcePrefix) + "-subscription"
	name := acctest.RandomWithPrefix(testResourcePrefix) + "-database"
	password := acctest.RandString(20)
	const resourceName = "rediscloud_active_active_subscription_database.example"
	const datasourceName = "data.rediscloud_active_active_subscription_database.example"
	const subscriptionResourceName = "rediscloud_active_active_subscription.example"

	var subId int

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			// Test database creation
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudActiveActiveDatabase, subscriptionName, name, password),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Test resource
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "dataset_size_in_gb", "3"),
					resource.TestCheckResourceAttr(resourceName, "support_oss_cluster_api", "false"),
					resource.TestCheckResourceAttr(resourceName, "global_data_persistence", "none"),
					resource.TestCheckResourceAttr(resourceName, "external_endpoint_for_oss_cluster_api", "false"),
					resource.TestCheckResourceAttr(resourceName, "global_password", password),
					resource.TestCheckResourceAttr(resourceName, "enable_tls", "false"),
					resource.TestCheckResourceAttr(resourceName, "data_eviction", "volatile-lru"),
					resource.TestCheckResourceAttr(resourceName, "global_alert.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "global_alert.0.name", "dataset-size"),
					resource.TestCheckResourceAttr(resourceName, "global_alert.0.value", "40"),
					resource.TestCheckResourceAttr(resourceName, "global_modules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "global_modules.0", "RedisJSON"),
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

					resource.TestCheckResourceAttr(resourceName, "tags.deployment_family", "blue"),
					resource.TestCheckResourceAttr(resourceName, "tags.priority", "code-2"),

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

					// Test datasource
					resource.TestCheckResourceAttrSet(datasourceName, "subscription_id"),
					resource.TestCheckResourceAttrSet(datasourceName, "db_id"),
					resource.TestCheckResourceAttr(datasourceName, "name", name),
					resource.TestCheckResourceAttr(datasourceName, "dataset_size_in_gb", "3"),
					resource.TestCheckResourceAttr(datasourceName, "support_oss_cluster_api", "false"),
					resource.TestCheckResourceAttr(datasourceName, "external_endpoint_for_oss_cluster_api", "false"),
					resource.TestCheckResourceAttr(datasourceName, "enable_tls", "false"),
					resource.TestCheckResourceAttr(datasourceName, "data_eviction", "volatile-lru"),
					resource.TestCheckResourceAttr(datasourceName, "global_modules.#", "1"),
					resource.TestCheckResourceAttr(datasourceName, "global_modules.0", "RedisJSON"),

					resource.TestCheckResourceAttr(datasourceName, "tags.deployment_family", "blue"),
					resource.TestCheckResourceAttr(datasourceName, "tags.priority", "code-2"),
				),
			},
			// Test database is updated successfully, including updates to both global and local alerts and clearing modules
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudActiveActiveDatabaseUpdate, subscriptionName, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Test resource
					resource.TestCheckResourceAttr(resourceName, "dataset_size_in_gb", "1"),
					resource.TestCheckResourceAttr(resourceName, "support_oss_cluster_api", "true"),
					resource.TestCheckResourceAttr(resourceName, "external_endpoint_for_oss_cluster_api", "true"),
					resource.TestCheckResourceAttr(resourceName, "global_data_persistence", "aof-every-1-second"),
					resource.TestCheckResourceAttr(resourceName, "global_password", "updated-password"),
					resource.TestCheckResourceAttr(resourceName, "global_alert.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "global_alert.0.name", "dataset-size"),
					resource.TestCheckResourceAttr(resourceName, "global_alert.0.value", "60"),

					// Changes are ignored after creation
					resource.TestCheckResourceAttr(resourceName, "global_modules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "global_modules.0", "RedisJSON"),

					resource.TestCheckResourceAttr(resourceName, "override_region.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "override_region.0.name", "us-east-1"),
					resource.TestCheckResourceAttr(resourceName, "override_region.0.override_global_data_persistence", "none"),
					resource.TestCheckResourceAttr(resourceName, "override_region.0.override_global_password", "password-updated"),
					resource.TestCheckResourceAttr(resourceName, "override_region.0.override_global_alert.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "override_region.0.override_global_alert.0.name", "dataset-size"),
					resource.TestCheckResourceAttr(resourceName, "override_region.0.override_global_alert.0.value", "41"),
					resource.TestCheckResourceAttr(resourceName, "override_region.0.override_global_source_ips.#", "0"),

					// Test datasource
					resource.TestCheckResourceAttr(datasourceName, "dataset_size_in_gb", "1"),
					resource.TestCheckResourceAttr(datasourceName, "support_oss_cluster_api", "true"),
					resource.TestCheckResourceAttr(datasourceName, "external_endpoint_for_oss_cluster_api", "true"),
				),
			},
			// Test database is updated, including deletion of global and local alerts and replacing modules
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudActiveActiveDatabaseUpdateNoAlerts, subscriptionName, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "dataset_size_in_gb", "1"),
					resource.TestCheckResourceAttr(resourceName, "support_oss_cluster_api", "true"),
					resource.TestCheckResourceAttr(resourceName, "external_endpoint_for_oss_cluster_api", "true"),
					resource.TestCheckResourceAttr(resourceName, "global_data_persistence", "aof-every-1-second"),
					resource.TestCheckResourceAttr(resourceName, "global_password", "updated-password"),
					resource.TestCheckResourceAttr(resourceName, "global_alert.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "global_modules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "global_modules.0", "RedisJSON"),

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
				Config:            fmt.Sprintf(testAccResourceRedisCloudActiveActiveDatabaseImport, subscriptionName, name),
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
					"override_region.0.latest_backup_status.#",
					"override_region.0.latest_backup_status.0.%",
					"override_region.0.latest_backup_status.0.error.#",
					"override_region.0.latest_backup_status.0.error.0.%",
					"override_region.0.latest_backup_status.0.error.0.description",
					"override_region.0.latest_backup_status.0.error.0.status",
					"override_region.0.latest_backup_status.0.error.0.type",
				},
			},
		},
	})
}

func TestAccResourceRedisCloudActiveActiveDatabase_optionalAttributes(t *testing.T) {
	// Test that attributes can be optional, either by setting them or not having them set when compared to CRUDI test
	subscriptionName := acctest.RandomWithPrefix(testResourcePrefix) + "-subscription"
	name := acctest.RandomWithPrefix(testResourcePrefix) + "-database"
	password := acctest.RandString(20)
	const resourceName = "rediscloud_active_active_subscription_database.example"
	portNumber := 10101

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudActiveActiveDatabaseOptionalAttributes, subscriptionName, name, password, portNumber),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "port", strconv.Itoa(portNumber)),
				),
			},
		},
	})
}

func TestAccResourceRedisCloudActiveActiveDatabase_timeUtcRequiresValidInterval(t *testing.T) {
	name := acctest.RandomWithPrefix(testResourcePrefix)
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")
	password := acctest.RandString(20)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccResourceRedisCloudActiveActiveDatabaseInvalidTimeUtc, testCloudAccountName, name, password),
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
`

// Create and Read tests
// TF config for provisioning a new database
const testAccResourceRedisCloudActiveActiveDatabase = activeActiveSubscriptionBoilerplate + `
resource "rediscloud_active_active_subscription_database" "example" {
    subscription_id = rediscloud_active_active_subscription.example.id
    name = "%s"
    dataset_size_in_gb = 3
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
	global_modules = ["RedisJSON"]
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

	tags = {
		"deployment_family" = "blue"
		"priority" = "code-2"
	}

}

data "rediscloud_active_active_subscription_database" "example" {
	subscription_id = rediscloud_active_active_subscription.example.id
	name = rediscloud_active_active_subscription_database.example.name
}
`

// TF config for updating a database
const testAccResourceRedisCloudActiveActiveDatabaseUpdate = activeActiveSubscriptionBoilerplate + `
resource "rediscloud_active_active_subscription_database" "example" {
    subscription_id = rediscloud_active_active_subscription.example.id
    name = "%s"
    dataset_size_in_gb = 1
    support_oss_cluster_api = true 
    external_endpoint_for_oss_cluster_api = true
    
    global_data_persistence = "aof-every-1-second"
    global_password = "updated-password" 
    global_source_ips = ["192.170.0.0/16"]
	global_alert {
		name = "dataset-size"
		value = 60
	}
	global_modules = []

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

data "rediscloud_active_active_subscription_database" "example" {
	subscription_id = rediscloud_active_active_subscription.example.id
	name = rediscloud_active_active_subscription_database.example.name
}
`

const testAccResourceRedisCloudActiveActiveDatabaseUpdateNoAlerts = activeActiveSubscriptionBoilerplate + `
resource "rediscloud_active_active_subscription_database" "example" {
    subscription_id = rediscloud_active_active_subscription.example.id
    name = "%s"
    dataset_size_in_gb = 1
    support_oss_cluster_api = true 
    external_endpoint_for_oss_cluster_api = true
    
    global_data_persistence = "aof-every-1-second"
    global_password = "updated-password" 
    global_source_ips = ["192.170.0.0/16"]

	global_modules = ["RedisJSON"]

	override_region {
		name = "us-east-1"
		override_global_data_persistence = "none"
		override_global_password = "password-updated"
	}
}
`

// TF config for updating a database
const testAccResourceRedisCloudActiveActiveDatabaseImport = activeActiveSubscriptionBoilerplate + `
resource "rediscloud_active_active_subscription_database" "example" {
    subscription_id = rediscloud_active_active_subscription.example.id
    name = "%s"
    dataset_size_in_gb = 1
}
`

const testAccResourceRedisCloudActiveActiveDatabaseOptionalAttributes = `
data "rediscloud_payment_method" "card" {
	card_type = "Visa"
}

resource "rediscloud_active_active_subscription" "example" {
	name = "%s"
	payment_method_id = data.rediscloud_payment_method.card.id
	cloud_provider = "AWS"
	redis_version = "latest"

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

resource "rediscloud_active_active_subscription_database" "example" {
	subscription_id = rediscloud_active_active_subscription.example.id
	name = "%s"
	dataset_size_in_gb = 3
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

const testAccResourceRedisCloudActiveActiveDatabaseInvalidTimeUtc = activeActiveSubscriptionBoilerplate + `
resource "rediscloud_active_active_subscription_database" "example" {
	subscription_id = rediscloud_active_active_subscription.example.id
	name = "%s"
	dataset_size_in_gb = 3
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
