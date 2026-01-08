package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	proSubscriptionConfigPath           = "./pro/testdata/pro_subscription_boilerplate.tf"
	proSubscriptionDataSourceConfigPath = "./pro/testdata/pro_subscription_data_source.tf"
	AADatabaseProDatasourceConfigPath   = "./pro/testdata/active_active_database_with_pro_data_source.tf"
)

func TestAccDataSourceRedisCloudProSubscription_basic(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := acctest.RandomWithPrefix("tf-test")

	const resourceName = "rediscloud_subscription.example"
	const dataSourceName = "data.rediscloud_subscription.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	proSubConfig := utils.GetTestConfig(t, proSubscriptionConfigPath)
	proSubConfig = fmt.Sprintf(proSubConfig, testCloudAccountName, name)

	proSubDataConfig := utils.GetTestConfig(t, proSubscriptionDataSourceConfigPath)
	proSubDataConfig = fmt.Sprintf(proSubDataConfig, name)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: proSubConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(resourceName, "name", regexp.MustCompile(name)),
				),
			},
			{
				Config: proSubDataConfig + proSubConfig,
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

	name := acctest.RandomWithPrefix(testResourcePrefix)
	password := acctest.RandString(20)

	config := utils.GetTestConfig(t, AADatabaseProDatasourceConfigPath)
	config = fmt.Sprintf(config, name+"-subscription", name+"-database", password)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile("Your query returned no results. Please change your search criteria and try again."),
			},
		},
	})
}
