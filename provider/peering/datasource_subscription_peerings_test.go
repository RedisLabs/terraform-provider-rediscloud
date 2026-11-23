package peering_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func TestAccDataSourceRedisCloudSubscriptionPeerings_basic(t *testing.T) {
	resourceTags, resourceTagsCheck := envchecks.BYOCResourceTagsValueAndCheck()
	cloudAccountName, cloudAccountCheck := envchecks.AWSBYOCValueAndCheck()
	awsPeering, awsPeeringCheck := envchecks.AwsPeeringValueAndCheck()
	name := utils.RandomWithPrefix()

	const dataSourceName = "data.rediscloud_subscription_peerings.example"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, envchecks.RedisCloudCheck, awsPeeringCheck, cloudAccountCheck, resourceTagsCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/datasource_subscription_peerings.tf"),
				ConfigVariables: config.Variables{
					"team":               config.StringVariable(resourceTags.Team),
					"owner":              config.StringVariable(resourceTags.Owner),
					"cloud_account_name": config.StringVariable(cloudAccountName),
					"subscription_name":  config.StringVariable(name),
					"subscription_cidr":  config.StringVariable("10.0.0.0/24"),
					"aws_region":         config.StringVariable(awsPeering.Region),
					"aws_account_id":     config.StringVariable(awsPeering.AccountId),
					"vpc_id":             config.StringVariable(awsPeering.VpcId),
					"vpc_cidr":           config.StringVariable(awsPeering.VpcCidr),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "id", "ALL"),
					resource.TestMatchTypeSetElemNestedAttrs(dataSourceName, "peerings.*", map[string]*regexp.Regexp{
						"peering_id":     regexp.MustCompile(`^\d+$`),
						"provider_name":  regexp.MustCompile("AWS"),
						"aws_account_id": regexp.MustCompile(regexp.QuoteMeta(awsPeering.AccountId)),
						"vpc_id":         regexp.MustCompile(regexp.QuoteMeta(awsPeering.VpcId)),
						"vpc_cidr":       regexp.MustCompile(regexp.QuoteMeta(awsPeering.VpcCidr)),
						"aws_peering_id": regexp.MustCompile("^pcx-"),
						"region":         regexp.MustCompile(regexp.QuoteMeta(awsPeering.Region)),
					}),
				),
			},
		},
	})
}
