package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"regexp"
	"testing"
)

func TestAccDataSourceRedisCloudEssentialsPlan_basic(t *testing.T) {

	const datasource = "data.rediscloud_essentials_plan.basic"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      nil, // Essentials Plans aren't managed by this provider
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceRedisCloudEssentialsPlan,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(datasource, "id", "34843"),
					resource.TestCheckResourceAttr(datasource, "name", "30MB"),
					resource.TestCheckResourceAttr(datasource, "size", "30"),
					resource.TestCheckResourceAttr(datasource, "size_measurement_unit", "MB"),
					resource.TestCheckResourceAttr(datasource, "cloud_provider", "AWS"),
					resource.TestCheckResourceAttr(datasource, "region", "us-east-1"),
					resource.TestCheckResourceAttr(datasource, "region_id", "1"),
					resource.TestCheckResourceAttrSet(datasource, "price"),
					resource.TestCheckResourceAttr(datasource, "price_currency", "USD"),
					resource.TestCheckResourceAttr(datasource, "price_period", "Month"),
					resource.TestCheckResourceAttr(datasource, "maximum_databases", "1"),
					resource.TestCheckResourceAttr(datasource, "maximum_throughput", "100"),
					resource.TestCheckResourceAttr(datasource, "maximum_bandwidth_in_gb", "5"),
					resource.TestCheckResourceAttr(datasource, "availability", "No replication"),
					resource.TestCheckResourceAttr(datasource, "connections", "30"),
					resource.TestCheckResourceAttr(datasource, "cidr_allow_rules", "1"),
					resource.TestCheckResourceAttr(datasource, "support_data_persistence", "false"),
					resource.TestCheckResourceAttr(datasource, "support_instant_and_daily_backups", "false"),
					resource.TestCheckResourceAttr(datasource, "support_replication", "false"),
					resource.TestCheckResourceAttr(datasource, "support_clustering", "false"),
					resource.TestCheckResourceAttr(datasource, "supported_alerts.#", "2"),
					resource.TestCheckResourceAttr(datasource, "customer_support", "Basic"),
				),
			},
		},
	})
}

func TestAccDataSourceRedisCloudEssentialsPlan_azure(t *testing.T) {

	const azureResource = "data.rediscloud_essentials_plan.azure"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      nil, // Essentials Plans aren't managed by this provider
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceRedisCloudEssentialsPlanAzure,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(azureResource, "id", "35008"),
					resource.TestCheckResourceAttr(azureResource, "name", "Single-Zone_Persistence_1GB"),
					resource.TestCheckResourceAttr(azureResource, "size", "1"),
					resource.TestCheckResourceAttr(azureResource, "size_measurement_unit", "GB"),
					resource.TestCheckResourceAttr(azureResource, "cloud_provider", "Azure"),
					resource.TestCheckResourceAttr(azureResource, "region", "west-us"),
					resource.TestCheckResourceAttr(azureResource, "region_id", "17"),
					resource.TestCheckResourceAttrSet(azureResource, "price"),
					resource.TestCheckResourceAttr(azureResource, "price_currency", "USD"),
					resource.TestCheckResourceAttr(azureResource, "price_period", "Month"),
					resource.TestCheckResourceAttr(azureResource, "maximum_databases", "1"),
					resource.TestCheckResourceAttr(azureResource, "maximum_throughput", "2000"),
					resource.TestCheckResourceAttr(azureResource, "maximum_bandwidth_in_gb", "200"),
					resource.TestCheckResourceAttr(azureResource, "availability", "Single-zone"),
					resource.TestCheckResourceAttr(azureResource, "connections", "1024"),
					resource.TestCheckResourceAttr(azureResource, "cidr_allow_rules", "8"),
					resource.TestCheckResourceAttr(azureResource, "support_data_persistence", "true"),
					resource.TestCheckResourceAttr(azureResource, "support_instant_and_daily_backups", "true"),
					resource.TestCheckResourceAttr(azureResource, "support_replication", "true"),
					resource.TestCheckResourceAttr(azureResource, "support_clustering", "false"),
					resource.TestCheckResourceAttr(azureResource, "supported_alerts.#", "5"),
					resource.TestCheckResourceAttr(azureResource, "customer_support", "Standard"),
				),
			},
		},
	})
}

func TestAccDataSourceRedisCloudEssentialsPlan_subscriptionId(t *testing.T) {

	const exampleResource = "data.rediscloud_essentials_plan.example"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      nil, // Essentials Plans aren't managed by this provider

		Steps: []resource.TestStep{
			{
				Config: testAccResourceRedisCloudPaidEssentialsSubscriptionDataSource,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(exampleResource, "id", "34858"),
					resource.TestCheckResourceAttr(exampleResource, "name", "250MB"),
					resource.TestCheckResourceAttrSet(exampleResource, "subscription_id"),
					resource.TestCheckResourceAttr(exampleResource, "size", "250"),
					resource.TestCheckResourceAttr(exampleResource, "size_measurement_unit", "MB"),
					resource.TestCheckResourceAttr(exampleResource, "cloud_provider", "AWS"),
					resource.TestCheckResourceAttr(exampleResource, "region", "us-east-1"),
					resource.TestCheckResourceAttr(exampleResource, "region_id", "1"),
					resource.TestCheckResourceAttrSet(exampleResource, "price"),
					resource.TestCheckResourceAttr(exampleResource, "price_currency", "USD"),
					resource.TestCheckResourceAttr(exampleResource, "price_period", "Month"),
					resource.TestCheckResourceAttr(exampleResource, "maximum_databases", "1"),
					resource.TestCheckResourceAttr(exampleResource, "maximum_throughput", "1000"),
					resource.TestCheckResourceAttr(exampleResource, "maximum_bandwidth_in_gb", "100"),
					resource.TestCheckResourceAttr(exampleResource, "availability", "No replication"),
					resource.TestCheckResourceAttr(exampleResource, "connections", "256"),
					resource.TestCheckResourceAttr(exampleResource, "cidr_allow_rules", "4"),
					resource.TestCheckResourceAttr(exampleResource, "support_data_persistence", "false"),
					resource.TestCheckResourceAttr(exampleResource, "support_instant_and_daily_backups", "true"),
					resource.TestCheckResourceAttr(exampleResource, "support_replication", "false"),
					resource.TestCheckResourceAttr(exampleResource, "support_clustering", "false"),
					resource.TestCheckResourceAttr(exampleResource, "supported_alerts.#", "5"),
					resource.TestCheckResourceAttr(exampleResource, "customer_support", "Standard"),
				),
			},
		},
	})
}

func TestAccDataSourceRedisCloudEssentialsPlan_ambiguous(t *testing.T) {

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      nil, // Essentials Plans aren't managed by this provider
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceRedisCloudEssentialsPlanAmbiguous,
				ExpectError: regexp.MustCompile("Your query returned more than one result. Please change try a more specific search criteria and try again."),
			},
		},
	})
}

func TestAccDataSourceRedisCloudEssentialsPlan_impossible(t *testing.T) {

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      nil, // Essentials Plans aren't managed by this provider
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceRedisCloudEssentialsPlanImpossible,
				ExpectError: regexp.MustCompile("Your query returned no results. Please change your search criteria and try again."),
			},
		},
	})
}

const testAccDataSourceRedisCloudEssentialsPlan = `
data "rediscloud_essentials_plan" "basic" {
  id = 34843
}
`

const testAccDataSourceRedisCloudEssentialsPlanAzure = `
data "rediscloud_essentials_plan" "azure" {
  id = 35008
  cloud_provider = "Azure"
}
`

const testAccDataSourceRedisCloudEssentialsPlanAmbiguous = `
data "rediscloud_essentials_plan" "ambiguous" {
  name = "30MB"
}
`

const testAccDataSourceRedisCloudEssentialsPlanImpossible = `
data "rediscloud_essentials_plan" "impossible" {
  name = "There should never be a essentials plan with this name!"
}
`

const testAccResourceRedisCloudPaidEssentialsSubscriptionDataSource = `
data "rediscloud_payment_method" "card" {
	card_type = "Visa"
}

data "rediscloud_essentials_plan" "fixed" {
	name = "250MB"
	cloud_provider = "AWS"
	region = "us-east-1"
}

resource "rediscloud_essentials_subscription" "fixed" {
	name = "fixed subscription test"
	plan_id = data.rediscloud_essentials_plan.fixed.id
	payment_method_id = data.rediscloud_payment_method.card.id
}

data "rediscloud_essentials_plan" "example" {
	name = data.rediscloud_essentials_plan.fixed.name
	subscription_id = rediscloud_essentials_subscription.fixed.id
}
`
