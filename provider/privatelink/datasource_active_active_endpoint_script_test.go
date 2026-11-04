package privatelink_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func TestAccDataSourceRedisCloudActiveActivePrivateLinkEndpointScript_basic(t *testing.T) {
	resourceTags, resourceTagsCheck := envchecks.BYOCResourceTagsValueAndCheck()

	subscriptionName := utils.RandomWithPrefix() + "-aa-pl-script"
	shareName := utils.RandomWithPrefix() + "-privatelink-aa"

	const dataSourceName = "data.rediscloud_active_active_private_link_endpoint_script.endpoint_script"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, resourceTagsCheck, envchecks.RedisCloudCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/datasource_active_active_endpoint_script.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(subscriptionName),
					"share_name":        config.StringVariable(shareName),
					"team":              config.StringVariable(resourceTags.Team),
					"owner":             config.StringVariable(resourceTags.Owner),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceName, "id"),
					resource.TestCheckResourceAttrPair(dataSourceName, "subscription_id", "rediscloud_active_active_subscription.aa_subscription", "id"),
					resource.TestCheckResourceAttrPair(dataSourceName, "region_id", "rediscloud_active_active_private_link.aa_private_link", "region_id"),
					resource.TestCheckResourceAttrSet(dataSourceName, "endpoint_script"),
				),
			},
		},
	})
}
