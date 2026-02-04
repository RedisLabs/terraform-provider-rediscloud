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

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

// checks that a redis 8 database can be provisioned
func TestAccResourceRedisCloudProDatabase_Redis8(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := acctest.RandomWithPrefix(testResourcePrefix)
	password := acctest.RandString(20)
	const resourceName = "rediscloud_subscription_database.example"
	const subscriptionResourceName = "rediscloud_subscription.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	var subId int

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: getRedis8DatabaseConfig(t, testCloudAccountName, name, password),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "example"),
					resource.TestCheckResourceAttr(resourceName, "protocol", "redis"),
					resource.TestCheckResourceAttr(resourceName, "dataset_size_in_gb", "3"),
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
					resource.TestCheckResourceAttr(resourceName, "enable_default_user", "true"),
					resource.TestCheckResourceAttr(resourceName, "redis_version", "8.2"),

					resource.TestCheckResourceAttr(resourceName, "tags.market", "emea"),
					resource.TestCheckResourceAttr(resourceName, "tags.material", "cardboard"),

					// Test databases exist
					func(s *terraform.State) error {
						r := s.RootModule().Resources[subscriptionResourceName]

						var err error
						subId, err = strconv.Atoi(r.Primary.ID)
						if err != nil {
							return fmt.Errorf("couldn't parse the subscription ID: %s", redis.StringValue(&r.Primary.ID))
						}

						apiClient := sharedTestClient(t)
						sub, err := apiClient.Client.Subscription.Get(context.TODO(), subId)
						if err != nil {
							return err
						}

						if redis.StringValue(sub.Name) != name {
							return fmt.Errorf("unexpected name value: %s", redis.StringValue(sub.Name))
						}

						listDb := apiClient.Client.Database.List(context.TODO(), subId)
						if listDb.Next() != true {
							return fmt.Errorf("no database found: %w", listDb.Err())
						}
						if listDb.Err() != nil {
							return listDb.Err()
						}

						return nil
					},
				),
			},
		},
	})
}

func TestAccResourceRedisCloudProFlexDatabase_Redis8_CRUDI(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := acctest.RandomWithPrefix(testResourcePrefix)
	password := acctest.RandString(20)
	const resourceName = "rediscloud_subscription_database.example"
	const subscriptionResourceName = "rediscloud_subscription.example"
	const replicaResourceName = "rediscloud_subscription_database.example_replica"

	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	var subId int

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			// Test database and replica database creation
			{
				Config: fmt.Sprintf(utils.GetTestConfig(t, "./pro/testdata/pro_flex_database_redis_8_with_replica.tf"), testCloudAccountName, name, password),
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
					resource.TestCheckResourceAttr(resourceName, "external_endpoint_for_oss_cluster_api", "false"),
					resource.TestCheckResourceAttr(resourceName, "password", password),
					resource.TestCheckResourceAttr(resourceName, "alert.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "alert.0.name", "dataset-size"),
					resource.TestCheckResourceAttr(resourceName, "alert.0.value", "1"),
					resource.TestCheckResourceAttr(resourceName, "modules.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "enable_default_user", "true"),
					resource.TestCheckResourceAttr(resourceName, "redis_version", "8.2"),
					resource.TestCheckResourceAttr(resourceName, "ram_percentage", "30"),

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

						client := sharedTestClient(t)
						sub, err := client.Client.Subscription.Get(context.TODO(), subId)
						if err != nil {
							return err
						}

						if redis.StringValue(sub.Name) != name {
							return fmt.Errorf("unexpected name value: %s", redis.StringValue(sub.Name))
						}

						listDb := client.Client.Database.List(context.TODO(), subId)
						if listDb.Next() != true {
							return fmt.Errorf("no database found: %w", listDb.Err())
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
				Config: fmt.Sprintf(utils.GetTestConfig(t, "./pro/testdata/pro_flex_database_redis_8_update.tf"), testCloudAccountName, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "example-updated"),
					resource.TestCheckResourceAttr(resourceName, "protocol", "redis"),
					resource.TestCheckResourceAttr(resourceName, "dataset_size_in_gb", "1"),
					resource.TestCheckResourceAttr(resourceName, "replication", "false"),
					resource.TestCheckResourceAttr(resourceName, "support_oss_cluster_api", "true"),
					resource.TestCheckResourceAttr(resourceName, "resp_version", "resp3"),
					resource.TestCheckResourceAttr(resourceName, "throughput_measurement_by", "operations-per-second"),
					resource.TestCheckResourceAttr(resourceName, "throughput_measurement_value", "2000"),
					resource.TestCheckResourceAttr(resourceName, "data_persistence", "aof-every-write"),
					resource.TestCheckResourceAttr(resourceName, "data_eviction", "volatile-lru"),
					resource.TestCheckResourceAttr(resourceName, "average_item_size_in_bytes", "0"),
					resource.TestCheckResourceAttr(resourceName, "client_ssl_certificate", ""),
					resource.TestCheckResourceAttr(resourceName, "external_endpoint_for_oss_cluster_api", "true"),
					resource.TestCheckResourceAttr(resourceName, "password", "updated-password"),
					resource.TestCheckResourceAttr(resourceName, "alert.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "alert.0.name", "dataset-size"),
					resource.TestCheckResourceAttr(resourceName, "alert.0.value", "80"),
					resource.TestCheckResourceAttr(resourceName, "modules.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "enable_default_user", "true"),
					resource.TestCheckResourceAttr(resourceName, "redis_version", "8.2"),
					resource.TestCheckResourceAttr(resourceName, "ram_percentage", "40"),
				),
			},
			// Test that alerts are deleted
			{
				Config: fmt.Sprintf(utils.GetTestConfig(t, "./pro/testdata/pro_flex_database_redis_8_update_destroy_alerts.tf"), testCloudAccountName, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "alert.#", "0"),
				),
			},
			// Test that a 32-character password is generated when no password is provided
			{
				Config: fmt.Sprintf(utils.GetTestConfig(t, "./pro/testdata/pro_flex_database_redis_8_no_password.tf"), testCloudAccountName, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "ram_percentage", "20"),
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

// Checks that users can upgrade from 7.2 to 8.0
func TestAccResourceRedisCloudProDatabase_Redis8_Upgrade(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := acctest.RandomWithPrefix(testResourcePrefix)
	password := acctest.RandString(20)
	const resourceName = "rediscloud_subscription_database.example"
	const subscriptionResourceName = "rediscloud_subscription.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	var subId int

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			// Test database and replica database creation with Redis 7.2
			{
				Config: getRedis7DatabaseConfig(t, testCloudAccountName, name, password),
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
					resource.TestCheckResourceAttr(resourceName, "enable_default_user", "true"),
					resource.TestCheckResourceAttr(resourceName, "redis_version", "7.4"),

					resource.TestCheckResourceAttr(resourceName, "tags.market", "emea"),
					resource.TestCheckResourceAttr(resourceName, "tags.material", "cardboard"),

					// Test databases exist
					func(s *terraform.State) error {
						r := s.RootModule().Resources[subscriptionResourceName]

						var err error
						subId, err = strconv.Atoi(r.Primary.ID)
						if err != nil {
							return fmt.Errorf("couldn't parse the subscription ID: %s", redis.StringValue(&r.Primary.ID))
						}

						apiClient := sharedTestClient(t)
						sub, err := apiClient.Client.Subscription.Get(context.TODO(), subId)
						if err != nil {
							return err
						}

						if redis.StringValue(sub.Name) != name {
							return fmt.Errorf("unexpected name value: %s", redis.StringValue(sub.Name))
						}

						listDb := apiClient.Client.Database.List(context.TODO(), subId)
						if listDb.Next() != true {
							return fmt.Errorf("no database found: %w", listDb.Err())
						}
						if listDb.Err() != nil {
							return listDb.Err()
						}

						return nil
					},
				),
			},
			// Test database is updated successfully to Redis 8.0
			{
				Config: getRedis8DatabaseConfig(t, testCloudAccountName, name, password),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "redis_version", "8.2"),
				),
			},
		},
	})
}

// Test that modules cannot be set on Redis 8.x
func TestAccResourceRedisCloudProDatabase_Redis8_ModulesBlocked(t *testing.T) {
	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := acctest.RandomWithPrefix(testResourcePrefix)
	password := acctest.RandString(20)
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config:      getRedis8WithModulesConfig(t, testCloudAccountName, name, password),
				ExpectError: regexp.MustCompile(`"modules" cannot be explicitly set for Redis version 8\.0 as modules are bundled by default`),
			},
		},
	})
}

func getRedis7DatabaseConfig(t *testing.T, cloudAccountName, subscriptionName, password string) string {
	content := utils.GetTestConfig(t, "./pro/testdata/pro_database_redis_7.tf")
	return fmt.Sprintf(content, cloudAccountName, subscriptionName, password)
}

func getRedis8DatabaseConfig(t *testing.T, cloudAccountName, subscriptionName, password string) string {
	content := utils.GetTestConfig(t, "./pro/testdata/pro_database_redis_8.tf")
	return fmt.Sprintf(content, cloudAccountName, subscriptionName, password)
}

func getRedis8WithModulesConfig(t *testing.T, cloudAccountName, subscriptionName, password string) string {
	content := utils.GetTestConfig(t, "./pro/testdata/pro_database_redis_8_with_modules.tf")
	return fmt.Sprintf(content, cloudAccountName, subscriptionName, password)
}
