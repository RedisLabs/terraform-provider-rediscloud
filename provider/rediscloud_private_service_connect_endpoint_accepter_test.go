package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/psc"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func TestAccResourceRedisCloudPrivateServiceConnectEndpointAccepter_Create(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	baseName := testRandomWithPrefix() + "-pro-pscea"

	const resourceName = "rediscloud_private_service_connect_endpoint_accepter.accepter"
	gcpProjectId := os.Getenv("GCP_PROJECT_ID")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); testAccGcpProjectPreCheck(t); testAccGcpCredentialsPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"google": {
				Source:            "hashicorp/google",
				VersionConstraint: "~> 6.5",
			},
		},
		CheckDestroy: testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./psc/testdata/pro_pscea.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(baseName),
					"gcp_project_id":    config.StringVariable(gcpProjectId),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					func(s *terraform.State) error {
						r := s.RootModule().Resources[resourceName]

						accepterId, err := toPscEndpointAccepterId(r.Primary.ID)
						if err != nil {
							return fmt.Errorf("couldn't parse the accepter ID: %s", r.Primary.ID)
						}

						client := sharedTestClient(t)
						endpoints, err := client.Client.PrivateServiceConnect.GetEndpoints(context.TODO(), accepterId.subscriptionId, accepterId.pscServiceId)
						if err != nil {
							return err
						}

						endpoint := findPrivateServiceConnectEndpoints(accepterId.endpointId, endpoints.Endpoints)
						if endpoint == nil {
							return fmt.Errorf("couldn't find endpoint with ID: %d", accepterId.endpointId)
						}

						if redis.StringValue(endpoint.Status) != psc.EndpointStatusActive {
							return fmt.Errorf("expected endpoint status to be active - current status %s", redis.StringValue(endpoint.Status))
						}

						return nil
					},
				),
			},
		},
	})
}
