package psc_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

const (
	testActiveActivePrivateServiceConnectEndpointConfigFile               = "testdata/resource_active_active_private_service_connect_endpoint.tf"
	testActiveActivePrivateServiceConnectEndpointWithDataSourceConfigFile = "testdata/resource_active_active_private_service_connect_endpoint_with_datasource.tf"
	activeActivePrivateServiceConnectEndpointResourceName                 = "rediscloud_active_active_private_service_connect_endpoint.psce"
	activeActivePrivateServiceConnectEndpointsDataSourceName              = "data.rediscloud_active_active_private_service_connect_endpoints.psce"
)

func activeActivePrivateServiceConnectEndpointVariables(gcpProjectId string) config.Variables {
	baseName := utils.RandomWithPrefix() + "-aa-psce"
	return config.Variables{
		"subscription_name":   config.StringVariable(baseName),
		"database_password":   config.StringVariable(acctest.RandString(20)),
		"gcp_project_id":      config.StringVariable(gcpProjectId),
		"gcp_vpc_name":        config.StringVariable(baseName + "-network"),
		"gcp_vpc_subnet_name": config.StringVariable(baseName + "-subnet"),
	}
}

func TestAccResourceRedisCloudActiveActivePrivateServiceConnectEndpoint_CRUDI(t *testing.T) {
	gcpProjectId, gcpProjectCheck := envchecks.GCPProjectValueAndCheck()
	variables := activeActivePrivateServiceConnectEndpointVariables(gcpProjectId)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, envchecks.RedisCloudCheck, gcpProjectCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile:      config.StaticFile(testActiveActivePrivateServiceConnectEndpointConfigFile),
				ConfigVariables: variables,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(activeActivePrivateServiceConnectEndpointResourceName, "id"),
					resource.TestCheckResourceAttrSet(activeActivePrivateServiceConnectEndpointResourceName, "region_id"),
					resource.TestCheckResourceAttrSet(activeActivePrivateServiceConnectEndpointResourceName, "private_service_connect_endpoint_id"),
					resource.TestCheckResourceAttr(activeActivePrivateServiceConnectEndpointResourceName, "gcp_project_id", gcpProjectId),
					resource.TestCheckResourceAttr(activeActivePrivateServiceConnectEndpointResourceName, "service_attachments.#", "1"),
				),
			},
			{
				ConfigFile:      config.StaticFile(testActiveActivePrivateServiceConnectEndpointWithDataSourceConfigFile),
				ConfigVariables: variables,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(activeActivePrivateServiceConnectEndpointsDataSourceName, "id"),
					resource.TestCheckResourceAttr(activeActivePrivateServiceConnectEndpointsDataSourceName, "endpoints.#", "1"),
					resource.TestCheckResourceAttr(activeActivePrivateServiceConnectEndpointsDataSourceName, "endpoints.0.gcp_project_id", gcpProjectId),
					resource.TestCheckResourceAttrPair(activeActivePrivateServiceConnectEndpointsDataSourceName, "endpoints.0.gcp_vpc_name", activeActivePrivateServiceConnectEndpointResourceName, "gcp_vpc_name"),
					resource.TestCheckResourceAttrPair(activeActivePrivateServiceConnectEndpointsDataSourceName, "endpoints.0.gcp_vpc_subnet_name", activeActivePrivateServiceConnectEndpointResourceName, "gcp_vpc_subnet_name"),
					resource.TestCheckResourceAttrWith(activeActivePrivateServiceConnectEndpointsDataSourceName, "endpoints.0.endpoint_connection_name", func(value string) error {
						if !strings.HasPrefix(value, "redis-") {
							return fmt.Errorf("expected %s to have prefix 'redis-'", value)
						}
						return nil
					}),
					resource.TestCheckResourceAttr(activeActivePrivateServiceConnectEndpointsDataSourceName, "endpoints.0.service_attachments.#", "1"),
				),
			},
			{
				ConfigFile:        config.StaticFile(testActiveActivePrivateServiceConnectEndpointWithDataSourceConfigFile),
				ConfigVariables:   variables,
				ResourceName:      activeActivePrivateServiceConnectEndpointResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccResourceRedisCloudActiveActivePrivateServiceConnectEndpoint_MigrateFromSDK checks that
// state written by the last SDKv2 release plans cleanly with the Plugin Framework implementation.
func TestAccResourceRedisCloudActiveActivePrivateServiceConnectEndpoint_MigrateFromSDK(t *testing.T) {
	gcpProjectId, gcpProjectCheck := envchecks.GCPProjectValueAndCheck()
	variables := activeActivePrivateServiceConnectEndpointVariables(gcpProjectId)

	// ExternalProviders cannot be combined with ConfigFile, so both steps use the file content.
	terraformConfig := utils.GetTestConfig(t, testActiveActivePrivateServiceConnectEndpointConfigFile)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     envchecks.ComposePreChecks(t, envchecks.RedisCloudCheck, gcpProjectCheck),
		CheckDestroy: testhelpers.CheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"rediscloud": {
						Source:            "RedisLabs/rediscloud",
						VersionConstraint: "2.20.0",
					},
				},
				Config:          terraformConfig,
				ConfigVariables: variables,
				Check:           resource.TestCheckResourceAttrSet(activeActivePrivateServiceConnectEndpointResourceName, "private_service_connect_endpoint_id"),
			},
			{
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				Config:                   terraformConfig,
				ConfigVariables:          variables,
				PlanOnly:                 true,
			},
		},
	})
}
