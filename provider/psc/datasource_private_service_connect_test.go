package psc_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func TestAccDataSourceRedisCloudPrivateServiceConnect_basic(t *testing.T) {
	subscriptionName := utils.RandomWithPrefix() + "-pro-psc"

	const dataSourceName = "data.rediscloud_private_service_connect.psc"
	const resourceName = "rediscloud_private_service_connect.psc"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, envchecks.RedisCloudCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/datasource_private_service_connect.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(subscriptionName),
					"database_name":     config.StringVariable(subscriptionName + "-database"),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(dataSourceName, "id", resourceName, "id"),
					resource.TestCheckResourceAttrPair(dataSourceName, "subscription_id", "rediscloud_subscription.subscription_resource", "id"),
					resource.TestCheckResourceAttrPair(dataSourceName, "private_service_connect_service_id", resourceName, "private_service_connect_service_id"),
					resource.TestCheckResourceAttrSet(dataSourceName, "connection_host_name"),
					resource.TestCheckResourceAttrSet(dataSourceName, "service_attachment_name"),
					resource.TestCheckResourceAttr(dataSourceName, "status", "active"),
				),
			},
		},
	})
}
