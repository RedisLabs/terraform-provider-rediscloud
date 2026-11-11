package privatelink_test

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"

	pl "github.com/RedisLabs/rediscloud-go-api/service/privatelink"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

const (
	testActiveActivePrivateLinkConfigFile                   = "testdata/resource_active_active_private_link.tf"
	testActiveActivePrivateLinkConfigWithoutPrivateLinkFile = "testdata/resource_active_active_private_link_without_privatelink.tf"
	activeActivePrivateLinkResourceName                     = "rediscloud_active_active_private_link.aa_private_link"
	activeActivePrivateLinkSubscriptionResourceName         = "rediscloud_active_active_subscription.aa_subscription"
	activeActivePrivateLinkRegionsDataSourceName            = "data.rediscloud_active_active_subscription_regions.aa_regions_info"
)

func TestAccResourceRedisCloudActiveActivePrivateLink_CRUDI(t *testing.T) {
	resourceTags, resourceTagsCheck := envchecks.BYOCResourceTagsValueAndCheck()
	_, cloudAccountCheck := envchecks.AWSBYOCValueAndCheck()

	subscriptionName := utils.RandomWithPrefix() + "-aa-private-link"
	shareName := utils.RandomWithPrefix() + "-privatelink-aa"
	password := acctest.RandString(20)

	baseVariables := func() config.Variables {
		return config.Variables{
			"subscription_name": config.StringVariable(subscriptionName),
			"database_password": config.StringVariable(password),
			"team":              config.StringVariable(resourceTags.Team),
			"owner":             config.StringVariable(resourceTags.Owner),
		}
	}
	withPrivateLink := func(principals ...config.Variable) config.Variables {
		variables := baseVariables()
		variables["share_name"] = config.StringVariable(shareName)
		variables["principals"] = config.ListVariable(principals...)
		return variables
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, resourceTagsCheck, envchecks.RedisCloudCheck, cloudAccountCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile(testActiveActivePrivateLinkConfigFile),
				ConfigVariables: withPrivateLink(
					principalVariable("123456789012", "terraform test aws account"),
					principalVariable("688576139039", "terraform test aws account 2"),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(activeActivePrivateLinkResourceName, "id"),
					resource.TestCheckResourceAttrPair(activeActivePrivateLinkResourceName, "subscription_id", activeActivePrivateLinkSubscriptionResourceName, "id"),
					resource.TestCheckResourceAttrPair(activeActivePrivateLinkResourceName, "region_id", activeActivePrivateLinkRegionsDataSourceName, "regions.0.region_id"),
					resource.TestCheckResourceAttr(activeActivePrivateLinkResourceName, "share_name", shareName),
					resource.TestCheckResourceAttr(activeActivePrivateLinkResourceName, "principal.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs(activeActivePrivateLinkResourceName, "principal.*", map[string]string{
						"principal":       "123456789012",
						"principal_type":  "aws_account",
						"principal_alias": "terraform test aws account",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(activeActivePrivateLinkResourceName, "principal.*", map[string]string{
						"principal":       "688576139039",
						"principal_type":  "aws_account",
						"principal_alias": "terraform test aws account 2",
					}),
					resource.TestCheckResourceAttrSet(activeActivePrivateLinkResourceName, "resource_configuration_id"),
					resource.TestCheckResourceAttrSet(activeActivePrivateLinkResourceName, "resource_configuration_arn"),
					resource.TestCheckResourceAttrSet(activeActivePrivateLinkResourceName, "share_arn"),
					resource.TestCheckResourceAttrSet(activeActivePrivateLinkResourceName, "connections.#"),
					resource.TestCheckResourceAttrSet(activeActivePrivateLinkResourceName, "databases.#"),
				),
			},
			{
				ConfigFile: config.StaticFile(testActiveActivePrivateLinkConfigFile),
				ConfigVariables: withPrivateLink(
					principalVariable("123456789012", "terraform test aws account"),
					principalVariable("345678901234", ""),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(activeActivePrivateLinkResourceName, "principal.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs(activeActivePrivateLinkResourceName, "principal.*", map[string]string{
						"principal":      "345678901234",
						"principal_type": "aws_account",
					}),
				),
			},
			{
				ConfigFile: config.StaticFile(testActiveActivePrivateLinkConfigFile),
				ConfigVariables: withPrivateLink(
					principalVariable("123456789012", "terraform test aws account"),
					principalVariable("345678901234", ""),
				),
				ResourceName:      activeActivePrivateLinkResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ConfigFile:      config.StaticFile(testActiveActivePrivateLinkConfigWithoutPrivateLinkFile),
				ConfigVariables: baseVariables(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(activeActivePrivateLinkSubscriptionResourceName, "id"),
					testAccCheckActiveActivePrivateLinkDeleted(activeActivePrivateLinkSubscriptionResourceName, activeActivePrivateLinkRegionsDataSourceName),
				),
			},
		},
	})
}

// TestAccResourceRedisCloudActiveActivePrivateLink_MigrateFromSDK checks that state
// written by the last SDKv2 release plans cleanly with the Plugin Framework implementation.
func TestAccResourceRedisCloudActiveActivePrivateLink_MigrateFromSDK(t *testing.T) {
	resourceTags, resourceTagsCheck := envchecks.BYOCResourceTagsValueAndCheck()
	_, cloudAccountCheck := envchecks.AWSBYOCValueAndCheck()

	variables := config.Variables{
		"subscription_name": config.StringVariable(utils.RandomWithPrefix() + "-aa-private-link"),
		"database_password": config.StringVariable(acctest.RandString(20)),
		"share_name":        config.StringVariable(utils.RandomWithPrefix() + "-privatelink-aa"),
		"team":              config.StringVariable(resourceTags.Team),
		"owner":             config.StringVariable(resourceTags.Owner),
		"principals": config.ListVariable(
			principalVariable("123456789012", "terraform test aws account"),
			principalVariable("688576139039", ""),
		),
	}

	// ExternalProviders cannot be combined with ConfigFile, so both steps use the file content.
	terraformConfig := utils.GetTestConfig(t, testActiveActivePrivateLinkConfigFile)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     envchecks.ComposePreChecks(t, resourceTagsCheck, envchecks.RedisCloudCheck, cloudAccountCheck),
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
				Check:           resource.TestCheckResourceAttr(activeActivePrivateLinkResourceName, "principal.#", "2"),
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

func testAccCheckActiveActivePrivateLinkDeleted(subscriptionResourceName, regionsDataSourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		subResource, ok := s.RootModule().Resources[subscriptionResourceName]
		if !ok {
			return fmt.Errorf("subscription not found: %s", subscriptionResourceName)
		}

		subId, err := strconv.Atoi(subResource.Primary.ID)
		if err != nil {
			return err
		}

		// Get region_id from the regions data source
		regionsResource, ok := s.RootModule().Resources[regionsDataSourceName]
		if !ok {
			return fmt.Errorf("regions data source not found: %s", regionsDataSourceName)
		}

		regionIdStr := regionsResource.Primary.Attributes["regions.0.region_id"]
		regionId, err := strconv.Atoi(regionIdStr)
		if err != nil {
			return fmt.Errorf("could not parse region_id: %w", err)
		}

		apiClient, err := client.GetTestClient()
		if err != nil {
			return err
		}

		_, err = apiClient.Client.PrivateLink.GetActiveActivePrivateLink(context.TODO(), subId, regionId)
		if err == nil {
			return fmt.Errorf("active-active privatelink for subscription %d region %d still exists after deletion", subId, regionId)
		}

		var notFound *pl.NotFoundActiveActive
		if !errors.As(err, &notFound) {
			return fmt.Errorf("unexpected error checking active-active privatelink: %w", err)
		}

		return nil
	}
}
