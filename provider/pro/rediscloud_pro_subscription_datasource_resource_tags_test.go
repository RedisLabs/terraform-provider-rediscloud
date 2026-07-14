package pro_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

// TestAccDataSourceRedisCloudProSubscription_ResourceTags exercises the data
// source's cloud_provider.0.resource_tags attribute. It requires a BYOC cloud
// account because the API only accepts tags for BYOC subscriptions.
func TestAccDataSourceRedisCloudProSubscription_ResourceTags(t *testing.T) {
	cloudAccountName, cloudAccountCheck := envchecks.AWSBYOCValueAndCheck()

	name := acctest.RandomWithPrefix("tf-test") + "-ds-resource-tags"
	const dataSourceName = "data.rediscloud_subscription.example"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, envchecks.RedisCloudCheck, cloudAccountCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             checkProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/pro_subscription_datasource_resource_tags.tf"),
				ConfigVariables: config.Variables{
					"cloud_account_name": config.StringVariable(cloudAccountName),
					"subscription_name":  config.StringVariable(name),
					"resource_tags": config.MapVariable(map[string]config.Variable{
						"environment": config.StringVariable("staging"),
						"team":        config.StringVariable("platform"),
					}),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "cloud_provider.0.resource_tags.%", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "cloud_provider.0.resource_tags.environment", "staging"),
					resource.TestCheckResourceAttr(dataSourceName, "cloud_provider.0.resource_tags.team", "platform"),
				),
			},
		},
	})
}
