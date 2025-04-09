package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// Generates the base Terraform config for a Pro Subscription with QPF
func formatSubscriptionConfig(name, cloudAccountName, qpf, extraConfig string) string {
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
    query_performance_factor = "%s"
    
	%s
  }
}`, cloudAccountName, name, qpf, extraConfig)
}

// Generic test helper for error cases
func testSubErrorCase(t *testing.T, config string, expectedError *regexp.Regexp) {
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

func TestAccResourceRedisCloudProSubscription_qpf(t *testing.T) {
	name := acctest.RandomWithPrefix(testResourcePrefix)
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")
	const resourceName = "rediscloud_subscription.example"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: formatSubscriptionConfig(name, testCloudAccountName, "2x", `modules = ["RediSearch"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "payment_method", "credit-card"),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.provider", "AWS"),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.region.0.preferred_availability_zones.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.region.0.networks.0.networking_subnet_id"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.average_item_size_in_bytes", "0"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.dataset_size_in_gb", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.query_performance_factor", "2x"),

					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.modules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.modules.0", "RediSearch"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.quantity", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.replication", "false"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.support_oss_cluster_api", "false"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.throughput_measurement_by", "operations-per-second"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.throughput_measurement_value", "1000"),
				),
			},
		},
	})
}

func TestAccResourceRedisCloudProSubscription_missingModule(t *testing.T) {
	name := acctest.RandomWithPrefix(testResourcePrefix)
	password := acctest.RandString(20)
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	config := formatDatabaseConfig(name, testCloudAccountName, password, "4x", "")

	testSubErrorCase(t, config, regexp.MustCompile("query_performance_factor\" requires the \"modules\" key to be explicitly defined in HCL"))
}

func TestAccResourceRedisCloudProSubscription_missingRediSearchModule(t *testing.T) {
	name := acctest.RandomWithPrefix(testResourcePrefix)
	password := acctest.RandString(20)
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	config := formatDatabaseConfig(name, testCloudAccountName, password, "4x", `modules = [{ name = "RediBloom" }]`)

	testSubErrorCase(t, config, regexp.MustCompile("query_performance_factor\" requires the \"modules\" list to contain \"RediSearch"))
}

func TestAccResourceRedisCloudProSubscription_invalidQueryPerformanceFactors(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-test")
	password := acctest.RandString(20)
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	config := formatDatabaseConfig(name, testCloudAccountName, password, "5x", `modules = [{ name = "RediSearch" }]`)

	testSubErrorCase(t, config, regexp.MustCompile(`"creation_plan\.0\.query_performance_factor" must be an even value between 2x and 16x \(inclusive\), got: 5x`))
}

func TestAccResourceRedisCloudProSubscription_invalidQueryPerformanceFactors_outOfRange(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-test")
	password := acctest.RandString(20)
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	config := formatDatabaseConfig(name, testCloudAccountName, password, "30x", `modules = [{ name = "RediSearch" }]`)

	testSubErrorCase(t, config, regexp.MustCompile(`"creation_plan\.0\.query_performance_factor" must be an even value between 2x and 16x \(inclusive\), got: 30x`))
}
