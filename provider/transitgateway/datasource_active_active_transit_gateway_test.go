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

func TestAccDataSourceRedisCloudActiveActiveTransitGateway_basic(t *testing.T) {
	resourceTags, resourceTagsCheck := envchecks.BYOCResourceTagsValueAndCheck()
	awsRegion, awsRegionCheck := envchecks.ValueAndCheck("AWS_REGION")

	subscriptionName := utils.RandomWithPrefix() + "-aa-tgw-ds"
	databaseName := utils.RandomWithPrefix() + "-aa-tgw-ds-db"
	databasePassword := acctest.RandString(20)

	const datasourceName = "data.rediscloud_active_active_transit_gateway.test"

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
				ConfigFile:               config.StaticFile("testdata/aa_transit_gateway_datasource.tf"),
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
					resource.TestMatchResourceAttr(datasourceName, "tgw_id", regexp.MustCompile(`^[1-9]\d*$`)),
					resource.TestCheckResourceAttrPair(datasourceName, "aws_tgw_uid", "aws_ec2_transit_gateway.test", "id"),
					resource.TestCheckResourceAttrSet(datasourceName, "region_id"),
					resource.TestCheckResourceAttr(datasourceName, "wait_for_tgw_timeout", "600"),
					resource.TestCheckResourceAttrSet(datasourceName, "status"),
					resource.TestCheckResourceAttrSet(datasourceName, "aws_account_id"),
					resource.TestCheckResourceAttr(datasourceName, "cidrs.#", "0"),
				),
			},
			{
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				ConfigFile:               config.StaticFile("testdata/aa_transit_gateway_datasource.tf"),
				ConfigVariables: config.Variables{
					"subscription_name":    config.StringVariable(subscriptionName),
					"database_name":        config.StringVariable(databaseName),
					"database_password":    config.StringVariable(databasePassword),
					"aws_region":           config.StringVariable(awsRegion),
					"team":                 config.StringVariable(resourceTags.Team),
					"owner":                config.StringVariable(resourceTags.Owner),
					"lookup_aws_tgw_uid":   config.StringVariable("tgw-00000000000000000"),
					"wait_for_tgw_timeout": config.IntegerVariable(0),
				},
				ExpectError: regexp.MustCompile("Your query returned no results"),
			},
		},
	})
}
