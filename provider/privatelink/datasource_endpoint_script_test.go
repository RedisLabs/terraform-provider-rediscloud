package privatelink_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func TestAccDataSourceRedisCloudPrivateLinkEndpointScript_basic(t *testing.T) {
	resourceTags, resourceTagsCheck := envchecks.BYOCResourceTagsValueAndCheck()
	cloudAccountName, cloudAccountCheck := envchecks.AWSBYOCValueAndCheck()

	subscriptionName := utils.RandomWithPrefix() + "-pl-script"
	shareName := utils.RandomWithPrefix() + "-privatelink"

	const dataSourceName = "data.rediscloud_private_link_endpoint_script.endpoint_script"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, resourceTagsCheck, envchecks.RedisCloudCheck, cloudAccountCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/datasource_endpoint_script.tf"),
				ConfigVariables: config.Variables{
					"subscription_name":  config.StringVariable(subscriptionName),
					"cloud_account_name": config.StringVariable(cloudAccountName),
					"share_name":         config.StringVariable(shareName),
					"team":               config.StringVariable(resourceTags.Team),
					"owner":              config.StringVariable(resourceTags.Owner),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceName, "id"),
					resource.TestCheckResourceAttrPair(dataSourceName, "subscription_id", "rediscloud_subscription.pro_subscription", "id"),
					resource.TestCheckResourceAttrSet(dataSourceName, "endpoint_script"),
				),
			},
		},
	})
}
