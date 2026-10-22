package essentialssubscription_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	fs "github.com/RedisLabs/rediscloud-go-api/service/fixed/subscriptions"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

const (
	resourceName         = "rediscloud_essentials_subscription.example"
	datasourceName       = "data.rediscloud_essentials_subscription.example"
	byNameDatasourceName = "data.rediscloud_essentials_subscription.by_name"
	byIDDatasourceName   = "data.rediscloud_essentials_subscription.by_id"

	missingSubscriptionID = 999999999
)

// These tests use resource.Test rather than ParallelTest because Redis Cloud allows only one essentials subscription per account.
// Each test's first step creates the subscription alone: the data source's name is known at plan time, so it would otherwise be read before the resource exists.
func TestAccDataSourceRedisCloudEssentialsSubscription_Free(t *testing.T) {
	name := utils.RandomWithPrefix()

	resource.Test(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, testhelpers.PreCheckNoEssentialsSubscription),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckEssentialsSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/free/subscription.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(name),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "payment_method_id", "0"),
				),
			},
			{
				ConfigDirectory: config.StaticDirectory("testdata/free"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(name),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkFreeDataSource(byNameDatasourceName, name),
					checkFreeDataSource(byIDDatasourceName, name),
				),
			},
			{
				ConfigDirectory: config.StaticDirectory("testdata/free"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(name),
					"lookup_name":       config.StringVariable(name + "-missing"),
				},
				ExpectError: regexp.MustCompile("No Essentials Subscriptions Found"),
			},
			{
				ConfigDirectory: config.StaticDirectory("testdata/free"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(name),
					"lookup_id":         config.IntegerVariable(missingSubscriptionID),
				},
				ExpectError: regexp.MustCompile("No Essentials Subscriptions Found"),
			},
		},
	})
}

func checkFreeDataSource(datasourceName, name string) resource.TestCheckFunc {
	return resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttrPair(datasourceName, "id", resourceName, "id"),
		resource.TestCheckResourceAttr(datasourceName, "name", name),
		resource.TestCheckResourceAttr(datasourceName, "status", fs.FixedSubscriptionStatusActive),
		resource.TestCheckResourceAttrSet(datasourceName, "plan_id"),
		resource.TestCheckResourceAttr(datasourceName, "payment_method_id", "0"),
		resource.TestCheckResourceAttrSet(datasourceName, "creation_date"),
	)
}

func TestAccDataSourceRedisCloudEssentialsSubscription_Paid_CreditCard(t *testing.T) {
	name := utils.RandomWithPrefix()

	resource.Test(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, testhelpers.PreCheckNoEssentialsSubscription),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckEssentialsSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/paid_by_name/subscription.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(name),
				},
				Check: resource.TestCheckResourceAttr(resourceName, "name", name),
			},
			{
				ConfigDirectory: config.StaticDirectory("testdata/paid_by_name"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(name),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(datasourceName, "id"),
					resource.TestCheckResourceAttr(datasourceName, "name", name),
					resource.TestCheckResourceAttr(datasourceName, "status", fs.FixedSubscriptionStatusActive),
					resource.TestCheckResourceAttrSet(datasourceName, "plan_id"),
					resource.TestCheckResourceAttrSet(datasourceName, "payment_method_id"),
					resource.TestCheckResourceAttrSet(datasourceName, "creation_date"),
					resource.TestCheckResourceAttrPair(datasourceName, "id", resourceName, "id"),
				),
			},
		},
	})
}

func TestAccDataSourceRedisCloudEssentialsSubscription_Paid_NoPaymentType(t *testing.T) {
	name := utils.RandomWithPrefix()

	resource.Test(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, testhelpers.PreCheckNoEssentialsSubscription),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckEssentialsSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/paid_no_payment_type_by_name/subscription.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(name),
				},
				Check: resource.TestCheckResourceAttr(resourceName, "name", name),
			},
			{
				ConfigDirectory: config.StaticDirectory("testdata/paid_no_payment_type_by_name"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(name),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(datasourceName, "id"),
					resource.TestCheckResourceAttr(datasourceName, "name", name),
					resource.TestCheckResourceAttr(datasourceName, "status", fs.FixedSubscriptionStatusActive),
					resource.TestCheckResourceAttrSet(datasourceName, "plan_id"),
					resource.TestCheckResourceAttrSet(datasourceName, "payment_method_id"),
					resource.TestCheckResourceAttrSet(datasourceName, "creation_date"),
					resource.TestCheckResourceAttrPair(datasourceName, "id", resourceName, "id"),
				),
			},
		},
	})
}
