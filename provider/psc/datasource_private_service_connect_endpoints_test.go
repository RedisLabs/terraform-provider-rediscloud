package psc_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func TestAccDataSourceRedisCloudPrivateServiceConnectEndpoints_basic(t *testing.T) {
	baseName := utils.RandomWithPrefix() + "-pro-psce"
	gcpProjectId, gcpProjectCheck := envchecks.GCPProjectValueAndCheck()
	gcpVPCName := baseName + "-network"
	gcpVPCSubnetName := baseName + "-subnet"

	const dataSourceName = "data.rediscloud_private_service_connect_endpoints.psce"
	const endpointResourceName = "rediscloud_private_service_connect_endpoint.psce"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, envchecks.RedisCloudCheck, gcpProjectCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/datasource_private_service_connect_endpoints.tf"),
				ConfigVariables: config.Variables{
					"subscription_name":   config.StringVariable(baseName),
					"gcp_project_id":      config.StringVariable(gcpProjectId),
					"gcp_vpc_name":        config.StringVariable(gcpVPCName),
					"gcp_vpc_subnet_name": config.StringVariable(gcpVPCSubnetName),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceName, "id"),
					resource.TestCheckResourceAttr(dataSourceName, "endpoints.#", "1"),
					resource.TestCheckResourceAttrPair(dataSourceName, "endpoints.0.private_service_connect_endpoint_id", endpointResourceName, "private_service_connect_endpoint_id"),
					resource.TestCheckResourceAttr(dataSourceName, "endpoints.0.gcp_project_id", gcpProjectId),
					resource.TestCheckResourceAttr(dataSourceName, "endpoints.0.gcp_vpc_name", gcpVPCName),
					resource.TestCheckResourceAttr(dataSourceName, "endpoints.0.gcp_vpc_subnet_name", gcpVPCSubnetName),
					resource.TestCheckResourceAttrWith(dataSourceName, "endpoints.0.endpoint_connection_name", func(value string) error {
						if !strings.HasPrefix(value, "redis-") {
							return fmt.Errorf("expected %s to have prefix 'redis-'", value)
						}
						return nil
					}),
					resource.TestCheckResourceAttrSet(dataSourceName, "endpoints.0.status"),
					resource.TestCheckResourceAttr(dataSourceName, "endpoints.0.service_attachments.#", "1"),
				),
			},
		},
	})
}
