package provider

import (
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

const (
	proSubscriptionConfigPath         = "./pro/testdata/pro_subscription_boilerplate.tf"
	AADatabaseProDatasourceConfigPath = "./pro/testdata/active_active_database_with_pro_data_source.tf"
)

func TestAccDataSourceRedisCloudProSubscription_basic(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := testRandomWithPrefix()

	const resourceName = "rediscloud_subscription.example"
	const dataSourceName = "data.rediscloud_subscription.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile(proSubscriptionConfigPath),
				ConfigVariables: config.Variables{
					"cloud_account_name": config.StringVariable(testCloudAccountName),
					"subscription_name":  config.StringVariable(name),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(resourceName, "name", regexp.MustCompile(name)),
				),
			},
			{
				ConfigFile: config.StaticFile(proSubscriptionConfigPath),
				ConfigVariables: config.Variables{
					"cloud_account_name": config.StringVariable(testCloudAccountName),
					"subscription_name":  config.StringVariable(name),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(dataSourceName, "name", regexp.MustCompile(name)),
					resource.TestCheckResourceAttr(dataSourceName, "payment_method", "credit-card"),
					resource.TestCheckResourceAttrSet(dataSourceName, "payment_method_id"),
					resource.TestMatchResourceAttr(dataSourceName, "memory_storage", regexp.MustCompile("ram")),
					resource.TestCheckResourceAttr(dataSourceName, "number_of_databases", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "cloud_provider.0.provider", "AWS"),
					resource.TestCheckResourceAttrSet(dataSourceName, "cloud_provider.0.cloud_account_id"),
					resource.TestCheckResourceAttrSet(dataSourceName, "cloud_provider.0.aws_account_id"),
					resource.TestCheckResourceAttr(dataSourceName, "cloud_provider.0.region.0.region", "eu-west-1"),
					resource.TestCheckResourceAttr(dataSourceName, "cloud_provider.0.region.0.networks.0.networking_deployment_cidr", "10.0.0.0/24"),
					resource.TestCheckResourceAttr(dataSourceName, "status", "active"),
					resource.TestCheckResourceAttr(dataSourceName, "customer_managed_key_enabled", "false"),
					resource.TestCheckResourceAttr(dataSourceName, "customer_managed_key_deletion_grace_period", ""),
					resource.TestCheckResourceAttr(dataSourceName, "customer_managed_key_redis_service_account", ""),
					resource.TestCheckResourceAttr(dataSourceName, "public_endpoint_access", "true"),
					resource.TestCheckResourceAttrSet(dataSourceName, "prometheus_endpoint"),

					resource.TestCheckResourceAttr(dataSourceName, "pricing.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "pricing.0.type", "Shards"),
					resource.TestCheckResourceAttr(dataSourceName, "pricing.0.type_details", "micro"),
					resource.TestCheckResourceAttr(dataSourceName, "pricing.0.quantity", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "pricing.0.quantity_measurement", "shards"),
					resource.TestCheckResourceAttrSet(dataSourceName, "pricing.0.price_per_unit"),
					resource.TestCheckResourceAttr(dataSourceName, "pricing.0.price_currency", "USD"),
					resource.TestCheckResourceAttr(dataSourceName, "pricing.0.price_period", "hour"),
				),
			},
		},
	})
}

func TestAccDataSourceRedisCloudProSubscription_ignoresAA(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := testRandomWithPrefix()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile(AADatabaseProDatasourceConfigPath),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(name + "-subscription"),
					"database_password": config.StringVariable(name + "-database"),
				},
				ExpectError: regexp.MustCompile("Your query returned no results. Please change your search criteria and try again."),
			},
		},
	})
}
