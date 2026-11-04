package privatelink_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func TestAccDataSourceRedisCloudPrivateLink_basic(t *testing.T) {
	resourceTags, resourceTagsCheck := envchecks.BYOCResourceTagsValueAndCheck()
	cloudAccountName, cloudAccountCheck := envchecks.AWSBYOCValueAndCheck()

	subscriptionName := utils.RandomWithPrefix() + "-pro-private-link"
	shareName := utils.RandomWithPrefix() + "-privatelink"

	const dataSourceName = "data.rediscloud_private_link.pro_private_link"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, resourceTagsCheck, envchecks.RedisCloudCheck, cloudAccountCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/datasource_private_link.tf"),
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
					resource.TestCheckResourceAttr(dataSourceName, "share_name", shareName),
					resource.TestCheckResourceAttr(dataSourceName, "principals.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs(dataSourceName, "principals.*", map[string]string{
						"principal":       "123456789012",
						"principal_type":  "aws_account",
						"principal_alias": "principal 1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(dataSourceName, "principals.*", map[string]string{
						"principal":       "234567890123",
						"principal_type":  "aws_account",
						"principal_alias": "principal 2",
					}),
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
