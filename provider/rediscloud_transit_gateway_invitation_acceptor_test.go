package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourceRedisCloudTransitGatewayInvitationAcceptor_CRUDI(t *testing.T) {
	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	testAwsRegion := os.Getenv("AWS_REGION")
	subscriptionName := acctest.RandomWithPrefix(testResourcePrefix) + "-pro-tgw"

	const invitationsDatasourceName = "data.rediscloud_transit_gateway_invitations.test"
	const acceptorResourceName = "rediscloud_transit_gateway_invitation_acceptor.test"
	const attachmentResourceName = "rediscloud_transit_gateway_attachment.test"
	const routeResourceName = "rediscloud_transit_gateway_route.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccAwsCredentialsPreCheck(t)
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
					utils.GetTestConfig(t, "./transitgateway/testdata/pro_transit_gateway_invitation_acceptor.tf"),
					subscriptionName, testAwsRegion),
			},
			{
				RefreshState: true,
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
					resource.TestCheckResourceAttrSet(attachmentResourceName, "id"),
					resource.TestCheckResourceAttrSet(attachmentResourceName, "aws_tgw_uid"),
					resource.TestCheckResourceAttrSet(attachmentResourceName, "attachment_uid"),
					resource.TestCheckResourceAttr(attachmentResourceName, "status", "available"),
					resource.TestCheckResourceAttr(attachmentResourceName, "attachment_status", "available"),
					resource.TestCheckResourceAttrSet(attachmentResourceName, "aws_account_id"),
					resource.TestCheckResourceAttrSet(attachmentResourceName, "tgw_id"),
					resource.TestCheckResourceAttrSet(routeResourceName, "id"),
					resource.TestCheckResourceAttr(routeResourceName, "cidrs.#", "1"),
					resource.TestCheckResourceAttr(routeResourceName, "cidrs.0", "10.10.20.0/24"),
					testAccCheckTransitGatewayInvitationOnApi(acceptorResourceName, "accepted"),
					testAccCheckTransitGatewayAttachmentOnApi(attachmentResourceName, "available", "available"),
					testAccCheckTransitGatewayRouteCidrsOnApi(routeResourceName, []string{"10.10.20.0/24"}),
				),
			},
			{
				Config: fmt.Sprintf(
					utils.GetTestConfig(t, "./transitgateway/testdata/pro_transit_gateway_route_update.tf"),
					subscriptionName, testAwsRegion),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(routeResourceName, "cidrs.#", "2"),
					resource.TestCheckResourceAttr(routeResourceName, "cidrs.0", "10.10.20.0/24"),
					resource.TestCheckResourceAttr(routeResourceName, "cidrs.1", "10.10.21.0/24"),
					testAccCheckTransitGatewayRouteCidrsOnApi(routeResourceName, []string{"10.10.20.0/24", "10.10.21.0/24"}),
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
				ResourceName:      routeResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckTransitGatewayInvitationOnApi(resourceName string, expectedStatus string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		subId, err := strconv.Atoi(rs.Primary.Attributes["subscription_id"])
		if err != nil {
			return err
		}
		invitationId, err := strconv.Atoi(rs.Primary.Attributes["tgw_invitation_id"])
		if err != nil {
			return err
		}

		apiClient := testProvider.Meta().(*client.ApiClient)
		invitations, err := apiClient.Client.TransitGatewayAttachments.ListInvitations(context.TODO(), subId)
		if err != nil {
			return err
		}

		for _, inv := range invitations {
			if redis.IntValue(inv.Id) == invitationId {
				if redis.StringValue(inv.Status) != expectedStatus {
					return fmt.Errorf("API invitation status mismatch: expected %s, got %s", expectedStatus, redis.StringValue(inv.Status))
				}
				return nil
			}
		}

		return fmt.Errorf("invitation %d not found on API", invitationId)
	}
}

func testAccCheckTransitGatewayAttachmentOnApi(resourceName string, expectedStatus string, expectedAttachmentStatus string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		subId, err := strconv.Atoi(rs.Primary.Attributes["subscription_id"])
		if err != nil {
			return err
		}
		tgwId, err := strconv.Atoi(rs.Primary.Attributes["tgw_id"])
		if err != nil {
			return err
		}

		apiClient := testProvider.Meta().(*client.ApiClient)
		tgwTask, err := apiClient.Client.TransitGatewayAttachments.Get(context.TODO(), subId)
		if err != nil {
			return err
		}

		if tgwTask.Response == nil || tgwTask.Response.Resource == nil {
			return fmt.Errorf("API returned nil response for subscription %d", subId)
		}

		for _, tgw := range tgwTask.Response.Resource.TransitGatewayAttachment {
			if redis.IntValue(tgw.Id) == tgwId {
				if redis.StringValue(tgw.Status) != expectedStatus {
					return fmt.Errorf("API TGW status mismatch: expected %s, got %s", expectedStatus, redis.StringValue(tgw.Status))
				}
				if redis.StringValue(tgw.AttachmentStatus) != expectedAttachmentStatus {
					return fmt.Errorf("API attachment status mismatch: expected %s, got %s", expectedAttachmentStatus, redis.StringValue(tgw.AttachmentStatus))
				}
				if redis.StringValue(tgw.AttachmentUid) == "" {
					return fmt.Errorf("API attachment_uid is empty")
				}
				return nil
			}
		}

		return fmt.Errorf("TGW %d not found on API", tgwId)
	}
}

func testAccCheckTransitGatewayRouteCidrsOnApi(resourceName string, expectedCidrs []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		subId, err := strconv.Atoi(rs.Primary.Attributes["subscription_id"])
		if err != nil {
			return err
		}
		tgwId, err := strconv.Atoi(rs.Primary.Attributes["tgw_id"])
		if err != nil {
			return err
		}

		apiClient := testProvider.Meta().(*client.ApiClient)
		tgwTask, err := apiClient.Client.TransitGatewayAttachments.Get(context.TODO(), subId)
		if err != nil {
			return err
		}

		if tgwTask.Response == nil || tgwTask.Response.Resource == nil {
			return fmt.Errorf("API returned nil response for subscription %d", subId)
		}

		for _, tgw := range tgwTask.Response.Resource.TransitGatewayAttachment {
			if redis.IntValue(tgw.Id) == tgwId {
				apiCidrs := make([]string, 0, len(tgw.Cidrs))
				for _, cidr := range tgw.Cidrs {
					apiCidrs = append(apiCidrs, redis.StringValue(cidr.CidrAddress))
				}

				if len(apiCidrs) != len(expectedCidrs) {
					return fmt.Errorf("API CIDRs count mismatch: expected %d, got %d (expected: %v, got: %v)", len(expectedCidrs), len(apiCidrs), expectedCidrs, apiCidrs)
				}

				for i, expected := range expectedCidrs {
					if apiCidrs[i] != expected {
						return fmt.Errorf("API CIDR mismatch at index %d: expected %s, got %s", i, expected, apiCidrs[i])
					}
				}

				return nil
			}
		}

		return fmt.Errorf("TGW %d not found on API", tgwId)
	}
}
