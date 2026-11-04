package peering_test

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func TestAccResourceRedisCloudActiveActiveSubscriptionPeering_aws(t *testing.T) {
	resourceTags, resourceTagsCheck := envchecks.BYOCResourceTagsValueAndCheck()
	name := utils.RandomWithPrefix()
	awsRegion := "eu-west-1"

	const resourceName = "rediscloud_active_active_subscription_peering.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     envchecks.ComposePreChecks(t, resourceTagsCheck, envchecks.RedisCloudCheck, envchecks.AWSProviderCheck),
		CheckDestroy: testhelpers.CheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				ConfigFile:               config.StaticFile("./testdata/resource_active_active_subscription_peering_aws.tf"),
				ConfigVariables: config.Variables{
					"team":              config.StringVariable(resourceTags.Team),
					"owner":             config.StringVariable(resourceTags.Owner),
					"subscription_name": config.StringVariable(name),
					"aws_region":        config.StringVariable(awsRegion),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(resourceName, "id", regexp.MustCompile(`^\d*/\d*$`)),
					resource.TestCheckResourceAttrSet(resourceName, "status"),
					resource.TestCheckResourceAttr(resourceName, "provider_name", "AWS"),
					resource.TestCheckResourceAttrSet(resourceName, "aws_account_id"),
					resource.TestCheckResourceAttrSet(resourceName, "vpc_id"),
					resource.TestCheckResourceAttrSet(resourceName, "vpc_cidr"),
					resource.TestCheckResourceAttr(resourceName, "vpc_cidrs.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "source_region"),
					resource.TestCheckResourceAttrSet(resourceName, "destination_region"),
					resource.TestCheckResourceAttrSet(resourceName, "aws_peering_id"),
					testAccCheckActiveActivePeeringAwsAttributesMatchApi(resourceName),
				),
			},
			{
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				ConfigFile:               config.StaticFile("./testdata/resource_active_active_subscription_peering_aws.tf"),
				ConfigVariables: config.Variables{
					"team":              config.StringVariable(resourceTags.Team),
					"owner":             config.StringVariable(resourceTags.Owner),
					"subscription_name": config.StringVariable(name),
					"aws_region":        config.StringVariable(awsRegion),
				},
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccResourceRedisCloudActiveActiveSubscriptionPeering_gcp(t *testing.T) {
	gcpVPCProject, gcpVPCProjectCheck := envchecks.ValueAndCheck("GCP_VPC_PROJECT")
	gcpVPCId, gcpVPCIdCheck := envchecks.ValueAndCheck("GCP_VPC_ID")
	name := utils.RandomWithPrefix()

	const resourceName = "rediscloud_active_active_subscription_peering.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, envchecks.RedisCloudCheck, gcpVPCProjectCheck, gcpVPCIdCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/resource_active_active_subscription_peering_gcp.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(name),
					"gcp_project_id":    config.StringVariable(gcpVPCProject),
					"gcp_network_name":  config.StringVariable(gcpVPCId),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(resourceName, "id", regexp.MustCompile(`^\d*/\d*$`)),
					resource.TestCheckResourceAttr(resourceName, "provider_name", "GCP"),
					resource.TestCheckResourceAttrSet(resourceName, "status"),
					resource.TestCheckResourceAttrSet(resourceName, "gcp_project_id"),
					resource.TestCheckResourceAttrSet(resourceName, "gcp_network_name"),
					resource.TestCheckResourceAttrSet(resourceName, "gcp_redis_project_id"),
					resource.TestCheckResourceAttrSet(resourceName, "gcp_redis_network_name"),
				),
			},
			{
				ConfigFile: config.StaticFile("./testdata/resource_active_active_subscription_peering_gcp.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(name),
					"gcp_project_id":    config.StringVariable(gcpVPCProject),
					"gcp_network_name":  config.StringVariable(gcpVPCId),
				},
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccResourceRedisCloudActiveActiveSubscriptionPeering_sdkStateCompatibility
// applies with the last SDKv2 release, then expects the Framework provider to
// plan no change. It uses the GCP fixture because the AWS one declares its own
// terraform block, which prevents plugin-testing from adding the external
// provider source.
func TestAccResourceRedisCloudActiveActiveSubscriptionPeering_sdkStateCompatibility(t *testing.T) {
	gcpVPCProject, gcpVPCProjectCheck := envchecks.ValueAndCheck("GCP_VPC_PROJECT")
	gcpVPCId, gcpVPCIdCheck := envchecks.ValueAndCheck("GCP_VPC_ID")
	name := utils.RandomWithPrefix()

	sdkConfig, err := os.ReadFile("./testdata/resource_active_active_subscription_peering_gcp.tf")
	if err != nil {
		t.Fatal(err)
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     envchecks.ComposePreChecks(t, envchecks.RedisCloudCheck, gcpVPCProjectCheck, gcpVPCIdCheck),
		CheckDestroy: testhelpers.CheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"rediscloud": {
						Source:            "RedisLabs/rediscloud",
						VersionConstraint: lastSDKProviderVersion,
					},
				},
				Config: string(sdkConfig),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(name),
					"gcp_project_id":    config.StringVariable(gcpVPCProject),
					"gcp_network_name":  config.StringVariable(gcpVPCId),
				},
			},
			{
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				ConfigFile:               config.StaticFile("./testdata/resource_active_active_subscription_peering_gcp.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(name),
					"gcp_project_id":    config.StringVariable(gcpVPCProject),
					"gcp_network_name":  config.StringVariable(gcpVPCId),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// testAccCheckActiveActivePeeringAwsAttributesMatchApi verifies that the state
// of an AWS Active-Active peering matches what the API returns, which catches a
// Read that fails to populate the AWS attributes on import.
func testAccCheckActiveActivePeeringAwsAttributesMatchApi(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		subId, err := strconv.Atoi(rs.Primary.Attributes["subscription_id"])
		if err != nil {
			return fmt.Errorf("failed to parse subscription_id: %w", err)
		}

		var peeringId int
		if _, err := fmt.Sscanf(rs.Primary.ID, "%d/%d", &subId, &peeringId); err != nil {
			return fmt.Errorf("failed to parse peering ID from %s: %w", rs.Primary.ID, err)
		}

		apiClient, err := client.GetTestClient()
		if err != nil {
			return fmt.Errorf("failed to get API client: %w", err)
		}

		peerings, err := apiClient.Client.Subscription.ListActiveActiveVPCPeering(context.TODO(), subId)
		if err != nil {
			return fmt.Errorf("failed to list peerings: %w", err)
		}

		for _, region := range peerings {
			for _, peering := range region.VPCPeerings {
				if redis.IntValue(peering.ID) != peeringId {
					continue
				}
				if got, want := rs.Primary.Attributes["aws_account_id"], redis.StringValue(peering.AWSAccountID); got != want {
					return fmt.Errorf("aws_account_id mismatch: state=%q, api=%q", got, want)
				}
				if got, want := rs.Primary.Attributes["vpc_id"], redis.StringValue(peering.VPCId); got != want {
					return fmt.Errorf("vpc_id mismatch: state=%q, api=%q", got, want)
				}
				if got, want := rs.Primary.Attributes["aws_peering_id"], redis.StringValue(peering.AWSPeeringID); got != want {
					return fmt.Errorf("aws_peering_id mismatch: state=%q, api=%q", got, want)
				}
				if got, want := rs.Primary.Attributes["source_region"], redis.StringValue(region.SourceRegion); got != want {
					return fmt.Errorf("source_region mismatch: state=%q, api=%q", got, want)
				}
				if got, want := rs.Primary.Attributes["destination_region"], redis.StringValue(peering.RegionName); got != want {
					return fmt.Errorf("destination_region mismatch: state=%q, api=%q", got, want)
				}
				if got := rs.Primary.Attributes["provider_name"]; got != "AWS" {
					return fmt.Errorf("provider_name mismatch: state=%q, expected=%q", got, "AWS")
				}
				return nil
			}
		}

		return fmt.Errorf("peering %d not found in API response for subscription %d", peeringId, subId)
	}
}
