package essentialssubscription_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	fs "github.com/RedisLabs/rediscloud-go-api/service/fixed/subscriptions"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

const (
	resourceName   = "rediscloud_essentials_subscription.example"
	datasourceName = "data.rediscloud_essentials_subscription.example"
)

// These tests use resource.Test rather than ParallelTest because Redis Cloud allows only one essentials subscription per account.
// Each test's first step creates the subscription alone: the data source's name is known at plan time, so it would otherwise be read before the resource exists.
func TestAccDataSourceRedisCloudEssentialsSubscription_Free(t *testing.T) {
	name := utils.RandomWithPrefix()

	resource.Test(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, preCheckNoEssentialsSubscription),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             checkEssentialsSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/free_by_name/subscription.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(name),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "payment_method_id", "0"),
				),
			},
			{
				ConfigDirectory: config.StaticDirectory("testdata/free_by_name"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(name),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(datasourceName, "id"),
					resource.TestCheckResourceAttr(datasourceName, "name", name),
					resource.TestCheckResourceAttr(datasourceName, "status", fs.FixedSubscriptionStatusActive),
					resource.TestCheckResourceAttrSet(datasourceName, "plan_id"),
					resource.TestCheckResourceAttr(datasourceName, "payment_method_id", "0"),
					resource.TestCheckResourceAttrSet(datasourceName, "creation_date"),
					resource.TestCheckResourceAttrPair(datasourceName, "id", resourceName, "id"),
				),
			},
			{
				ConfigDirectory: config.StaticDirectory("testdata/free_by_id"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(name),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(datasourceName, "id", resourceName, "id"),
					resource.TestCheckResourceAttr(datasourceName, "name", name),
					resource.TestCheckResourceAttr(datasourceName, "status", fs.FixedSubscriptionStatusActive),
					resource.TestCheckResourceAttrSet(datasourceName, "plan_id"),
					resource.TestCheckResourceAttr(datasourceName, "payment_method_id", "0"),
					resource.TestCheckResourceAttrSet(datasourceName, "creation_date"),
				),
			},
		},
	})
}

func TestAccDataSourceRedisCloudEssentialsSubscription_Paid_CreditCard(t *testing.T) {
	name := utils.RandomWithPrefix()

	resource.Test(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, preCheckNoEssentialsSubscription),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             checkEssentialsSubscriptionDestroy,
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
		PreCheck:                 envchecks.ComposePreChecks(t, preCheckNoEssentialsSubscription),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             checkEssentialsSubscriptionDestroy,
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

// preCheckNoEssentialsSubscription fails fast because Redis Cloud allows only one
// essentials subscription per account.
func preCheckNoEssentialsSubscription(t *testing.T) bool {
	t.Helper()
	if !envchecks.RedisCloudCheck(t) {
		return false
	}

	apiClient, err := client.GetTestClient()
	if err != nil {
		t.Errorf("Failed to create API client: %v", err)
		return false
	}

	subs, err := apiClient.Client.FixedSubscriptions.List(context.TODO())
	if err != nil {
		t.Errorf("Failed to list essentials subscriptions: %v", err)
		return false
	}

	if len(subs) > 0 {
		t.Errorf("Essentials subscription already exists (ID: %d). Redis Cloud allows only 1 essentials subscription per account. Please delete the existing subscription before running this test.", redis.IntValue(subs[0].ID))
		return false
	}
	return true
}

func checkEssentialsSubscriptionDestroy(s *terraform.State) error {
	apiClient, err := client.GetTestClient()
	if err != nil {
		return err
	}

	for _, r := range s.RootModule().Resources {
		if r.Type != "rediscloud_essentials_subscription" {
			continue
		}

		subId, err := strconv.Atoi(r.Primary.ID)
		if err != nil {
			return err
		}

		subs, err := apiClient.Client.FixedSubscriptions.List(context.TODO())
		if err != nil {
			return err
		}

		for _, sub := range subs {
			if redis.IntValue(sub.ID) == subId {
				return fmt.Errorf("fixed subscription %d still exists", subId)
			}
		}
	}

	return nil
}
