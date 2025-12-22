package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourceRedisCloudActiveActiveSubscriptionPeering_aws(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TEST_PEERING")

	name := acctest.RandomWithPrefix(testResourcePrefix)

	cidrRange := os.Getenv("AWS_VPC_CIDR")
	// Choose a CIDR range for the subscription that's unlikely to overlap with any VPC CIDR
	subCidrRange := "10.0.0.0/24"

	overlap, err := cidrRangesOverlap(subCidrRange, cidrRange)
	if err != nil {
		t.Fatalf("AWS_VPC_CIDR is not a valid CIDR range %s: %s", cidrRange, err)
	}
	if overlap {
		subCidrRange = "172.16.0.0/24"
	}

	peeringRegion := os.Getenv("AWS_PEERING_REGION")
	matchesRegex(t, peeringRegion, "^[a-z]+-[a-z]+-\\d+$")

	accountId := os.Getenv("AWS_ACCOUNT_ID")
	matchesRegex(t, accountId, "^\\d+$")

	vpcId := os.Getenv("AWS_VPC_ID")
	matchesRegex(t, vpcId, "^vpc-[a-z\\d]+$")

	const resourceName = "rediscloud_active_active_subscription_peering.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPeeringPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: utils.RenderTestConfig(t, "./peering/testdata/active_active_peering_aws.tf", map[string]string{
					"__SUBSCRIPTION_NAME__":     name,
					"__SUBSCRIPTION_CIDR__":     subCidrRange,
					"__PEERING_SOURCE_REGION__": "us-east-2",
					"__PEERING_DEST_REGION__":   peeringRegion,
					"__AWS_ACCOUNT_ID__":        accountId,
					"__VPC_ID__":                vpcId,
					"__VPC_CIDR__":              cidrRange,
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(resourceName, "id", regexp.MustCompile("^\\d*/\\d*$")),
					resource.TestCheckResourceAttrSet(resourceName, "status"),
					resource.TestCheckResourceAttr(resourceName, "provider_name", "AWS"),
					resource.TestCheckResourceAttrSet(resourceName, "aws_account_id"),
					resource.TestCheckResourceAttrSet(resourceName, "vpc_id"),
					resource.TestCheckResourceAttr(resourceName, "vpc_cidr", cidrRange),
					resource.TestCheckResourceAttr(resourceName, "vpc_cidrs.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "vpc_cidrs.0", cidrRange),
					resource.TestCheckResourceAttrSet(resourceName, "source_region"),
					resource.TestCheckResourceAttrSet(resourceName, "destination_region"),
					resource.TestCheckResourceAttrSet(resourceName, "aws_peering_id"),
					testAccCheckActiveActivePeeringAwsAttributesMatchApi(resourceName),
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

func TestAccResourceRedisCloudActiveActiveSubscriptionPeering_gcp(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TEST_PEERING")

	name := acctest.RandomWithPrefix(testResourcePrefix)

	tf := fmt.Sprintf(testAccResourceRedisCloudActiveActiveSubscriptionPeeringGCP,
		name,
		os.Getenv("GCP_VPC_PROJECT"),
		os.Getenv("GCP_VPC_ID"),
	)
	const resourceName = "rediscloud_active_active_subscription_peering.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: tf,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(resourceName, "id", regexp.MustCompile("^\\d*/\\d*$")),
					resource.TestCheckResourceAttr(resourceName, "provider_name", "GCP"),
					resource.TestCheckResourceAttrSet(resourceName, "status"),
					resource.TestCheckResourceAttrSet(resourceName, "gcp_project_id"),
					resource.TestCheckResourceAttrSet(resourceName, "gcp_network_name"),
					resource.TestCheckResourceAttrSet(resourceName, "gcp_redis_project_id"),
					resource.TestCheckResourceAttrSet(resourceName, "gcp_redis_network_name"),
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

// testAccCheckActiveActivePeeringAwsAttributesMatchApi verifies that the Terraform state
// for an AWS Active-Active peering matches what the API returns. This is particularly
// important for testing import functionality - if the Read function doesn't properly
// populate AWS attributes during import, this check will fail.
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

		peeringIdStr := rs.Primary.ID
		// ID format is "subId/peeringId"
		var peeringId int
		if _, err := fmt.Sscanf(peeringIdStr, "%d/%d", &subId, &peeringId); err != nil {
			return fmt.Errorf("failed to parse peering ID from %s: %w", peeringIdStr, err)
		}

		apiClient, err := getTestClient()
		if err != nil {
			return fmt.Errorf("failed to get API client: %w", err)
		}

		peerings, err := apiClient.Client.Subscription.ListActiveActiveVPCPeering(context.TODO(), subId)
		if err != nil {
			return fmt.Errorf("failed to list peerings: %w", err)
		}

		// Find the peering by ID
		for _, region := range peerings {
			for _, peering := range region.VPCPeerings {
				if redis.IntValue(peering.ID) == peeringId {
					// Verify AWS-specific attributes match API
					if stateAwsAccountId := rs.Primary.Attributes["aws_account_id"]; stateAwsAccountId != redis.StringValue(peering.AWSAccountID) {
						return fmt.Errorf("aws_account_id mismatch: state=%q, api=%q", stateAwsAccountId, redis.StringValue(peering.AWSAccountID))
					}
					if stateVpcId := rs.Primary.Attributes["vpc_id"]; stateVpcId != redis.StringValue(peering.VPCId) {
						return fmt.Errorf("vpc_id mismatch: state=%q, api=%q", stateVpcId, redis.StringValue(peering.VPCId))
					}
					if stateAwsPeeringId := rs.Primary.Attributes["aws_peering_id"]; stateAwsPeeringId != redis.StringValue(peering.AWSPeeringID) {
						return fmt.Errorf("aws_peering_id mismatch: state=%q, api=%q", stateAwsPeeringId, redis.StringValue(peering.AWSPeeringID))
					}
					if stateSourceRegion := rs.Primary.Attributes["source_region"]; stateSourceRegion != redis.StringValue(region.SourceRegion) {
						return fmt.Errorf("source_region mismatch: state=%q, api=%q", stateSourceRegion, redis.StringValue(region.SourceRegion))
					}
					if stateDestRegion := rs.Primary.Attributes["destination_region"]; stateDestRegion != redis.StringValue(peering.RegionName) {
						return fmt.Errorf("destination_region mismatch: state=%q, api=%q", stateDestRegion, redis.StringValue(peering.RegionName))
					}

					// Verify provider_name is "AWS"
					if stateProviderName := rs.Primary.Attributes["provider_name"]; stateProviderName != "AWS" {
						return fmt.Errorf("provider_name mismatch: state=%q, expected=%q", stateProviderName, "AWS")
					}

					return nil
				}
			}
		}

		return fmt.Errorf("peering %d not found in API response for subscription %d", peeringId, subId)
	}
}

const testAccResourceRedisCloudActiveActiveSubscriptionPeeringGCP = `
data "rediscloud_payment_method" "card" {
	card_type = "Visa"
	last_four_numbers = "5556"
}

resource "rediscloud_active_active_subscription" "example" {
  name = "%s"
  payment_method_id = data.rediscloud_payment_method.card.id
  cloud_provider = "GCP"
  creation_plan {
    memory_limit_in_gb = 1
    quantity = 1
    region {
      region = "europe-west1"
      networking_deployment_cidr = "192.168.0.0/24"
      write_operations_per_second = 1000
      read_operations_per_second = 1000
    }
    region {
      region = "europe-west2"
      networking_deployment_cidr = "192.168.1.0/24"
      write_operations_per_second = 1000
      read_operations_per_second = 1000
    }
  }
}

resource "rediscloud_active_active_subscription_peering" "test" {
  subscription_id = rediscloud_active_active_subscription.example.id
  provider_name = "GCP"
  gcp_project_id = "%s"
  gcp_network_name = "%s"

  source_region = "europe-west2"
}
`
