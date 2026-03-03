package pro_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func TestAccRedisCloudProDatabase_Passwordless(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	const databaseResource = "rediscloud_subscription_database.example"
	const datasourceName = "data.rediscloud_database.example"
	subscriptionName := utils.RandomWithPrefix()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             checkProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/pro_database_passwordless.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(subscriptionName),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					// Subscription checks
					resource.TestCheckResourceAttr("rediscloud_subscription.example", "public_endpoint_access", "false"),

					// Database resource checks
					resource.TestCheckResourceAttr(databaseResource, "name", subscriptionName),
					resource.TestCheckResourceAttr(databaseResource, "enable_passwordless", "true"),
					resource.TestCheckResourceAttr(databaseResource, "password", ""),

					// Data source checks
					resource.TestCheckResourceAttr(datasourceName, "enable_passwordless", "true"),
				),
			},
		},
	})
}
