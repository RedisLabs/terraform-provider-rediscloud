package privatelink_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func TestAccDataSourceRedisCloudActiveActivePrivateLink_basic(t *testing.T) {
	resourceTags, resourceTagsCheck := envchecks.BYOCResourceTagsValueAndCheck()

	subscriptionName := utils.RandomWithPrefix() + "-aa-private-link"
	shareName := utils.RandomWithPrefix() + "-privatelink-aa"

	const dataSourceName = "data.rediscloud_active_active_private_link.aa_private_link"
	const resourceName = "rediscloud_active_active_private_link.aa_private_link"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, resourceTagsCheck, envchecks.RedisCloudCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/datasource_active_active_private_link.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(subscriptionName),
					"share_name":        config.StringVariable(shareName),
					"team":              config.StringVariable(resourceTags.Team),
					"owner":             config.StringVariable(resourceTags.Owner),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(dataSourceName, "id", resourceName, "id"),
					resource.TestCheckResourceAttrPair(dataSourceName, "subscription_id", "rediscloud_active_active_subscription.aa_subscription", "id"),
					resource.TestCheckResourceAttrPair(dataSourceName, "region_id", resourceName, "region_id"),
					resource.TestCheckResourceAttr(dataSourceName, "share_name", shareName),
					resource.TestCheckResourceAttrSet(dataSourceName, "resource_configuration_id"),
					resource.TestCheckResourceAttrSet(dataSourceName, "resource_configuration_arn"),
					resource.TestCheckResourceAttrSet(dataSourceName, "share_arn"),
					resource.TestCheckResourceAttrSet(dataSourceName, "connections.#"),
					resource.TestCheckResourceAttrSet(dataSourceName, "databases.#"),
				),
			},
		},
	})
}
