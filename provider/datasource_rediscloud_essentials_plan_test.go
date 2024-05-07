package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"regexp"
	"testing"
)

func TestAccDataSourceRedisCloudEssentialsPlan_basic(t *testing.T) {

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      nil, // Essentials Plans aren't managed by this provider
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceRedisCloudEssentialsPlan,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.basic", "id", "34843"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.basic", "name", "30MB"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.basic", "size", "30"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.basic", "size_measurement_unit", "MB"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.basic", "cloud_provider", "AWS"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.basic", "region", "us-east-1"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.basic", "region_id", "1"),
					resource.TestCheckResourceAttrSet("data.rediscloud_essentials_plan.basic", "price"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.basic", "price_currency", "USD"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.basic", "price_period", "Month"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.basic", "maximum_databases", "1"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.basic", "maximum_throughput", "100"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.basic", "maximum_bandwidth_in_gb", "5"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.basic", "availability", "No replication"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.basic", "connections", "30"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.basic", "cidr_allow_rules", "1"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.basic", "support_data_persistence", "false"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.basic", "support_instant_and_daily_backups", "false"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.basic", "support_replication", "false"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.basic", "support_clustering", "false"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.basic", "supported_alerts.#", "2"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.basic", "customer_support", "Basic"),
				),
			},
		},
	})
}

func TestAccDataSourceRedisCloudEssentialsPlan_azure(t *testing.T) {

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      nil, // Essentials Plans aren't managed by this provider
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceRedisCloudEssentialsPlanAzure,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.azure", "id", "35008"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.azure", "name", "Single-Zone_Persistence_1GB"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.azure", "size", "1"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.azure", "size_measurement_unit", "GB"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.azure", "cloud_provider", "Azure"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.azure", "region", "west-us"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.azure", "region_id", "17"),
					resource.TestCheckResourceAttrSet("data.rediscloud_essentials_plan.azure", "price"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.azure", "price_currency", "USD"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.azure", "price_period", "Month"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.azure", "maximum_databases", "1"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.azure", "maximum_throughput", "2000"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.azure", "maximum_bandwidth_in_gb", "200"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.azure", "availability", "Single-zone"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.azure", "connections", "1024"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.azure", "cidr_allow_rules", "8"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.azure", "support_data_persistence", "true"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.azure", "support_instant_and_daily_backups", "true"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.azure", "support_replication", "true"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.azure", "support_clustering", "false"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.azure", "supported_alerts.#", "5"),
					resource.TestCheckResourceAttr("data.rediscloud_essentials_plan.azure", "customer_support", "Standard"),
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
