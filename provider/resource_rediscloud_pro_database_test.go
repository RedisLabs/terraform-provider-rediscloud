package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	client2 "github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// Checks CRUDI (CREATE, READ, UPDATE, IMPORT) operations on the database resource.
func TestAccResourceRedisCloudProDatabase_CRUDI(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

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
				Config: fmt.Sprintf(utils.GetTestConfig(t, "./pro/testdata/pro_database_with_replica.tf"), testCloudAccountName, name, password),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "example"),
					resource.TestCheckResourceAttr(resourceName, "protocol", "redis"),
					resource.TestCheckResourceAttr(resourceName, "dataset_size_in_gb", "1"),
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
					resource.TestCheckResourceAttr(resourceName, "alert.0.value", "1"),
					resource.TestCheckResourceAttr(resourceName, "modules.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "enable_default_user", "true"),
					resource.TestCheckResourceAttr(resourceName, "redis_version", "8.2"),

					resource.TestCheckResourceAttr(resourceName, "tags.market", "emea"),
					resource.TestCheckResourceAttr(resourceName, "tags.material", "cardboard"),

					// Replica tests
					resource.TestCheckResourceAttr(replicaResourceName, "name", "example-replica"),
					// should be the value specified in the replica config, rather than the primary database
					resource.TestCheckResourceAttr(replicaResourceName, "dataset_size_in_gb", "1"),
					resource.TestCheckResourceAttr(replicaResourceName, "replica_of.#", "1"),

					// Test databases exist
					func(s *terraform.State) error {
						r := s.RootModule().Resources[subscriptionResourceName]

						var err error
						subId, err = strconv.Atoi(r.Primary.ID)
						if err != nil {
							return fmt.Errorf("couldn't parse the subscription ID: %s", redis.StringValue(&r.Primary.ID))
						}

						client := testProvider.Meta().(*client2.ApiClient)
						sub, err := client.Client.Subscription.Get(context.TODO(), subId)
						if err != nil {
							return err
						}

						if redis.StringValue(sub.Name) != name {
							return fmt.Errorf("unexpected name value: %s", redis.StringValue(sub.Name))
						}

						listDb := client.Client.Database.List(context.TODO(), subId)
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
				Config: fmt.Sprintf(utils.GetTestConfig(t, "./pro/testdata/pro_database_update.tf"), testCloudAccountName, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "example-updated"),
					resource.TestCheckResourceAttr(resourceName, "protocol", "redis"),
					resource.TestCheckResourceAttr(resourceName, "dataset_size_in_gb", "1"),
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
					resource.TestCheckResourceAttr(resourceName, "modules.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "enable_default_user", "true"),
					resource.TestCheckResourceAttr(resourceName, "redis_version", "8.2"),
				),
			},
			// Test that alerts are deleted
			{
				Config: fmt.Sprintf(utils.GetTestConfig(t, "./pro/testdata/pro_database_update_destroy_alerts.tf"), testCloudAccountName, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "alert.#", "0"),
				),
			},
			// Test that a 32-character password is generated when no password is provided
			{
				Config: fmt.Sprintf(utils.GetTestConfig(t, "./pro/testdata/pro_database_no_password.tf"), testCloudAccountName, name),
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

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

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
				Config: fmt.Sprintf(utils.GetTestConfig(t, "./pro/testdata/pro_database_optional_attributes.tf"), testCloudAccountName, name, portNumber),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "protocol", "redis"),
					resource.TestCheckResourceAttr(resourceName, "port", strconv.Itoa(portNumber)),
				),
			},
		},
	})
}

func TestAccResourceRedisCloudProDatabase_timeUtcRequiresValidInterval(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := acctest.RandomWithPrefix(testResourcePrefix)
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(utils.GetTestConfig(t, "./pro/testdata/pro_database_invalid_time_utc.tf"), testCloudAccountName, name),
				ExpectError: regexp.MustCompile("unexpected value at remote_backup\\.0\\.time_utc - time_utc can only be set when interval is either every-24-hours or every-12-hours"),
			},
		},
	})
}

// Tests the multi-modules feature in a database resource.
func TestAccResourceRedisCloudProDatabase_MultiModules(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

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
				Config: fmt.Sprintf(utils.GetTestConfig(t, "./pro/testdata/pro_database_multi_modules.tf"), testCloudAccountName, name, dbName),
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

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

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
				Config: fmt.Sprintf(utils.GetTestConfig(t, "./pro/testdata/pro_database_resp_versions.tf"), testCloudAccountName, name, portNumber, "resp2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "resp_version", "resp2"),
				),
			},
			{
				Config: fmt.Sprintf(utils.GetTestConfig(t, "./pro/testdata/pro_database_resp_versions.tf"), testCloudAccountName, name, portNumber, "resp3"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "resp_version", "resp3"),
				),
			},
			{
				Config:      fmt.Sprintf(utils.GetTestConfig(t, "./pro/testdata/pro_database_resp_versions.tf"), testCloudAccountName, name, portNumber, "best_resp_100"),
				ExpectError: regexp.MustCompile("Bad Request: JSON parameter contains unsupported fields / values. JSON parse error: Cannot deserialize value of type `mappings.RespVersion` from String \"best_resp_100\": not one of the values accepted for Enum class: \\[resp2, resp3]"),
			},
		},
	})
}

func TestAccResourceRedisCloudProDatabase_autoMinorVersionUpgrade(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")
	t.Skip("auto_minor_version_upgrade feature temporarily removed")

	name := acctest.RandomWithPrefix(testResourcePrefix)
	const resourceName = "rediscloud_subscription_database.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			// Test database creation with auto_minor_version_upgrade set to false
			{
				Config: fmt.Sprintf(utils.GetTestConfig(t, "./pro/testdata/pro_database_auto_minor_version_upgrade.tf"), testCloudAccountName, name, "false"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "auto-minor-version-upgrade-test"),
					resource.TestCheckResourceAttr(resourceName, "auto_minor_version_upgrade", "false"),
				),
			},
			// Test database update with auto_minor_version_upgrade set to true
			{
				Config: fmt.Sprintf(utils.GetTestConfig(t, "./pro/testdata/pro_database_auto_minor_version_upgrade.tf"), testCloudAccountName, name, "true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "auto_minor_version_upgrade", "true"),
				),
			},
		},
	})
}
