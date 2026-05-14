package provider

import (
	"testing"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccResourceRedisCloudPrivateServiceConnect_CRUDI(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	baseName := testRandomWithPrefix() + "-pro-psc"

	const resourceName = "rediscloud_private_service_connect.psc"
	const datasourceName = "data.rediscloud_private_service_connect.psc"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./psc/testdata/pro_psc_step1.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(baseName),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "subscription_id"),
					resource.TestCheckResourceAttrSet(resourceName, "private_service_connect_service_id"),
				),
			},
			{
				ConfigFile: config.StaticFile("./psc/testdata/pro_psc_step2.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(baseName),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(datasourceName, "id"),
					resource.TestCheckResourceAttrSet(datasourceName, "subscription_id"),
					resource.TestCheckResourceAttrSet(datasourceName, "private_service_connect_service_id"),
					resource.TestCheckResourceAttrSet(datasourceName, "connection_host_name"),
					resource.TestCheckResourceAttrSet(datasourceName, "service_attachment_name"),
					resource.TestCheckResourceAttr(datasourceName, "status", "active"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
