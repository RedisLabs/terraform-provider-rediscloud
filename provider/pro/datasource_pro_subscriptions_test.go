package pro_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

// TestAccDataSourceRedisCloudProSubscriptions_basic creates a pro subscription and reads
// it back through the list rediscloud_subscriptions data source using the exact-name
// filter (so the list narrows to just the one created here). It asserts the
// Subscription.List-derived fields are populated and that maintenance_windows / pricing
// are absent.
func TestAccDataSourceRedisCloudProSubscriptions_basic(t *testing.T) {
	name := utils.RandomWithPrefix()
	const dataSourceName = "data.rediscloud_subscriptions.example"
	cloudAccountName, cloudAccountCheck := envchecks.AWSBYOCValueAndCheck()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, envchecks.RedisCloudCheck, cloudAccountCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             checkProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/pro_subscriptions_data_source.tf"),
				ConfigVariables: config.Variables{
					"cloud_account_name": config.StringVariable(cloudAccountName),
					"subscription_name":  config.StringVariable(name),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					// The name filter is exact and the name is random, so the list narrows
					// to exactly the subscription created above.
					resource.TestCheckResourceAttr(dataSourceName, "subscriptions.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "subscriptions.0.name", name),
					resource.TestCheckResourceAttrSet(dataSourceName, "subscriptions.0.id"),
					resource.TestCheckResourceAttr(dataSourceName, "subscriptions.0.payment_method", "credit-card"),
					resource.TestCheckResourceAttrSet(dataSourceName, "subscriptions.0.payment_method_id"),
					resource.TestCheckResourceAttr(dataSourceName, "subscriptions.0.memory_storage", "ram"),
					// No database is created, so the subscription reports zero databases.
					resource.TestCheckResourceAttr(dataSourceName, "subscriptions.0.number_of_databases", "0"),
					resource.TestCheckResourceAttr(dataSourceName, "subscriptions.0.status", "active"),
					resource.TestCheckResourceAttr(dataSourceName, "subscriptions.0.public_endpoint_access", "true"),
					resource.TestCheckResourceAttrSet(dataSourceName, "subscriptions.0.prometheus_endpoint"),
					resource.TestCheckResourceAttr(dataSourceName, "subscriptions.0.customer_managed_key_enabled", "false"),
					resource.TestCheckResourceAttr(dataSourceName, "subscriptions.0.cloud_provider.0.provider", "AWS"),
					resource.TestCheckResourceAttrSet(dataSourceName, "subscriptions.0.cloud_provider.0.cloud_account_id"),
					resource.TestCheckResourceAttrSet(dataSourceName, "subscriptions.0.cloud_provider.0.aws_account_id"),
					resource.TestCheckResourceAttr(dataSourceName, "subscriptions.0.cloud_provider.0.region.0.region", "eu-west-1"),
					resource.TestCheckResourceAttr(dataSourceName, "subscriptions.0.cloud_provider.0.region.0.networks.0.networking_deployment_cidr", "10.0.0.0/24"),
					// maintenance_windows and pricing are intentionally not in the list data source
					resource.TestCheckNoResourceAttr(dataSourceName, "subscriptions.0.maintenance_windows.#"),
					resource.TestCheckNoResourceAttr(dataSourceName, "subscriptions.0.pricing.#"),
				),
			},
		},
	})
}

// TestAccDataSourceRedisCloudProSubscriptions_emptyWhenNoMatch verifies the list data
// source returns an empty list (not an error) when the name filter matches nothing. It
// creates no infrastructure, so it is cheap and needs only Redis Cloud credentials.
func TestAccDataSourceRedisCloudProSubscriptions_emptyWhenNoMatch(t *testing.T) {
	// A random name that will not match any subscription in the account.
	name := utils.RandomWithPrefix()
	const dataSourceName = "data.rediscloud_subscriptions.example"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, envchecks.RedisCloudCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/pro_subscriptions_data_source_empty.tf"),
				ConfigVariables: config.Variables{
					"filter_name": config.StringVariable(name),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "subscriptions.#", "0"),
				),
			},
		},
	})
}
