package transitgateway_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func TestAccDataSourceRedisCloudTransitGatewayInvitations_basic(t *testing.T) {
	resourceTags, resourceTagsCheck := envchecks.BYOCResourceTagsValueAndCheck()
	awsRegion, awsRegionCheck := envchecks.ValueAndCheck("AWS_REGION")

	subscriptionName := utils.RandomWithPrefix() + "-pro-tgw-inv"

	const datasourceName = "data.rediscloud_transit_gateway_invitations.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: envchecks.ComposePreChecks(t,
			resourceTagsCheck,
			envchecks.RedisCloudCheck,
			envchecks.AWSProviderCheck,
			awsRegionCheck,
		),
		CheckDestroy: testhelpers.CheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				ConfigFile:               config.StaticFile("testdata/pro_transit_gateway_invitations_datasource.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(subscriptionName),
					"aws_region":        config.StringVariable(awsRegion),
					"team":              config.StringVariable(resourceTags.Team),
					"owner":             config.StringVariable(resourceTags.Owner),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(datasourceName, "id", regexp.MustCompile(`^\d+$`)),
					resource.TestCheckResourceAttrPair(datasourceName, "subscription_id", "rediscloud_subscription.example", "id"),
					resource.TestCheckResourceAttr(datasourceName, "wait_for_invitations_timeout", "600"),
					resource.TestCheckResourceAttrSet(datasourceName, "invitations.#"),
					resource.TestCheckTypeSetElemNestedAttrs(datasourceName, "invitations.*", map[string]string{
						"name": subscriptionName,
					}),
					resource.TestCheckResourceAttrSet(datasourceName, "invitations.0.id"),
					resource.TestCheckResourceAttrSet(datasourceName, "invitations.0.resource_share_uid"),
					resource.TestCheckResourceAttrSet(datasourceName, "invitations.0.aws_account_id"),
					resource.TestCheckResourceAttrSet(datasourceName, "invitations.0.shared_date"),
				),
			},
		},
	})
}
