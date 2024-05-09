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
func TestAccResourceRedisCloudFlexibleDatabase_CRUDI(t *testing.T) {

	name := acctest.RandomWithPrefix(testResourcePrefix)
	password := acctest.RandString(20)
	resourceName := "rediscloud_flexible_database.example"
	subscriptionResourceName := "rediscloud_flexible_subscription.example"
	replicaResourceName := "rediscloud_flexible_database.example_replica"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	var subId int

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckSubscriptionDestroy,
		Steps: []resource.TestStep{
			// Test database and replica database creation
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudFlexibleDatabase, testCloudAccountName, name, password) + testAccResourceRedisCloudFlexibleDatabaseReplica,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "example"),
					resource.TestCheckResourceAttr(resourceName, "protocol", "redis"),
					resource.TestCheckResourceAttr(resourceName, "memory_limit_in_gb", "3"),
					resource.TestCheckResourceAttr(resourceName, "replication", "false"),
					resource.TestCheckResourceAttr(resourceName, "support_oss_cluster_api", "false"),
					resource.TestCheckResourceAttr(resourceName, "resp_version", "resp2"),
					resource.TestCheckResourceAttr(resourceName, "throughput_measurement_by", "operations-per-second"),
					resource.TestCheckResourceAttr(resourceName, "throughput_measurement_value", "1000"),
					resource.TestCheckResourceAttr(resourceName, "data_persistence", "none"),
					resource.TestCheckResourceAttr(resourceName, "data_eviction", "allkeys-random"),
					resource.TestCheckResourceAttr(resourceName, "average_item_size_in_bytes", "0"),
					resource.TestCheckResourceAttr(resourceName, "client_ssl_certificate", ""),
					resource.TestCheckResourceAttr(resourceName, "periodic_backup_path", ""),
					resource.TestCheckResourceAttr(resourceName, "external_endpoint_for_oss_cluster_api", "false"),
					resource.TestCheckResourceAttr(resourceName, "password", password),
					resource.TestCheckResourceAttr(resourceName, "alert.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "alert.0.name", "dataset-size"),
					resource.TestCheckResourceAttr(resourceName, "alert.0.value", "40"),
					resource.TestCheckResourceAttr(resourceName, "modules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "modules.0.name", "RedisBloom"),
					resource.TestCheckResourceAttr(resourceName, "enable_default_user", "true"),
					// Replica tests
					resource.TestCheckResourceAttr(replicaResourceName, "name", "example-replica"),
					// should be the value specified in the replica config, rather than the primary database
					resource.TestCheckResourceAttr(replicaResourceName, "memory_limit_in_gb", "1"),
					resource.TestCheckResourceAttr(replicaResourceName, "replica_of.#", "1"),

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

						if redis.StringValue(sub.Name) != name {
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
			// Test database is updated successfully
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudFlexibleDatabaseUpdate, testCloudAccountName, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "example-updated"),
					resource.TestCheckResourceAttr(resourceName, "protocol", "redis"),
					resource.TestCheckResourceAttr(resourceName, "memory_limit_in_gb", "1"),
					resource.TestCheckResourceAttr(resourceName, "replication", "true"),
					resource.TestCheckResourceAttr(resourceName, "support_oss_cluster_api", "true"),
					resource.TestCheckResourceAttr(resourceName, "resp_version", "resp2"),
					resource.TestCheckResourceAttr(resourceName, "throughput_measurement_by", "operations-per-second"),
					resource.TestCheckResourceAttr(resourceName, "throughput_measurement_value", "2000"),
					resource.TestCheckResourceAttr(resourceName, "data_persistence", "aof-every-write"),
					resource.TestCheckResourceAttr(resourceName, "data_eviction", "volatile-lru"),
					resource.TestCheckResourceAttr(resourceName, "average_item_size_in_bytes", "0"),
					resource.TestCheckResourceAttr(resourceName, "client_ssl_certificate", ""),
					resource.TestCheckResourceAttr(resourceName, "periodic_backup_path", ""),
					resource.TestCheckResourceAttr(resourceName, "external_endpoint_for_oss_cluster_api", "true"),
					resource.TestCheckResourceAttr(resourceName, "password", "updated-password"),
					resource.TestCheckResourceAttr(resourceName, "alert.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "alert.0.name", "dataset-size"),
					resource.TestCheckResourceAttr(resourceName, "alert.0.value", "80"),
					resource.TestCheckResourceAttr(resourceName, "modules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "modules.0.name", "RedisBloom"),
					resource.TestCheckResourceAttr(resourceName, "enable_default_user", "true"),
				),
			},
			// Test that Alerts are deleted
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudFlexibleDatabaseUpdateDestroyAlerts, testCloudAccountName, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "alert.#", "0"),
				),
			},
			// Test that a 32-character password is generated when no password is provided
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudFlexibleDatabaseNoPassword, testCloudAccountName, name),
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						is := s.RootModule().Resources["rediscloud_flexible_database.no_password_database"].Primary
						if len(is.Attributes["password"]) != 32 {
							return fmt.Errorf("password should be set to a random 32-character string")
						}
						return nil
					},
				),
			},
			// Test that that database is imported successfully
			{
				ResourceName:      "rediscloud_flexible_database.no_password_database",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccResourceRedisCloudFlexibleDatabase_optionalAttributes(t *testing.T) {
	// Test that attributes can be optional, either by setting them or not having them set when compared to CRUDI test
	name := acctest.RandomWithPrefix(testResourcePrefix)
	resourceName := "rediscloud_flexible_database.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")
	portNumber := 10101

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudFlexibleDatabaseOptionalAttributes, testCloudAccountName, name, portNumber),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "protocol", "redis"),
					resource.TestCheckResourceAttr(resourceName, "port", strconv.Itoa(portNumber)),
				),
			},
		},
	})
}

func TestAccResourceRedisCloudFlexibleDatabase_timeUtcRequiresValidInterval(t *testing.T) {
	name := acctest.RandomWithPrefix(testResourcePrefix)
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccResourceRedisCloudFlexibleDatabaseInvalidTimeUtc, testCloudAccountName, name),
				ExpectError: regexp.MustCompile("unexpected value at remote_backup\\.0\\.time_utc - time_utc can only be set when interval is either every-24-hours or every-12-hours"),
			},
		},
	})
}

// Tests the multi-modules feature in a database resource.
func TestAccResourceRedisCloudFlexibleDatabase_MultiModules(t *testing.T) {
	name := acctest.RandomWithPrefix(testResourcePrefix)
	dbName := "db-multi-modules"
	resourceName := "rediscloud_flexible_database.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudFlexibleDatabaseMultiModules, testCloudAccountName, name, dbName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", dbName),
					resource.TestCheckResourceAttr(resourceName, "modules.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "modules.0.name", "RedisBloom"),
					resource.TestCheckResourceAttr(resourceName, "modules.1.name", "RedisJSON"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccResourceRedisCloudFlexibleDatabase_respversion(t *testing.T) {
	// Test that attributes can be optional, either by setting them or not having them set when compared to CRUDI test
	name := acctest.RandomWithPrefix(testResourcePrefix)
	resourceName := "rediscloud_flexible_database.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")
	portNumber := 10101

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudFlexibleDatabaseRespVersions, testCloudAccountName, name, portNumber, "resp2"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "resp_version", "resp2"),
				),
			},
			{
				Config:      fmt.Sprintf(testAccResourceRedisCloudFlexibleDatabaseRespVersions, testCloudAccountName, name, portNumber, "resp3"),
				ExpectError: regexp.MustCompile("Selected RESP version is not supported for this database version\\.*"),
			},
			{
				Config:      fmt.Sprintf(testAccResourceRedisCloudFlexibleDatabaseRespVersions, testCloudAccountName, name, portNumber, "best_resp_100"),
				ExpectError: regexp.MustCompile("Bad Request: JSON parameter contains unsupported fields / values. JSON parse error: Cannot deserialize value of type `mappings.RespVersion` from String \"best_resp_100\": not one of the values accepted for Enum class: \\[resp2, resp3]"),
			},
		},
	})
}

const flexibleSubscriptionBoilerplate = `
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type = "AWS" 
  name = "%s"
}

resource "rediscloud_flexible_subscription" "example" {

  name = "%s"
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage = "ram"

  allowlist {
    cidrs = ["192.168.0.0/16"]
    security_group_ids = []
  }

  cloud_provider {
    provider = data.rediscloud_cloud_account.account.provider_type
    cloud_account_id = data.rediscloud_cloud_account.account.id
    region {
      region = "eu-west-1"
      networking_deployment_cidr = "10.0.0.0/24"
      preferred_availability_zones = ["eu-west-1a"]
    }
  }

  creation_plan {
    memory_limit_in_gb = 1
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 1000
    quantity = 1
    replication=false
    support_oss_cluster_api=false
    modules = []
  }
}
`

const multiModulesFlexibleSubscriptionBoilerplate = `
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}
data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type            = "AWS"
  name                     = "%s"
}
resource "rediscloud_flexible_subscription" "example" {
  name              = "%s"
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage    = "ram"
  allowlist {
    cidrs = ["192.168.0.0/16"]
    security_group_ids = []
  }
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
    memory_limit_in_gb           = 1
    quantity                     = 1
    replication                  = false
    support_oss_cluster_api      = false
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 1000
    modules                      = ["RedisJSON", "RedisBloom"]
  }
}
`

// Create and Read tests
// TF config for provisioning a new database
const testAccResourceRedisCloudFlexibleDatabase = flexibleSubscriptionBoilerplate + `
resource "rediscloud_flexible_database" "example" {
    subscription_id = rediscloud_flexible_subscription.example.id
    name = "example"
    protocol = "redis"
    memory_limit_in_gb = 3
    data_persistence = "none"
    data_eviction = "allkeys-random"
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 1000
    password = "%s"
    support_oss_cluster_api = false 
    external_endpoint_for_oss_cluster_api = false
    replication = false
    average_item_size_in_bytes = 0
    client_ssl_certificate = "" 
    periodic_backup_path = ""
	enable_default_user = true

    alert {
        name = "dataset-size"
        value = 40
    }

    modules = [
        {
          name = "RedisBloom"
        }
    ]
} 
`

const testAccResourceRedisCloudFlexibleDatabaseOptionalAttributes = flexibleSubscriptionBoilerplate + `
resource "rediscloud_flexible_database" "example" {
    subscription_id = rediscloud_flexible_subscription.example.id
    name = "example-no-protocol"
    memory_limit_in_gb = 1
    data_persistence = "none"
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 1000
    port = %d
}
`

const testAccResourceRedisCloudFlexibleDatabaseInvalidTimeUtc = flexibleSubscriptionBoilerplate + `
resource "rediscloud_flexible_database" "example" {
    subscription_id = rediscloud_flexible_subscription.example.id
    name = "example-no-protocol"
    memory_limit_in_gb = 1
    data_persistence = "none"
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 1000
    remote_backup {
        interval = "every-6-hours"
        time_utc = "16:00"
        storage_type = "aws-s3"
        storage_path = "uri://interval.not.12.or.24.hours.test"
    }
} 
`

// TF config for provisioning a database where the password is not specified
const testAccResourceRedisCloudFlexibleDatabaseNoPassword = flexibleSubscriptionBoilerplate + `
resource "rediscloud_flexible_database" "no_password_database" {
    subscription_id = rediscloud_flexible_subscription.example.id
    name = "example-no-password"
    protocol = "redis"
    memory_limit_in_gb = 1
    data_persistence = "none"
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 1000   
} 
`

// TF config for provisioning a database which is a replica of an existing database
const testAccResourceRedisCloudFlexibleDatabaseReplica = `
resource "rediscloud_flexible_database" "example_replica" {
  subscription_id = rediscloud_flexible_subscription.example.id
  name = "example-replica"
  protocol = "redis"
  memory_limit_in_gb = 1
  throughput_measurement_by = "operations-per-second"
  throughput_measurement_value = 1000
  replica_of = [format("redis://%s", rediscloud_flexible_database.example.public_endpoint)]
} 
`

// TF config for updating a database
const testAccResourceRedisCloudFlexibleDatabaseUpdate = flexibleSubscriptionBoilerplate + `
resource "rediscloud_flexible_database" "example" {
    subscription_id = rediscloud_flexible_subscription.example.id
    name = "example-updated"
    protocol = "redis"
    memory_limit_in_gb = 1
    data_persistence = "aof-every-write"
	data_eviction = "volatile-lru"
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 2000
	password = "updated-password"
	support_oss_cluster_api = true 
	external_endpoint_for_oss_cluster_api = true
	replication = true
	average_item_size_in_bytes = 0
	enable_default_user = true

	alert {
		name = "dataset-size"
		value = 80
	}

    modules = [
        {
          name = "RedisBloom"
        }
    ]
} 
`

const testAccResourceRedisCloudFlexibleDatabaseUpdateDestroyAlerts = flexibleSubscriptionBoilerplate + `
resource "rediscloud_flexible_database" "example" {
    subscription_id = rediscloud_flexible_subscription.example.id
    name = "example-updated"
    protocol = "redis"
    memory_limit_in_gb = 1
    data_persistence = "aof-every-write"
	data_eviction = "volatile-lru"
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 2000
	password = "updated-password"
	support_oss_cluster_api = true 
	external_endpoint_for_oss_cluster_api = true
	replication = true
	average_item_size_in_bytes = 0

    modules = [
        {
          name = "RedisBloom"
        }
    ]
} 
`

const testAccResourceRedisCloudFlexibleDatabaseMultiModules = multiModulesFlexibleSubscriptionBoilerplate + `
resource "rediscloud_flexible_database" "example" {
    subscription_id              = rediscloud_flexible_subscription.example.id
    name                         = "%s"
    protocol                     = "redis"
    memory_limit_in_gb           = 1
    data_persistence             = "none"
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 1000
    modules = [
        {
          name = "RedisJSON"
        },
        {
          name = "RedisBloom"
        }
    ]
    alert {
      name  = "latency"
      value = 11
    }
}
`

const testAccResourceRedisCloudFlexibleDatabaseRespVersions = flexibleSubscriptionBoilerplate + `
resource "rediscloud_flexible_database" "example" {
    subscription_id = rediscloud_flexible_subscription.example.id
    name = "example"
    memory_limit_in_gb = 1
    data_persistence = "none"
	throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 1000
    port = %d
	resp_version = "%s"
}
`
