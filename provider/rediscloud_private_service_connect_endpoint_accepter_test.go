package provider

import (
	"context"
	"fmt"
	client2 "github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"os"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/psc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourceRedisCloudPrivateServiceConnectEndpointAccepter_Create(t *testing.T) {

	testAccRequiresEnvVar(t, "EXECUTE_TESTS")

	baseName := acctest.RandomWithPrefix(testResourcePrefix) + "-pro-pscea"

	const resourceName = "rediscloud_private_service_connect_endpoint_accepter.accepter"
	gcpProjectId := os.Getenv("GCP_PROJECT_ID")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccGcpProjectPreCheck(t); testAccGcpCredentialsPreCheck(t) },
		ProviderFactories: providerFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"google": {
				Source:            "hashicorp/google",
				VersionConstraint: "~> 6.5",
			},
		},
		CheckDestroy: testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudPrivateServiceConnectEndpointAccepterPro, baseName, gcpProjectId),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					func(s *terraform.State) error {
						r := s.RootModule().Resources[resourceName]

						accepterId, err := toPscEndpointAccepterId(r.Primary.ID)
						if err != nil {
							return fmt.Errorf("couldn't parse the accepter ID: %s", r.Primary.ID)
						}

						client := testProvider.Meta().(*client2.ApiClient)
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

const testAccResourceRedisCloudPrivateServiceConnectEndpointAccepterPro = `
data "rediscloud_payment_method" "card" {
	card_type = "Visa"
	last_four_numbers = "5556"
}

resource "rediscloud_subscription" "subscription" {
  name = "%s"
  payment_method_id = data.rediscloud_payment_method.card.id

  cloud_provider {
    provider = "GCP"
    region {
      region                     = "us-central1"
      networking_deployment_cidr = "10.0.0.0/24"
    }
  }

  creation_plan {
    dataset_size_in_gb           = 1
    quantity                     = 1
    replication                  = true
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 20000
  }
}

resource "rediscloud_private_service_connect" "psc" {
  subscription_id = rediscloud_subscription.subscription.id
}

resource "rediscloud_private_service_connect_endpoint" "psce" {
  subscription_id = rediscloud_subscription.subscription.id
  private_service_connect_service_id = rediscloud_private_service_connect.psc.private_service_connect_service_id
  gcp_project_id = "%[2]s"
  gcp_vpc_name = google_compute_network.network.name
  gcp_vpc_subnet_name = google_compute_subnetwork.subnet.name
  endpoint_connection_name = "redis-${rediscloud_subscription.subscription.id}"
}

resource "google_compute_network" "network" {
  project                 = "%[2]s"
  name                    = "%[1]s"
  auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "subnet" {
  project       = "%[2]s"
  name          = "%[1]s"
  ip_cidr_range = "10.0.1.0/24"
  region        = "us-central1"
  network       = google_compute_network.network.id
}

locals {
  service_attachment_count = 1
}

resource "google_compute_address" "default" {
  count = local.service_attachment_count

  project      = "%[2]s"
  name         = rediscloud_private_service_connect_endpoint.psce.service_attachments[count.index].ip_address_name
  subnetwork   = google_compute_subnetwork.subnet.id
  address_type = "INTERNAL"
  region       = "us-central1"
}

resource "google_compute_forwarding_rule" "default" {
  count = local.service_attachment_count

  name                  = rediscloud_private_service_connect_endpoint.psce.service_attachments[count.index].forwarding_rule_name
  project               = "%[2]s"
  region                = "us-central1"
  ip_address            = google_compute_address.default[count.index].id
  network               = google_compute_network.network.name
  target                = rediscloud_private_service_connect_endpoint.psce.service_attachments[count.index].name
  load_balancing_scheme = ""
}

resource "rediscloud_private_service_connect_endpoint_accepter" "accepter" {
  subscription_id                     = rediscloud_subscription.subscription.id
  private_service_connect_service_id  = rediscloud_private_service_connect.psc.private_service_connect_service_id
  private_service_connect_endpoint_id = rediscloud_private_service_connect_endpoint.psce.private_service_connect_endpoint_id

  action = "accept"

  depends_on = [google_compute_forwarding_rule.default]
}
`
