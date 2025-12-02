package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceRedisCloudActiveActiveTransitGatewayInvitationAcceptor_CRUDI(t *testing.T) {
	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")
	testAwsRegion := os.Getenv("AWS_REGION")
	subscriptionName := acctest.RandomWithPrefix(testResourcePrefix) + "-aa-tgw-inv"

	const invitationsDatasourceName = "data.rediscloud_active_active_transit_gateway_invitations.test"
	const acceptorResourceName = "rediscloud_active_active_transit_gateway_invitation_acceptor.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccAwsPreExistingCloudAccountPreCheck(t)
			testAccAwsCredentialsPreCheck(t)
		},
		ProviderFactories: providerFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"aws": {
				Source:            "hashicorp/aws",
				VersionConstraint: "~> 5.0",
			},
		},
		CheckDestroy: testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					utils.GetTestConfig(t, "./activeactive/testdata/transit_gateway_invitation_acceptor.tf"),
					testCloudAccountName, subscriptionName, testAwsRegion),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(invitationsDatasourceName, "subscription_id"),
					resource.TestCheckResourceAttrSet(invitationsDatasourceName, "invitations.#"),
					resource.TestCheckResourceAttrSet(acceptorResourceName, "id"),
					resource.TestCheckResourceAttr(acceptorResourceName, "action", "accept"),
					resource.TestCheckResourceAttrSet(acceptorResourceName, "tgw_id"),
					resource.TestCheckResourceAttrSet(acceptorResourceName, "aws_tgw_uid"),
					resource.TestCheckResourceAttrSet(acceptorResourceName, "status"),
					resource.TestCheckResourceAttrSet(acceptorResourceName, "aws_account_id"),
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
