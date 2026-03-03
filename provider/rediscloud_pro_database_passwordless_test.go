package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func TestAccRedisCloudProDatabase_Passwordless(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	const databaseResource = "rediscloud_subscription_database.example"
	const datasourceName = "data.rediscloud_database.example"
	subscriptionName := testRandomWithPrefix()

	content := utils.GetTestConfig(t, "./pro/testdata/pro_database_passwordless.tf")
	config := fmt.Sprintf(content, subscriptionName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
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
