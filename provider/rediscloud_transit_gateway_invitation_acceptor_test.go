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

func TestAccResourceRedisCloudTransitGatewayInvitationAcceptor_CRUDI(t *testing.T) {
	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")
	testAwsRegion := os.Getenv("AWS_REGION")
	rediscloudAwsAccountId := os.Getenv("REDISCLOUD_AWS_ACCOUNT_ID")
	subscriptionName := acctest.RandomWithPrefix(testResourcePrefix) + "-pro-tgw"

	const invitationsDatasourceName = "data.rediscloud_transit_gateway_invitations.test"
	const acceptorResourceName = "rediscloud_transit_gateway_invitation_acceptor.test"
	const attachmentResourceName = "rediscloud_transit_gateway_attachment.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccAwsPreExistingCloudAccountPreCheck(t)
			testAccAwsCredentialsPreCheck(t)
			testAccRedisCloudAwsAccountPreCheck(t)
		},
		ProviderFactories: providerFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"aws": {
				Source:            "hashicorp/aws",
				VersionConstraint: "~> 5.0",
			},
			"time": {
				Source:            "hashicorp/time",
				VersionConstraint: "~> 0.9",
			},
		},
		CheckDestroy: testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					utils.GetTestConfig(t, "./pro/testdata/transit_gateway_invitation_acceptor.tf"),
					testCloudAccountName, subscriptionName, testAwsRegion, rediscloudAwsAccountId),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Invitations data source checks
					resource.TestCheckResourceAttrSet(invitationsDatasourceName, "subscription_id"),
					resource.TestCheckResourceAttrSet(invitationsDatasourceName, "invitations.#"),
					// Acceptor resource checks
					resource.TestCheckResourceAttrSet(acceptorResourceName, "id"),
					resource.TestCheckResourceAttr(acceptorResourceName, "action", "accept"),
					resource.TestCheckResourceAttrSet(acceptorResourceName, "name"),
					resource.TestCheckResourceAttrSet(acceptorResourceName, "resource_share_uid"),
					resource.TestCheckResourceAttrSet(acceptorResourceName, "status"),
					resource.TestCheckResourceAttrSet(acceptorResourceName, "aws_account_id"),
					resource.TestCheckResourceAttrSet(acceptorResourceName, "shared_date"),
					// Attachment resource checks
					resource.TestCheckResourceAttrSet(attachmentResourceName, "id"),
					resource.TestCheckResourceAttrSet(attachmentResourceName, "aws_tgw_uid"),
					resource.TestCheckResourceAttrSet(attachmentResourceName, "attachment_uid"),
					resource.TestCheckResourceAttr(attachmentResourceName, "status", "available"),
					resource.TestCheckResourceAttr(attachmentResourceName, "attachment_status", "pending-acceptance"),
					resource.TestCheckResourceAttrSet(attachmentResourceName, "aws_account_id"),
					resource.TestCheckResourceAttrSet(attachmentResourceName, "tgw_id"),
					resource.TestCheckResourceAttr(attachmentResourceName, "cidrs.#", "0"),
				),
			},
			{
				ResourceName:            acceptorResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"action"},
			},
			{
				ResourceName:      attachmentResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(
					utils.GetTestConfig(t, "./pro/testdata/transit_gateway_invitation_acceptor_with_cidrs.tf"),
					testCloudAccountName, subscriptionName, testAwsRegion, rediscloudAwsAccountId),
				ExpectError: regexp.MustCompile("Transit Gateway attachment is not active|SUBSCRIPTION_INVALID_REGION_ID"),
			},
		},
	})
}
