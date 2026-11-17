package transitgateway_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func TestAccDataSourceRedisCloudActiveActiveTransitGatewayInvitations_basic(t *testing.T) {
	resourceTags, resourceTagsCheck := envchecks.BYOCResourceTagsValueAndCheck()
	awsRegion, awsRegionCheck := envchecks.ValueAndCheck("AWS_REGION")

	subscriptionName := utils.RandomWithPrefix() + "-aa-tgw-inv"
	databaseName := utils.RandomWithPrefix() + "-aa-tgw-inv-db"
	databasePassword := acctest.RandString(20)

	const datasourceName = "data.rediscloud_active_active_transit_gateway_invitations.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: envchecks.ComposePreChecks(t,
			resourceTagsCheck,
			envchecks.RedisCloudCheck,
			envchecks.AWSProviderCheck,
			awsRegionCheck,
		),
		CheckDestroy: testhelpers.CheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				ConfigFile:               config.StaticFile("testdata/aa_transit_gateway_invitations_datasource.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(subscriptionName),
					"database_name":     config.StringVariable(databaseName),
					"database_password": config.StringVariable(databasePassword),
					"aws_region":        config.StringVariable(awsRegion),
					"team":              config.StringVariable(resourceTags.Team),
					"owner":             config.StringVariable(resourceTags.Owner),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(datasourceName, "id", regexp.MustCompile(`^\d+/\d+$`)),
					resource.TestCheckResourceAttrPair(datasourceName, "subscription_id", "rediscloud_active_active_subscription.test", "id"),
					resource.TestCheckResourceAttrSet(datasourceName, "region_id"),
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
