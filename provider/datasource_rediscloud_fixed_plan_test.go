package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"regexp"
	"testing"
)

func TestAccDataSourceRedisCloudFixedPlan_basic(t *testing.T) {

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      nil, // Fixed Plans aren't managed by this provider
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceRedisCloudFixedPlan,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.basic", "id", "34843"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.basic", "name", "30MB"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.basic", "size", "30"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.basic", "size_measurement_unit", "MB"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.basic", "cloud_provider", "AWS"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.basic", "region", "us-east-1"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.basic", "region_id", "1"),
					resource.TestCheckResourceAttrSet("data.rediscloud_fixed_plan.basic", "price"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.basic", "price_currency", "USD"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.basic", "price_period", "Month"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.basic", "maximum_databases", "1"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.basic", "maximum_throughput", "100"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.basic", "maximum_bandwidth_in_gb", "5"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.basic", "availability", "No replication"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.basic", "connections", "30"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.basic", "cidr_allow_rules", "1"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.basic", "support_data_persistence", "false"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.basic", "support_instant_and_daily_backups", "false"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.basic", "support_replication", "false"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.basic", "support_clustering", "false"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.basic", "supported_alerts.#", "2"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.basic", "customer_support", "Basic"),
				),
			},
		},
	})
}

func TestAccDataSourceRedisCloudFixedPlan_azure(t *testing.T) {

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      nil, // Fixed Plans aren't managed by this provider
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceRedisCloudFixedPlanAzure,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.azure", "id", "35008"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.azure", "name", "Single-Zone_Persistence_1GB"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.azure", "size", "1"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.azure", "size_measurement_unit", "GB"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.azure", "cloud_provider", "Azure"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.azure", "region", "west-us"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.azure", "region_id", "17"),
					resource.TestCheckResourceAttrSet("data.rediscloud_fixed_plan.azure", "price"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.azure", "price_currency", "USD"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.azure", "price_period", "Month"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.azure", "maximum_databases", "1"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.azure", "maximum_throughput", "2000"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.azure", "maximum_bandwidth_in_gb", "200"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.azure", "availability", "Single-zone"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.azure", "connections", "1024"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.azure", "cidr_allow_rules", "8"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.azure", "support_data_persistence", "true"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.azure", "support_instant_and_daily_backups", "true"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.azure", "support_replication", "true"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.azure", "support_clustering", "false"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.azure", "supported_alerts.#", "5"),
					resource.TestCheckResourceAttr("data.rediscloud_fixed_plan.azure", "customer_support", "Standard"),
				),
			},
		},
	})
}

func TestAccDataSourceRedisCloudFixedPlan_ambiguous(t *testing.T) {

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      nil, // Fixed Plans aren't managed by this provider
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceRedisCloudFixedPlanAmbiguous,
				ExpectError: regexp.MustCompile("Your query returned more than one result. Please change try a more specific search criteria and try again."),
			},
		},
	})
}

func TestAccDataSourceRedisCloudFixedPlan_impossible(t *testing.T) {

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      nil, // Fixed Plans aren't managed by this provider
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceRedisCloudFixedPlanImpossible,
				ExpectError: regexp.MustCompile("Your query returned no results. Please change your search criteria and try again."),
			},
		},
	})
}

const testAccDataSourceRedisCloudFixedPlan = `
data "rediscloud_fixed_plan" "basic" {
  id = 34843
}
`

const testAccDataSourceRedisCloudFixedPlanAzure = `
data "rediscloud_fixed_plan" "azure" {
  id = 35008
  cloud_provider = "Azure"
}
`

const testAccDataSourceRedisCloudFixedPlanAmbiguous = `
data "rediscloud_fixed_plan" "ambiguous" {
  name = "30MB"
}
`

const testAccDataSourceRedisCloudFixedPlanImpossible = `
data "rediscloud_fixed_plan" "impossible" {
  name = "There should never be a fixed plan with this name!"
}
`
