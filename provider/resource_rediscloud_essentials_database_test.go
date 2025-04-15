package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"regexp"
	"testing"
)

func TestAccResourceRedisCloudEssentialsDatabase_CRUDI(t *testing.T) {

	testAccRequiresEnvVar(t, "EXECUTE_TESTS")

	subscriptionName := acctest.RandomWithPrefix(testResourcePrefix)
	databaseName := subscriptionName + "-db"
	databaseNameUpdated := databaseName + "-updated"

	const resourceName = "rediscloud_essentials_database.example"
	const datasourceName = "data.rediscloud_essentials_database.example"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckEssentialsSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudEssentialsDatabaseBasic, subscriptionName, databaseName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Test the resource
					resource.TestMatchResourceAttr(resourceName, "id", regexp.MustCompile("^\\d+/\\d+$")),
					resource.TestCheckResourceAttrSet(resourceName, "subscription_id"),
					resource.TestCheckResourceAttrSet(resourceName, "db_id"),
					resource.TestCheckResourceAttr(resourceName, "name", databaseName),
					resource.TestCheckResourceAttr(resourceName, "protocol", "stack"),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider", "AWS"),
					resource.TestCheckResourceAttr(resourceName, "region", "eu-west-1"),
					resource.TestCheckResourceAttrSet(resourceName, "redis_version_compliance"),
					resource.TestCheckResourceAttr(resourceName, "resp_version", "resp3"),
					resource.TestCheckResourceAttr(resourceName, "data_persistence", "none"),
					resource.TestCheckResourceAttr(resourceName, "data_eviction", "volatile-lru"),
					resource.TestCheckResourceAttr(resourceName, "replication", "false"),
					resource.TestCheckResourceAttrSet(resourceName, "public_endpoint"),
					resource.TestCheckResourceAttr(resourceName, "private_endpoint", ""),
					resource.TestCheckResourceAttr(resourceName, "source_ips.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "alert.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "alert.0.name", "throughput-higher-than"),
					resource.TestCheckResourceAttr(resourceName, "alert.0.value", "80"),
					resource.TestCheckResourceAttr(resourceName, "enable_default_user", "true"),
					resource.TestCheckResourceAttr(resourceName, "password", "j43589rhe39f"),

					resource.TestCheckResourceAttr(resourceName, "enable_payg_features", "false"),
					resource.TestCheckResourceAttr(resourceName, "memory_limit_in_gb", "0"),
					resource.TestCheckResourceAttr(resourceName, "support_oss_cluster_api", "false"),
					resource.TestCheckResourceAttr(resourceName, "external_endpoint_for_oss_cluster_api", "false"),
					resource.TestCheckResourceAttr(resourceName, "enable_database_clustering", "false"),
					resource.TestCheckResourceAttr(resourceName, "regex_rules.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "enable_tls", "false"),

					resource.TestCheckResourceAttr(resourceName, "tags.environment", "production"),
					resource.TestCheckResourceAttr(resourceName, "tags.cost_center", "0700"),
					resource.TestCheckResourceAttr(resourceName, "tags.department", "finance"),

					// Test the datasource
					resource.TestMatchResourceAttr(datasourceName, "id", regexp.MustCompile("^\\d+/\\d+$")),
					resource.TestCheckResourceAttrSet(datasourceName, "subscription_id"),
					resource.TestCheckResourceAttrSet(datasourceName, "db_id"),
					resource.TestCheckResourceAttr(datasourceName, "name", databaseName),
					resource.TestCheckResourceAttr(datasourceName, "protocol", "stack"),
					resource.TestCheckResourceAttr(datasourceName, "cloud_provider", "AWS"),
					resource.TestCheckResourceAttr(datasourceName, "region", "eu-west-1"),
					resource.TestCheckResourceAttrSet(datasourceName, "redis_version_compliance"),
					resource.TestCheckResourceAttr(datasourceName, "resp_version", "resp3"),
					resource.TestCheckResourceAttr(datasourceName, "data_persistence", "none"),
					resource.TestCheckResourceAttr(datasourceName, "data_eviction", "volatile-lru"),
					resource.TestCheckResourceAttr(datasourceName, "replication", "false"),
					resource.TestCheckResourceAttrSet(datasourceName, "public_endpoint"),
					resource.TestCheckResourceAttr(datasourceName, "private_endpoint", ""),
					resource.TestCheckResourceAttr(datasourceName, "source_ips.#", "0"),
					resource.TestCheckResourceAttr(datasourceName, "alert.#", "1"),
					resource.TestCheckResourceAttr(datasourceName, "alert.0.name", "throughput-higher-than"),
					resource.TestCheckResourceAttr(datasourceName, "alert.0.value", "80"),
					resource.TestCheckResourceAttr(datasourceName, "enable_default_user", "true"),
					resource.TestCheckResourceAttr(datasourceName, "password", "j43589rhe39f"),

					resource.TestCheckResourceAttr(datasourceName, "memory_limit_in_gb", "0"),
					resource.TestCheckResourceAttr(datasourceName, "support_oss_cluster_api", "false"),
					resource.TestCheckResourceAttr(datasourceName, "external_endpoint_for_oss_cluster_api", "false"),
					resource.TestCheckResourceAttr(datasourceName, "enable_database_clustering", "false"),
					resource.TestCheckResourceAttr(datasourceName, "regex_rules.#", "0"),
					resource.TestCheckResourceAttr(datasourceName, "enable_tls", "false"),

					resource.TestCheckResourceAttr(datasourceName, "tags.environment", "production"),
					resource.TestCheckResourceAttr(datasourceName, "tags.cost_center", "0700"),
					resource.TestCheckResourceAttr(datasourceName, "tags.department", "finance"),
				),
			},
			{
				Config:      fmt.Sprintf(testAccResourceRedisCloudEssentialsDatabaseBasicWithUpperCaseTagKey, subscriptionName, databaseName),
				ExpectError: regexp.MustCompile("tag keys and values must be lower case, invalid entries: UpperCaseKey"),
			},
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudEssentialsDatabaseBasic, subscriptionName, databaseNameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Test the resource
					resource.TestMatchResourceAttr(resourceName, "id", regexp.MustCompile("^\\d+/\\d+$")),
					resource.TestCheckResourceAttrSet(resourceName, "subscription_id"),
					resource.TestCheckResourceAttrSet(resourceName, "db_id"),
					resource.TestCheckResourceAttr(resourceName, "name", databaseNameUpdated),
					resource.TestCheckResourceAttr(resourceName, "protocol", "stack"),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider", "AWS"),
					resource.TestCheckResourceAttr(resourceName, "region", "eu-west-1"),
					resource.TestCheckResourceAttrSet(resourceName, "redis_version_compliance"),
					resource.TestCheckResourceAttr(resourceName, "resp_version", "resp3"),
					resource.TestCheckResourceAttr(resourceName, "data_persistence", "none"),
					resource.TestCheckResourceAttr(resourceName, "data_eviction", "volatile-lru"),
					resource.TestCheckResourceAttr(resourceName, "replication", "false"),
					resource.TestCheckResourceAttrSet(resourceName, "public_endpoint"),
					resource.TestCheckResourceAttrSet(resourceName, "public_endpoint"),
					resource.TestCheckResourceAttr(resourceName, "private_endpoint", ""),
					resource.TestCheckResourceAttr(resourceName, "source_ips.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "alert.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "alert.0.name", "throughput-higher-than"),
					resource.TestCheckResourceAttr(resourceName, "alert.0.value", "80"),
					resource.TestCheckResourceAttr(resourceName, "enable_default_user", "true"),
					resource.TestCheckResourceAttr(resourceName, "password", "j43589rhe39f"),

					resource.TestCheckResourceAttr(resourceName, "enable_payg_features", "false"),
					resource.TestCheckResourceAttr(resourceName, "memory_limit_in_gb", "0"),
					resource.TestCheckResourceAttr(resourceName, "support_oss_cluster_api", "false"),
					resource.TestCheckResourceAttr(resourceName, "external_endpoint_for_oss_cluster_api", "false"),
					resource.TestCheckResourceAttr(resourceName, "enable_database_clustering", "false"),
					resource.TestCheckResourceAttr(resourceName, "regex_rules.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "enable_tls", "false"),

					// Test the datasource
					resource.TestMatchResourceAttr(datasourceName, "id", regexp.MustCompile("^\\d+/\\d+$")),
					resource.TestCheckResourceAttrSet(datasourceName, "subscription_id"),
					resource.TestCheckResourceAttrSet(datasourceName, "db_id"),
					resource.TestCheckResourceAttr(datasourceName, "name", databaseNameUpdated),
					resource.TestCheckResourceAttr(datasourceName, "protocol", "stack"),
					resource.TestCheckResourceAttr(datasourceName, "cloud_provider", "AWS"),
					resource.TestCheckResourceAttr(datasourceName, "region", "eu-west-1"),
					resource.TestCheckResourceAttrSet(datasourceName, "redis_version_compliance"),
					resource.TestCheckResourceAttr(datasourceName, "resp_version", "resp3"),
					resource.TestCheckResourceAttr(datasourceName, "data_persistence", "none"),
					resource.TestCheckResourceAttr(datasourceName, "data_eviction", "volatile-lru"),
					resource.TestCheckResourceAttr(datasourceName, "replication", "false"),
					resource.TestCheckResourceAttrSet(datasourceName, "public_endpoint"),
					resource.TestCheckResourceAttr(datasourceName, "private_endpoint", ""),
					resource.TestCheckResourceAttr(datasourceName, "source_ips.#", "0"),
					resource.TestCheckResourceAttr(datasourceName, "alert.#", "1"),
					resource.TestCheckResourceAttr(datasourceName, "alert.0.name", "throughput-higher-than"),
					resource.TestCheckResourceAttr(datasourceName, "alert.0.value", "80"),
					resource.TestCheckResourceAttr(datasourceName, "enable_default_user", "true"),
					resource.TestCheckResourceAttr(datasourceName, "password", "j43589rhe39f"),

					resource.TestCheckResourceAttr(datasourceName, "memory_limit_in_gb", "0"),
					resource.TestCheckResourceAttr(datasourceName, "support_oss_cluster_api", "false"),
					resource.TestCheckResourceAttr(datasourceName, "external_endpoint_for_oss_cluster_api", "false"),
					resource.TestCheckResourceAttr(datasourceName, "enable_database_clustering", "false"),
					resource.TestCheckResourceAttr(datasourceName, "regex_rules.#", "0"),
					resource.TestCheckResourceAttr(datasourceName, "enable_tls", "false"),
				),
			},
			{
				Config:                  fmt.Sprintf(testAccResourceRedisCloudEssentialsDatabaseBasic, subscriptionName, databaseName),
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password", "enable_payg_features"},
			},
		},
	})
}

const testAccResourceRedisCloudEssentialsDatabaseBasic = `
data "rediscloud_payment_method" "card" {
	card_type = "Visa"
}
data "rediscloud_essentials_plan" "example" {
	name = "250MB"
	cloud_provider = "AWS"
	region = "eu-west-1"
}
resource "rediscloud_essentials_subscription" "example" {
	name = "%s"
	plan_id = data.rediscloud_essentials_plan.example.id
	payment_method_id = data.rediscloud_payment_method.card.id
}
resource "rediscloud_essentials_database" "example" {
	subscription_id = rediscloud_essentials_subscription.example.id
	name = "%s"
	enable_default_user = true
	password = "j43589rhe39f"

	data_persistence = "none"
	replication = false

	alert {
		name = "throughput-higher-than"
		value = 80
	}

	tags = {
		"environment" = "production"
		"cost_center" = "0700"
		"department" = "finance"
	}
}
data "rediscloud_essentials_database" "example" {
	subscription_id = rediscloud_essentials_subscription.example.id
	name = rediscloud_essentials_database.example.name
}
`

const testAccResourceRedisCloudEssentialsDatabaseBasicWithUpperCaseTagKey = `
data "rediscloud_payment_method" "card" {
	card_type = "Visa"
}
data "rediscloud_essentials_plan" "example" {
	name = "250MB"
	cloud_provider = "AWS"
	region = "eu-west-1"
}
resource "rediscloud_essentials_subscription" "example" {
	name = "%s"
	plan_id = data.rediscloud_essentials_plan.example.id
	payment_method_id = data.rediscloud_payment_method.card.id
}
resource "rediscloud_essentials_database" "example" {
	subscription_id = rediscloud_essentials_subscription.example.id
	name = "%s"
	enable_default_user = true
	password = "j43589rhe39f"

	data_persistence = "none"
	replication = false

	alert {
		name = "throughput-higher-than"
		value = 80
	}

	tags = {
		"UpperCaseKey" = "invalid"
		"environment" = "production"
		"cost_center" = "0700"
		"department" = "finance"
	}
}
data "rediscloud_essentials_database" "example" {
	subscription_id = rediscloud_essentials_subscription.example.id
	name = rediscloud_essentials_database.example.name
}
`
