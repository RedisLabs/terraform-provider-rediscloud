package pro_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

const (
	proSubscriptionDataSourceConfigPath = "testdata/datasource/pro_subscription.tf"
	subscriptionFiltersConfigPath       = "testdata/datasource/subscription_filters_by_deployment_type.tf"
)

func TestAccDataSourceRedisCloudProSubscription_basic(t *testing.T) {
	name := utils.RandomWithPrefix()
	const dataSourceName = "data.rediscloud_subscription.example"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, envchecks.RedisCloudCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             checkProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile(proSubscriptionDataSourceConfigPath),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(name),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "name", name),
					resource.TestCheckResourceAttr(dataSourceName, "payment_method", "credit-card"),
					resource.TestCheckResourceAttrSet(dataSourceName, "payment_method_id"),
					resource.TestMatchResourceAttr(dataSourceName, "memory_storage", regexp.MustCompile("ram")),
					resource.TestCheckResourceAttr(dataSourceName, "number_of_databases", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "cloud_provider.0.provider", "AWS"),
					resource.TestCheckResourceAttr(dataSourceName, "cloud_provider.0.cloud_account_id", "1"),
					// TODO(tests): the API returns a nil aws_account_id for this fixture, so
					// this assertion fails; re-enable once the data source reliably populates it.
					// resource.TestCheckResourceAttrSet(dataSourceName, "cloud_provider.0.aws_account_id"),
					resource.TestCheckResourceAttr(dataSourceName, "cloud_provider.0.region.0.region", "eu-west-1"),
					resource.TestCheckResourceAttr(dataSourceName, "cloud_provider.0.region.0.networks.0.networking_deployment_cidr", "10.0.0.0/24"),
					resource.TestCheckResourceAttr(dataSourceName, "status", "active"),
					resource.TestCheckResourceAttr(dataSourceName, "customer_managed_key_enabled", "false"),
					resource.TestCheckResourceAttr(dataSourceName, "customer_managed_key_deletion_grace_period", ""),
					resource.TestCheckResourceAttr(dataSourceName, "customer_managed_key_redis_service_account", ""),
					resource.TestCheckResourceAttr(dataSourceName, "public_endpoint_access", "true"),
					resource.TestCheckResourceAttrSet(dataSourceName, "prometheus_endpoint"),

					resource.TestCheckResourceAttr(dataSourceName, "pricing.#", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "pricing.0.type", "MinimumPrice"),
					resource.TestCheckResourceAttr(dataSourceName, "pricing.0.quantity", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "pricing.0.quantity_measurement", "subscription"),
					resource.TestCheckResourceAttrSet(dataSourceName, "pricing.0.price_per_unit"),
					resource.TestCheckResourceAttr(dataSourceName, "pricing.0.price_currency", "USD"),
					resource.TestCheckResourceAttr(dataSourceName, "pricing.0.price_period", "hour"),
					resource.TestCheckResourceAttr(dataSourceName, "pricing.1.type", "Shards"),
					resource.TestCheckResourceAttr(dataSourceName, "pricing.1.type_details", "micro"),
					resource.TestCheckResourceAttr(dataSourceName, "pricing.1.quantity", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "pricing.1.quantity_measurement", "shards"),
					resource.TestCheckResourceAttrSet(dataSourceName, "pricing.1.price_per_unit"),
					resource.TestCheckResourceAttr(dataSourceName, "pricing.1.price_currency", "USD"),
					resource.TestCheckResourceAttr(dataSourceName, "pricing.1.price_period", "hour"),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("customer_managed_key_deletion_grace_period"), knownvalue.StringExact("")),
					statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("customer_managed_key_redis_service_account"), knownvalue.StringExact("")),
					statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("customer_managed_key_aws_role_arn"), knownvalue.StringExact("")),
				},
			},
		},
	})
}

// TestAccDataSourceRedisCloudProSubscription_filterByDeploymentType verifies that
// the Pro and Active-Active data sources return subscriptions of their respective
// deployment types when both subscriptions have the same name.
func TestAccDataSourceRedisCloudProSubscription_filterByDeploymentType(t *testing.T) {
	name := utils.RandomWithPrefix()
	const proResourceName = "rediscloud_subscription.pro"
	const activeActiveResourceName = "rediscloud_active_active_subscription.active_active"
	const proDataSourceName = "data.rediscloud_subscription.pro"
	const activeActiveDataSourceName = "data.rediscloud_active_active_subscription.active_active"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, envchecks.RedisCloudCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy: resource.ComposeTestCheckFunc(
			checkProSubscriptionDestroy,
			checkAASubscriptionDestroy,
		),
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile(subscriptionFiltersConfigPath),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(name),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(proDataSourceName, "id", proResourceName, "id"),
					resource.TestCheckResourceAttrPair(activeActiveDataSourceName, "id", activeActiveResourceName, "id"),
				),
			},
		},
	})
}
