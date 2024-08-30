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
func TestAccResourceRedisCloudProDatabase_CRUDI(t *testing.T) {

	name := acctest.RandomWithPrefix(testResourcePrefix)
	password := acctest.RandString(20)
	const resourceName = "rediscloud_subscription_database.example"
	const subscriptionResourceName = "rediscloud_subscription.example"
	const replicaResourceName = "rediscloud_subscription_database.example_replica"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	var subId int

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			// Test database and replica database creation
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudProDatabase, testCloudAccountName, name, password) + testAccResourceRedisCloudProDatabaseReplica,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "example"),
					resource.TestCheckResourceAttr(resourceName, "protocol", "redis"),
					resource.TestCheckResourceAttr(resourceName, "memory_limit_in_gb", "3"),
					resource.TestCheckResourceAttr(resourceName, "replication", "false"),
					resource.TestCheckResourceAttr(resourceName, "support_oss_cluster_api", "false"),
					resource.TestCheckResourceAttr(resourceName, "resp_version", "resp3"),
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

					resource.TestCheckResourceAttr(resourceName, "tags.market", "emea"),
					resource.TestCheckResourceAttr(resourceName, "tags.material", "cardboard"),

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
				Config: fmt.Sprintf(testAccResourceRedisCloudProDatabaseUpdate, testCloudAccountName, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "example-updated"),
					resource.TestCheckResourceAttr(resourceName, "protocol", "redis"),
					resource.TestCheckResourceAttr(resourceName, "memory_limit_in_gb", "1"),
					resource.TestCheckResourceAttr(resourceName, "replication", "true"),
					resource.TestCheckResourceAttr(resourceName, "support_oss_cluster_api", "true"),
					resource.TestCheckResourceAttr(resourceName, "resp_version", "resp3"),
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
			// Test that alerts are deleted
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudProDatabaseUpdateDestroyAlerts, testCloudAccountName, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "alert.#", "0"),
				),
			},
			// Test that a 32-character password is generated when no password is provided
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudProDatabaseNoPassword, testCloudAccountName, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					func(s *terraform.State) error {
						is := s.RootModule().Resources["rediscloud_subscription_database.no_password_database"].Primary
						if len(is.Attributes["password"]) != 32 {
							return fmt.Errorf("password should be set to a random 32-character string")
						}
						return nil
					},
				),
			},
			// Test that that database is imported successfully
			{
				ResourceName:      "rediscloud_subscription_database.no_password_database",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccResourceRedisCloudProDatabase_optionalAttributes(t *testing.T) {
	// Test that attributes can be optional, either by setting them or not having them set when compared to CRUDI test
	name := acctest.RandomWithPrefix(testResourcePrefix)
	const resourceName = "rediscloud_subscription_database.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")
	portNumber := 10101

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudProDatabaseOptionalAttributes, testCloudAccountName, name, portNumber),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "protocol", "redis"),
					resource.TestCheckResourceAttr(resourceName, "port", strconv.Itoa(portNumber)),
				),
			},
		},
	})
}

func TestAccResourceRedisCloudProDatabase_timeUtcRequiresValidInterval(t *testing.T) {
	name := acctest.RandomWithPrefix(testResourcePrefix)
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccResourceRedisCloudProDatabaseInvalidTimeUtc, testCloudAccountName, name),
				ExpectError: regexp.MustCompile("unexpected value at remote_backup\\.0\\.time_utc - time_utc can only be set when interval is either every-24-hours or every-12-hours"),
			},
		},
	})
}

// Tests the multi-modules feature in a database resource.
func TestAccResourceRedisCloudProDatabase_MultiModules(t *testing.T) {
	name := acctest.RandomWithPrefix(testResourcePrefix)
	dbName := "db-multi-modules"
	const resourceName = "rediscloud_subscription_database.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudProDatabaseMultiModules, testCloudAccountName, name, dbName),
				Check: resource.ComposeAggregateTestCheckFunc(
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

func TestAccResourceRedisCloudProDatabase_respversion(t *testing.T) {
	// Test that attributes can be optional, either by setting them or not having them set when compared to CRUDI test
	name := acctest.RandomWithPrefix(testResourcePrefix)
	const resourceName = "rediscloud_subscription_database.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")
	portNumber := 10101

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudProDatabaseRespVersions, testCloudAccountName, name, portNumber, "resp2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "resp_version", "resp2"),
				),
			},
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudProDatabaseRespVersions, testCloudAccountName, name, portNumber, "resp3"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "resp_version", "resp3"),
				),
			},
			{
				Config:      fmt.Sprintf(testAccResourceRedisCloudProDatabaseRespVersions, testCloudAccountName, name, portNumber, "best_resp_100"),
				ExpectError: regexp.MustCompile("Bad Request: JSON parameter contains unsupported fields / values. JSON parse error: Cannot deserialize value of type `mappings.RespVersion` from String \"best_resp_100\": not one of the values accepted for Enum class: \\[resp2, resp3]"),
			},
		},
	})
}

const proSubscriptionBoilerplate = `
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type = "AWS" 
  name = "%s"
}

resource "rediscloud_subscription" "example" {

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

const multiModulesProSubscriptionBoilerplate = `
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}
data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type            = "AWS"
  name                     = "%s"
}
resource "rediscloud_subscription" "example" {
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
const testAccResourceRedisCloudProDatabase = proSubscriptionBoilerplate + `
resource "rediscloud_subscription_database" "example" {
    subscription_id = rediscloud_subscription.example.id
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

	tags = {
		"market" = "emea"
		"material" = "cardboard"
	}
} 
`

const testAccResourceRedisCloudProDatabaseOptionalAttributes = proSubscriptionBoilerplate + `
resource "rediscloud_subscription_database" "example" {
    subscription_id = rediscloud_subscription.example.id
    name = "example-no-protocol"
    memory_limit_in_gb = 1
    data_persistence = "none"
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 1000
    port = %d
}
`

const testAccResourceRedisCloudProDatabaseInvalidTimeUtc = proSubscriptionBoilerplate + `
resource "rediscloud_subscription_database" "example" {
    subscription_id = rediscloud_subscription.example.id
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
const testAccResourceRedisCloudProDatabaseNoPassword = proSubscriptionBoilerplate + `
resource "rediscloud_subscription_database" "no_password_database" {
    subscription_id = rediscloud_subscription.example.id
    name = "example-no-password"
    protocol = "redis"
    memory_limit_in_gb = 1
    data_persistence = "none"
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 1000   
} 
`

// TF config for provisioning a database which is a replica of an existing database
const testAccResourceRedisCloudProDatabaseReplica = `
resource "rediscloud_subscription_database" "example_replica" {
  subscription_id = rediscloud_subscription.example.id
  name = "example-replica"
  protocol = "redis"
  memory_limit_in_gb = 1
  throughput_measurement_by = "operations-per-second"
  throughput_measurement_value = 1000
  replica_of = [format("redis://%s", rediscloud_subscription_database.example.public_endpoint)]
} 
`

// TF config for updating a database
const testAccResourceRedisCloudProDatabaseUpdate = proSubscriptionBoilerplate + `
resource "rediscloud_subscription_database" "example" {
    subscription_id = rediscloud_subscription.example.id
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

const testAccResourceRedisCloudProDatabaseUpdateDestroyAlerts = proSubscriptionBoilerplate + `
resource "rediscloud_subscription_database" "example" {
    subscription_id = rediscloud_subscription.example.id
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

const testAccResourceRedisCloudProDatabaseMultiModules = multiModulesProSubscriptionBoilerplate + `
resource "rediscloud_subscription_database" "example" {
    subscription_id              = rediscloud_subscription.example.id
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

const testAccResourceRedisCloudProDatabaseRespVersions = proSubscriptionBoilerplate + `
resource "rediscloud_subscription_database" "example" {
    subscription_id = rediscloud_subscription.example.id
    name = "example"
    memory_limit_in_gb = 1
    data_persistence = "none"
	throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 1000
    port = %d
	resp_version = "%s"
}
`
