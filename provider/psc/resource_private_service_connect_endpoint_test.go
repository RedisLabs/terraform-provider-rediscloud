package psc_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

const (
	testPrivateServiceConnectEndpointConfigFile = "testdata/resource_private_service_connect_endpoint.tf"
	privateServiceConnectEndpointResourceName   = "rediscloud_private_service_connect_endpoint.psce"
)

func privateServiceConnectEndpointVariables(gcpProjectId string) config.Variables {
	baseName := utils.RandomWithPrefix() + "-pro-psce"
	return config.Variables{
		"subscription_name":   config.StringVariable(baseName),
		"gcp_project_id":      config.StringVariable(gcpProjectId),
		"gcp_vpc_name":        config.StringVariable(baseName + "-network"),
		"gcp_vpc_subnet_name": config.StringVariable(baseName + "-subnet"),
	}
}

func TestAccResourceRedisCloudPrivateServiceConnectEndpoint_CRUDI(t *testing.T) {
	gcpProjectId, gcpProjectCheck := envchecks.GCPProjectValueAndCheck()
	variables := privateServiceConnectEndpointVariables(gcpProjectId)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, envchecks.RedisCloudCheck, gcpProjectCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile:      config.StaticFile(testPrivateServiceConnectEndpointConfigFile),
				ConfigVariables: variables,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(privateServiceConnectEndpointResourceName, "id"),
					resource.TestCheckResourceAttrPair(privateServiceConnectEndpointResourceName, "subscription_id", "rediscloud_subscription.subscription_resource", "id"),
					resource.TestCheckResourceAttrPair(privateServiceConnectEndpointResourceName, "private_service_connect_service_id", "rediscloud_private_service_connect.psc", "private_service_connect_service_id"),
					resource.TestCheckResourceAttrSet(privateServiceConnectEndpointResourceName, "private_service_connect_endpoint_id"),
					resource.TestCheckResourceAttr(privateServiceConnectEndpointResourceName, "gcp_project_id", gcpProjectId),
					resource.TestCheckResourceAttr(privateServiceConnectEndpointResourceName, "service_attachments.#", "1"),
				),
			},
			{
				ConfigFile:        config.StaticFile(testPrivateServiceConnectEndpointConfigFile),
				ConfigVariables:   variables,
				ResourceName:      privateServiceConnectEndpointResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccResourceRedisCloudPrivateServiceConnectEndpoint_MigrateFromSDK checks that
// state written by the last SDKv2 release plans cleanly with the Plugin Framework implementation.
func TestAccResourceRedisCloudPrivateServiceConnectEndpoint_MigrateFromSDK(t *testing.T) {
	gcpProjectId, gcpProjectCheck := envchecks.GCPProjectValueAndCheck()
	variables := privateServiceConnectEndpointVariables(gcpProjectId)

	// ExternalProviders cannot be combined with ConfigFile, so both steps use the file content.
	terraformConfig := utils.GetTestConfig(t, testPrivateServiceConnectEndpointConfigFile)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     envchecks.ComposePreChecks(t, envchecks.RedisCloudCheck, gcpProjectCheck),
		CheckDestroy: testhelpers.CheckProSubscriptionDestroy,
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
				Check:           resource.TestCheckResourceAttrSet(privateServiceConnectEndpointResourceName, "private_service_connect_endpoint_id"),
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
