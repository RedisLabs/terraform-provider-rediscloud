package activeactive_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

const (
	aaSubscriptionDataSourceConfigDirectory = "testdata/active_active_subscription_datasource"
	aaSubscriptionResourceConfigFile        = aaSubscriptionDataSourceConfigDirectory + "/subscription.tf"
)

func TestAccDataSourceRedisCloudActiveActiveSubscription_basic(t *testing.T) {
	name := testRandomWithPrefix()
	const resourceName = "rediscloud_active_active_subscription.example"
	const dataSourceName = "data.rediscloud_active_active_subscription.example"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, envchecks.RedisCloudCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             checkAASubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile(aaSubscriptionResourceConfigFile),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(name),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.mode", "manual"),
				),
			},
			{
				ConfigDirectory: config.StaticDirectory(aaSubscriptionDataSourceConfigDirectory),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(name),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "name", name),
					resource.TestCheckResourceAttr(dataSourceName, "payment_method", "credit-card"),
					resource.TestCheckResourceAttrSet(dataSourceName, "payment_method_id"),
					resource.TestCheckResourceAttr(dataSourceName, "cloud_provider", "AWS"),
					resource.TestCheckResourceAttrSet(dataSourceName, "aws_account_id"),
					resource.TestCheckResourceAttr(dataSourceName, "number_of_databases", "0"),
					resource.TestCheckResourceAttr(dataSourceName, "status", "active"),
					resource.TestCheckResourceAttr(dataSourceName, "customer_managed_key_enabled", "false"),
					resource.TestCheckResourceAttr(dataSourceName, "public_endpoint_access", "true"),
					resource.TestCheckResourceAttr(dataSourceName, "maintenance_windows.0.mode", "manual"),
					resource.TestCheckResourceAttr(dataSourceName, "maintenance_windows.0.window.#", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "maintenance_windows.0.window.0.start_hour", "22"),
					resource.TestCheckResourceAttr(dataSourceName, "maintenance_windows.0.window.0.duration_in_hours", "8"),
					resource.TestCheckResourceAttr(dataSourceName, "maintenance_windows.0.window.0.days.#", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "maintenance_windows.0.window.0.days.0", "Monday"),
					resource.TestCheckResourceAttr(dataSourceName, "maintenance_windows.0.window.0.days.1", "Thursday"),
					resource.TestCheckResourceAttr(dataSourceName, "maintenance_windows.0.window.1.start_hour", "12"),
					resource.TestCheckResourceAttr(dataSourceName, "maintenance_windows.0.window.1.duration_in_hours", "6"),
					resource.TestCheckResourceAttr(dataSourceName, "maintenance_windows.0.window.1.days.#", "3"),
					resource.TestCheckResourceAttr(dataSourceName, "maintenance_windows.0.window.1.days.0", "Friday"),
					resource.TestCheckResourceAttr(dataSourceName, "maintenance_windows.0.window.1.days.1", "Saturday"),
					resource.TestCheckResourceAttr(dataSourceName, "maintenance_windows.0.window.1.days.2", "Sunday"),
					resource.TestCheckResourceAttr(dataSourceName, "pricing.#", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "pricing.0.type", "MinimumPrice"),
					resource.TestCheckResourceAttr(dataSourceName, "pricing.0.quantity", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "pricing.0.quantity_measurement", "subscription"),
					resource.TestCheckResourceAttrSet(dataSourceName, "pricing.0.price_per_unit"),
					resource.TestCheckResourceAttr(dataSourceName, "pricing.0.price_currency", "USD"),
					resource.TestCheckResourceAttr(dataSourceName, "pricing.0.price_period", "hour"),
					resource.TestCheckResourceAttr(dataSourceName, "pricing.1.type", "MinimumPrice"),
					resource.TestCheckResourceAttr(dataSourceName, "pricing.1.quantity", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "pricing.1.quantity_measurement", "subscription"),
					resource.TestCheckResourceAttrSet(dataSourceName, "pricing.1.price_per_unit"),
					resource.TestCheckResourceAttr(dataSourceName, "pricing.1.price_currency", "USD"),
					resource.TestCheckResourceAttr(dataSourceName, "pricing.1.price_period", "hour"),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					// ExpectKnownValue distinguishes the compatibility value "" from null.
					statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("customer_managed_key_deletion_grace_period"), knownvalue.StringExact("")),
					statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("customer_managed_key_redis_service_account"), knownvalue.StringExact("")),
					statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("customer_managed_key_aws_role_arn"), knownvalue.StringExact("")),
				},
			},
		},
	})
}
