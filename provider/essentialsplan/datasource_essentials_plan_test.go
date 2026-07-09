package essentialsplan_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAccDataSourceRedisCloudEssentialsPlan_basic(t *testing.T) {

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testhelpers.BasicPreCheck(t) },
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             nil, // Essentials Plans aren't managed by this provider
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/datasource_by_id.tf"),
				ConfigVariables: config.Variables{
					"id": config.IntegerVariable(34843),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_id", "id", "34843"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_id", "name", "30MB"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_id", "size", "30"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_id", "size_measurement_unit", "MB"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_id", "cloud_provider", "AWS"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_id", "region", "us-east-1"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_id", "region_id", "1"),
					resource.TestCheckResourceAttrSet(
						"data.rediscloud_essentials_plan.by_id", "price"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_id", "price_currency", "USD"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_id", "price_period", "Month"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_id", "maximum_databases", "1"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_id", "maximum_throughput", "100"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_id", "maximum_bandwidth_in_gb", "5"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_id", "availability", "No replication"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_id", "connections", "30"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_id", "cidr_allow_rules", "1"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_id", "support_data_persistence", "false"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_id", "support_instant_and_daily_backups", "false"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_id", "support_replication", "false"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_id", "support_clustering", "false"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_id", "supported_alerts.#", "2"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_id", "customer_support", "Basic"),
				),
			},
		},
	})
}

func TestAccDataSourceRedisCloudEssentialsPlan_azure(t *testing.T) {

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testhelpers.BasicPreCheck(t) },
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             nil, // Essentials Plans aren't managed by this provider
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/datasource_by_provider.tf"),
				ConfigVariables: config.Variables{
					"id":            config.IntegerVariable(35008),
					"provider_name": config.StringVariable("Azure"),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_provider", "id", "35008"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_provider", "name", "Single-Zone_Persistence_1GB"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_provider", "size", "1"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_provider", "size_measurement_unit", "GB"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_provider", "cloud_provider", "Azure"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_provider", "region", "west-us"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_provider", "region_id", "17"),
					resource.TestCheckResourceAttrSet(
						"data.rediscloud_essentials_plan.by_provider", "price"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_provider", "price_currency", "USD"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_provider", "price_period", "Month"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_provider", "maximum_databases", "1"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_provider", "maximum_throughput", "2000"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_provider", "maximum_bandwidth_in_gb", "200"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_provider", "availability", "Single-zone"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_provider", "connections", "1024"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_provider", "cidr_allow_rules", "8"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_provider", "support_data_persistence", "true"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_provider", "support_instant_and_daily_backups", "true"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_provider", "support_replication", "true"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_provider", "support_clustering", "false"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_provider", "supported_alerts.#", "5"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.by_provider", "customer_support", "Standard"),
				),
			},
		},
	})
}

func TestAccDataSourceRedisCloudEssentialsPlan_ambiguous(t *testing.T) {

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testhelpers.BasicPreCheck(t) },
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             nil, // Essentials Plans aren't managed by this provider
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/datasource_by_name.tf"),
				ConfigVariables: config.Variables{
					"name": config.StringVariable("30MB"),
				},
				ExpectError: regexp.MustCompile("Your query returned more than one result. Please change try a more specific search criteria and try again."),
			},
		},
	})
}

func TestAccDataSourceRedisCloudEssentialsPlan_impossible(t *testing.T) {

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testhelpers.BasicPreCheck(t) },
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             nil, // Essentials Plans aren't managed by this provider
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/datasource_by_name.tf"),
				ConfigVariables: config.Variables{
					"name": config.StringVariable("There should never be a essentials plan with this name!"),
				},
				ExpectError: regexp.MustCompile("Your query returned no results. Please change your search criteria and try again."),
			},
		},
	})
}

func TestAccDataSourceRedisCloudEssentialsPlan_subscription(t *testing.T) {

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testhelpers.BasicPreCheck(t) },
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             nil, // Essentials Plans aren't managed by this provider
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/datasource_with_subscription.tf"),
				ConfigVariables: config.Variables{
					"card_type":         config.StringVariable("Visa"),
					"last_four_numbers": config.StringVariable("5556"),
					"plan_name":         config.StringVariable("250MB"),
					"cloud_provider":    config.StringVariable("AWS"),
					"region":            config.StringVariable("us-east-1"),
					"subscription_name": config.StringVariable("fixed subscription test"),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.with_subscription", "id", "34858"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.with_subscription", "name", "250MB"),
					resource.TestCheckResourceAttrSet(
						"data.rediscloud_essentials_plan.with_subscription", "subscription_id"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.with_subscription", "size", "250"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.with_subscription", "size_measurement_unit", "MB"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.with_subscription", "cloud_provider", "AWS"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.with_subscription", "region", "us-east-1"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.with_subscription", "region_id", "1"),
					resource.TestCheckResourceAttrSet(
						"data.rediscloud_essentials_plan.with_subscription", "price"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.with_subscription", "price_currency", "USD"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.with_subscription", "price_period", "Month"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.with_subscription", "maximum_databases", "1"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.with_subscription", "maximum_throughput", "1000"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.with_subscription", "maximum_bandwidth_in_gb", "100"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.with_subscription", "availability", "No replication"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.with_subscription", "connections", "256"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.with_subscription", "cidr_allow_rules", "4"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.with_subscription", "support_data_persistence", "false"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.with_subscription", "support_instant_and_daily_backups", "true"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.with_subscription", "support_replication", "false"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.with_subscription", "support_clustering", "false"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.with_subscription", "supported_alerts.#", "5"),
					resource.TestCheckResourceAttr(
						"data.rediscloud_essentials_plan.with_subscription", "customer_support", "Standard"),
				),
			},
		},
	})
}
