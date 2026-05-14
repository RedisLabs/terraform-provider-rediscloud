package provider

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccResourceRedisCloudPrivateServiceConnectEndpoint_CRUDI(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	baseName := testRandomWithPrefix() + "-pro-psce"

	const resourceName = "rediscloud_private_service_connect_endpoint.psce"
	const datasourceName = "data.rediscloud_private_service_connect_endpoints.psce"
	gcpProjectId := os.Getenv("GCP_PROJECT_ID")
	gcpVPCName := fmt.Sprintf("%s-network", baseName)
	gcpVPCSubnetName := fmt.Sprintf("%s-subnet", baseName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); testAccGcpProjectPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./psc/testdata/pro_psce_step1.tf"),
				ConfigVariables: config.Variables{
					"subscription_name":   config.StringVariable(baseName),
					"gcp_project_id":      config.StringVariable(gcpProjectId),
					"gcp_vpc_name":        config.StringVariable(gcpVPCName),
					"gcp_vpc_subnet_name": config.StringVariable(gcpVPCSubnetName),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "private_service_connect_endpoint_id"),
				),
			},
			{
				ConfigFile: config.StaticFile("./psc/testdata/pro_psce_step2.tf"),
				ConfigVariables: config.Variables{
					"subscription_name":   config.StringVariable(baseName),
					"gcp_project_id":      config.StringVariable(gcpProjectId),
					"gcp_vpc_name":        config.StringVariable(gcpVPCName),
					"gcp_vpc_subnet_name": config.StringVariable(gcpVPCSubnetName),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(datasourceName, "id"),
					resource.TestCheckResourceAttr(datasourceName, "endpoints.#", "1"),
					resource.TestCheckResourceAttr(datasourceName, "endpoints.0.gcp_project_id", gcpProjectId),
					resource.TestCheckResourceAttr(datasourceName, "endpoints.0.gcp_vpc_name", gcpVPCName),
					resource.TestCheckResourceAttr(datasourceName, "endpoints.0.gcp_vpc_subnet_name", gcpVPCSubnetName),
					resource.TestCheckResourceAttrWith(datasourceName, "endpoints.0.endpoint_connection_name", func(value string) error {
						if !strings.HasPrefix(value, "redis-") {
							return fmt.Errorf("expected %s to have prefix 'redis-'", value)
						}
						return nil
					}),
					resource.TestCheckResourceAttr(datasourceName, "endpoints.0.service_attachments.#", "1"),
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
