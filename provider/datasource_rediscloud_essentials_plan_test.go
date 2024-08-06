package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"regexp"
	"testing"
)

func TestAccDataSourceRedisCloudEssentialsPlan_basic(t *testing.T) {

	const basicPlan = "data.rediscloud_essentials_plan.basic"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      nil, // Essentials Plans aren't managed by this provider
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceRedisCloudEssentialsPlan,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(basicPlan, "id", "34843"),
					resource.TestCheckResourceAttr(basicPlan, "name", "30MB"),
					resource.TestCheckResourceAttr(basicPlan, "size", "30"),
					resource.TestCheckResourceAttr(basicPlan, "size_measurement_unit", "MB"),
					resource.TestCheckResourceAttr(basicPlan, "cloud_provider", "AWS"),
					resource.TestCheckResourceAttr(basicPlan, "region", "us-east-1"),
					resource.TestCheckResourceAttr(basicPlan, "region_id", "1"),
					resource.TestCheckResourceAttrSet(basicPlan, "price"),
					resource.TestCheckResourceAttr(basicPlan, "price_currency", "USD"),
					resource.TestCheckResourceAttr(basicPlan, "price_period", "Month"),
					resource.TestCheckResourceAttr(basicPlan, "maximum_databases", "1"),
					resource.TestCheckResourceAttr(basicPlan, "maximum_throughput", "100"),
					resource.TestCheckResourceAttr(basicPlan, "maximum_bandwidth_in_gb", "5"),
					resource.TestCheckResourceAttr(basicPlan, "availability", "No replication"),
					resource.TestCheckResourceAttr(basicPlan, "connections", "30"),
					resource.TestCheckResourceAttr(basicPlan, "cidr_allow_rules", "1"),
					resource.TestCheckResourceAttr(basicPlan, "support_data_persistence", "false"),
					resource.TestCheckResourceAttr(basicPlan, "support_instant_and_daily_backups", "false"),
					resource.TestCheckResourceAttr(basicPlan, "support_replication", "false"),
					resource.TestCheckResourceAttr(basicPlan, "support_clustering", "false"),
					resource.TestCheckResourceAttr(basicPlan, "supported_alerts.#", "2"),
					resource.TestCheckResourceAttr(basicPlan, "customer_support", "Basic"),
				),
			},
		},
	})
}

func TestAccDataSourceRedisCloudEssentialsPlan_azure(t *testing.T) {

	const azurePlan = "data.rediscloud_essentials_plan.azure"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      nil, // Essentials Plans aren't managed by this provider
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceRedisCloudEssentialsPlanAzure,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(azurePlan, "id", "35008"),
					resource.TestCheckResourceAttr(azurePlan, "name", "Single-Zone_Persistence_1GB"),
					resource.TestCheckResourceAttr(azurePlan, "size", "1"),
					resource.TestCheckResourceAttr(azurePlan, "size_measurement_unit", "GB"),
					resource.TestCheckResourceAttr(azurePlan, "cloud_provider", "Azure"),
					resource.TestCheckResourceAttr(azurePlan, "region", "west-us"),
					resource.TestCheckResourceAttr(azurePlan, "region_id", "17"),
					resource.TestCheckResourceAttrSet(azurePlan, "price"),
					resource.TestCheckResourceAttr(azurePlan, "price_currency", "USD"),
					resource.TestCheckResourceAttr(azurePlan, "price_period", "Month"),
					resource.TestCheckResourceAttr(azurePlan, "maximum_databases", "1"),
					resource.TestCheckResourceAttr(azurePlan, "maximum_throughput", "2000"),
					resource.TestCheckResourceAttr(azurePlan, "maximum_bandwidth_in_gb", "200"),
					resource.TestCheckResourceAttr(azurePlan, "availability", "Single-zone"),
					resource.TestCheckResourceAttr(azurePlan, "connections", "1024"),
					resource.TestCheckResourceAttr(azurePlan, "cidr_allow_rules", "8"),
					resource.TestCheckResourceAttr(azurePlan, "support_data_persistence", "true"),
					resource.TestCheckResourceAttr(azurePlan, "support_instant_and_daily_backups", "true"),
					resource.TestCheckResourceAttr(azurePlan, "support_replication", "true"),
					resource.TestCheckResourceAttr(azurePlan, "support_clustering", "false"),
					resource.TestCheckResourceAttr(azurePlan, "supported_alerts.#", "5"),
					resource.TestCheckResourceAttr(azurePlan, "customer_support", "Standard"),
				),
			},
		},
	})
}

func TestAccDataSourceRedisCloudEssentialsPlan_subscriptionId(t *testing.T) {

	const examplePlan = "data.rediscloud_essentials_plan.example"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      nil, // Essentials Plans aren't managed by this provider

		Steps: []resource.TestStep{
			{
				Config: testAccResourceRedisCloudPaidEssentialsSubscriptionDataSource,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(examplePlan, "id", "34858"),
					resource.TestCheckResourceAttr(examplePlan, "name", "250MB"),
					resource.TestCheckResourceAttrSet(examplePlan, "subscription_id"),
					resource.TestCheckResourceAttr(examplePlan, "size", "250"),
					resource.TestCheckResourceAttr(examplePlan, "size_measurement_unit", "MB"),
					resource.TestCheckResourceAttr(examplePlan, "cloud_provider", "AWS"),
					resource.TestCheckResourceAttr(examplePlan, "region", "us-east-1"),
					resource.TestCheckResourceAttr(examplePlan, "region_id", "1"),
					resource.TestCheckResourceAttrSet(examplePlan, "price"),
					resource.TestCheckResourceAttr(examplePlan, "price_currency", "USD"),
					resource.TestCheckResourceAttr(examplePlan, "price_period", "Month"),
					resource.TestCheckResourceAttr(examplePlan, "maximum_databases", "1"),
					resource.TestCheckResourceAttr(examplePlan, "maximum_throughput", "1000"),
					resource.TestCheckResourceAttr(examplePlan, "maximum_bandwidth_in_gb", "100"),
					resource.TestCheckResourceAttr(examplePlan, "availability", "No replication"),
					resource.TestCheckResourceAttr(examplePlan, "connections", "256"),
					resource.TestCheckResourceAttr(examplePlan, "cidr_allow_rules", "4"),
					resource.TestCheckResourceAttr(examplePlan, "support_data_persistence", "false"),
					resource.TestCheckResourceAttr(examplePlan, "support_instant_and_daily_backups", "true"),
					resource.TestCheckResourceAttr(examplePlan, "support_replication", "false"),
					resource.TestCheckResourceAttr(examplePlan, "support_clustering", "false"),
					resource.TestCheckResourceAttr(examplePlan, "supported_alerts.#", "5"),
					resource.TestCheckResourceAttr(examplePlan, "customer_support", "Standard"),
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
