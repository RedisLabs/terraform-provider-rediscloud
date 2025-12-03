package provider

import (
	"fmt"
	"os"
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
	subscriptionName := acctest.RandomWithPrefix(testResourcePrefix) + "-pro-tgw-inv"

	const invitationsDatasourceName = "data.rediscloud_transit_gateway_invitations.test"
	const acceptorResourceName = "rediscloud_transit_gateway_invitation_acceptor.test"

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
					resource.TestCheckResourceAttrSet(invitationsDatasourceName, "subscription_id"),
					resource.TestCheckResourceAttrSet(invitationsDatasourceName, "invitations.#"),
					resource.TestCheckResourceAttrSet(acceptorResourceName, "id"),
					resource.TestCheckResourceAttr(acceptorResourceName, "action", "accept"),
					resource.TestCheckResourceAttrSet(acceptorResourceName, "name"),
					resource.TestCheckResourceAttrSet(acceptorResourceName, "resource_share_uid"),
					resource.TestCheckResourceAttrSet(acceptorResourceName, "status"),
					resource.TestCheckResourceAttrSet(acceptorResourceName, "aws_account_id"),
					resource.TestCheckResourceAttrSet(acceptorResourceName, "shared_date"),
				),
			},
			{
				ResourceName:            acceptorResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"action"},
			},
		},
	})
}
