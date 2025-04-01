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

// Generates the base Terraform config for a Pro Subscription with QPF
func proSubscriptionQPFBoilerplate(name, cloudAccountName string) string {
	return fmt.Sprintf(`
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
    dataset_size_in_gb = 1
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 1000
    quantity = 1
    replication = false
    support_oss_cluster_api = false
    query_performance_factor = "2x"
    modules = ["RediSearch"]
  }
}`, cloudAccountName, name)
}

// Generates Terraform configuration for the database
func formatDatabaseConfig(name, cloudAccountName, password string, extraConfig string) string {
	return proSubscriptionQPFBoilerplate(name, cloudAccountName) + fmt.Sprintf(`
resource "rediscloud_subscription_database" "example" {
    subscription_id = rediscloud_subscription.example.id
    name = "example"
    protocol = "redis"
    dataset_size_in_gb = 3
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
    query_performance_factor = "2x"

    alert {
        name = "dataset-size"
        value = 40
    }

    tags = {
        "market" = "emea"
        "material" = "cardboard"
    }

    %s
}`, password, extraConfig)
}

// Generic test helper for error cases
func testErrorCase(t *testing.T, config string, expectedError *regexp.Regexp) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: expectedError,
			},
		},
	})
}

func TestAccResourceRedisCloudProDatabase_qpf(t *testing.T) {
	name := acctest.RandomWithPrefix(testResourcePrefix)
	password := acctest.RandString(20)
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	var subId int

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: formatDatabaseConfig(name, testCloudAccountName, password, `modules = [{ name = "RediSearch" }]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("rediscloud_subscription_database.example", "name", "example"),
					resource.TestCheckResourceAttr("rediscloud_subscription_database.example", "protocol", "redis"),
					resource.TestCheckResourceAttr("rediscloud_subscription_database.example", "dataset_size_in_gb", "3"),
					resource.TestCheckResourceAttr("rediscloud_subscription_database.example", "query_performance_factor", "2x"),
					resource.TestCheckResourceAttr("rediscloud_subscription_database.example", "tags.market", "emea"),
					resource.TestCheckResourceAttr("rediscloud_subscription_database.example", "tags.material", "cardboard"),

					func(s *terraform.State) error {
						r := s.RootModule().Resources["rediscloud_subscription.example"]

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
						if !listDb.Next() {
							return fmt.Errorf("no database found: %s", listDb.Err())
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

func TestAccResourceRedisCloudProDatabase_missingModule(t *testing.T) {
	name := acctest.RandomWithPrefix(testResourcePrefix)
	password := acctest.RandString(20)
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	config := formatDatabaseConfig(name, testCloudAccountName, password, "")

	testErrorCase(t, config, regexp.MustCompile("query_performance_factor\" requires the \"modules\" key to be explicitly defined in HCL"))
}

func TestAccResourceRedisCloudProDatabase_missingRediSearchModule(t *testing.T) {
	name := acctest.RandomWithPrefix(testResourcePrefix)
	password := acctest.RandString(20)
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	config := formatDatabaseConfig(name, testCloudAccountName, password, `modules = [{ name = "RediBloom" }]`)

	testErrorCase(t, config, regexp.MustCompile("query_performance_factor\" requires the \"modules\" list to contain \"RediSearch"))
}
