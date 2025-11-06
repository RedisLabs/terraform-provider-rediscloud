package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestAccResourceRedisCloudActiveActiveTransitGatewayAttachment_CRUDI tests the basic lifecycle of an Active-Active TGW attachment.
// Note: This test cannot verify successful CIDR updates because that requires manual acceptance of the attachment in the AWS console.
func TestAccResourceRedisCloudActiveActiveTransitGatewayAttachment_CRUDI(t *testing.T) {
	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")
	testTgwId := os.Getenv("AWS_TEST_TGW_ID")
	subscriptionName := acctest.RandomWithPrefix(testResourcePrefix) + "-aa-tgwa"

	const resourceName = "rediscloud_active_active_transit_gateway_attachment.test"
	const datasourceName = "data.rediscloud_active_active_transit_gateway.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingTgwCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			// Step 1: Read TGW datasource before attachment
			{
				Config: fmt.Sprintf(
					utils.GetTestConfig(t, "./activeactive/testdata/transit_gateway_datasource.tf"),
					testCloudAccountName, subscriptionName, testTgwId),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(datasourceName, "id"),
					resource.TestCheckResourceAttr(datasourceName, "aws_tgw_uid", testTgwId),
					resource.TestCheckResourceAttr(datasourceName, "attachment_uid", ""),
					resource.TestCheckResourceAttr(datasourceName, "status", "available"),
					resource.TestCheckResourceAttr(datasourceName, "attachment_status", ""),
					resource.TestCheckResourceAttrSet(datasourceName, "aws_account_id"),
					resource.TestCheckResourceAttr(datasourceName, "cidrs.#", "0"),
				),
			},
			// Step 2: Create TGW attachment (will be pending-acceptance)
			{
				Config: fmt.Sprintf(
					utils.GetTestConfig(t, "./activeactive/testdata/transit_gateway_attachment.tf"),
					testCloudAccountName, subscriptionName, testTgwId),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "aws_tgw_uid", testTgwId),
					resource.TestCheckResourceAttrSet(resourceName, "attachment_uid"),
					resource.TestCheckResourceAttr(resourceName, "status", "available"),
					resource.TestCheckResourceAttr(resourceName, "attachment_status", "pending-acceptance"),
					resource.TestCheckResourceAttrSet(resourceName, "aws_account_id"),
					resource.TestCheckResourceAttrSet(resourceName, "region_id"),
					resource.TestCheckResourceAttrSet(resourceName, "tgw_id"),
					resource.TestCheckResourceAttr(resourceName, "cidrs.#", "0"),
				),
			},
			// Step 3: Test import
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Step 4: Verify CIDRs cannot be added while pending-acceptance
			// This is expected behaviour - attachment must be accepted first
			{
				Config: fmt.Sprintf(
					utils.GetTestConfig(t, "./activeactive/testdata/transit_gateway_attachment_with_cidrs.tf"),
					testCloudAccountName, subscriptionName, testTgwId),
				ExpectError: regexp.MustCompile("Transit Gateway attachment is not active|SUBSCRIPTION_INVALID_REGION_ID"),
			},
		},
	})
}
