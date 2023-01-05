package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceRedisCloudActiveActiveSubscriptionPeering_aws(t *testing.T) {

	name := acctest.RandomWithPrefix(testResourcePrefix)

	//testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")
	cidrRange := os.Getenv("AWS_VPC_CIDR")

	// Choose a CIDR range for the subscription that's unlikely to overlap with any VPC CIDR
	subCidrRange := "10.202.0.0/24"

	overlap, err := cidrRangesOverlap(subCidrRange, cidrRange)
	if err != nil {
		t.Fatalf("AWS_VPC_CIDR is not a valid CIDR range %s: %s", cidrRange, err)
	}
	if overlap {
		subCidrRange = "172.16.0.0/24"
	}

	os.Setenv("AWS_PEERING_REGION", "us-east-1")
	os.Setenv("AWS_ACCOUNT_ID", "277885626557")
	os.Setenv("AWS_VPC_ID", "vpc-0896d84b605a91d75")

	peeringRegion := os.Getenv("AWS_PEERING_REGION")
	matchesRegex(t, peeringRegion, "^[a-z]+-[a-z]+-\\d+$")

	accountId := os.Getenv("AWS_ACCOUNT_ID")
	matchesRegex(t, accountId, "^\\d+$")

	vpcId := os.Getenv("AWS_VPC_ID")
	matchesRegex(t, vpcId, "^vpc-[a-z\\d]+$")

	tf := fmt.Sprintf(testAccResourceRedisCloudActiveActiveSubscriptionPeeringAWS,
		name,
		subCidrRange,
		peeringRegion,
		accountId,
		vpcId,
		cidrRange,
	)
	resourceName := "rediscloud_active_active_subscription_peering.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPeeringPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: tf,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(resourceName, "id", regexp.MustCompile("^\\d*/\\d*$")),
					resource.TestCheckResourceAttrSet(resourceName, "status"),
					resource.TestCheckResourceAttrSet(resourceName, "provider_name"),
					resource.TestCheckResourceAttrSet(resourceName, "aws_account_id"),
					resource.TestCheckResourceAttrSet(resourceName, "vpc_id"),
					resource.TestCheckResourceAttr(resourceName, "vpc_cidrs.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "source_region"),
					resource.TestCheckResourceAttrSet(resourceName, "destination_region"),
					resource.TestCheckResourceAttrSet(resourceName, "aws_peering_id"),
				),
			},
		},
	})
}

func TestAccResourceRedisCloudActiveActiveSubscriptionPeering_gcp(t *testing.T) {

	if testing.Short() {
		t.Skip("Required environment variables currently not available under CI")
	}

	name := acctest.RandomWithPrefix(testResourcePrefix)

	tf := fmt.Sprintf(testAccResourceRedisCloudSubscriptionPeeringGCP,
		name,
		os.Getenv("GCP_VPC_PROJECT"),
		os.Getenv("GCP_VPC_ID"),
	)
	resourceName := "rediscloud_active_active_subscription_peering.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: tf,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(resourceName, "id", regexp.MustCompile("^\\d*/\\d*$")),
					resource.TestCheckResourceAttr(resourceName, "provider_name", "GCP"),
					resource.TestCheckResourceAttrSet(resourceName, "status"),
					resource.TestCheckResourceAttrSet(resourceName, "gcp_project_id"),
					resource.TestCheckResourceAttrSet(resourceName, "gcp_network_name"),
					resource.TestCheckResourceAttrSet(resourceName, "gcp_redis_project_id"),
					resource.TestCheckResourceAttrSet(resourceName, "gcp_redis_network_name"),
					resource.TestCheckResourceAttrSet(resourceName, "gcp_peering_id"),
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

const testAccResourceRedisCloudActiveActiveSubscriptionPeeringAWS = `
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

resource "rediscloud_active_active_subscription" "example" {
    name = "%s"
    payment_method_id = data.rediscloud_payment_method.card.id
    cloud_provider = "AWS"
    creation_plan {
        memory_limit_in_gb = 1
        quantity = 1
        support_oss_cluster_api=false
        region {
            region = "us-east-1"
            networking_deployment_cidr = "192.168.0.0/24"
            write_operations_per_second = 1000
            read_operations_per_second = 1000
        }
        region {
            region = "us-east-2"
            networking_deployment_cidr = "10.0.1.0/24"
            write_operations_per_second = 1000
            read_operations_per_second = 1000
        }
    }
}

resource "rediscloud_active_active_subscription_peering" "test" {
  subscription_id = rediscloud_active_active_subscription.example.id
  provider_name = "AWS"
  source_region = "us-east-1"
  destination_region = "eu-west-2"
  aws_account_id = "%s"
  vpc_id = "%s"
  vpc_cidrs = ["%s"]
}

resource "aws_vpc_peering_connection_accepter" "example-peering" {
	vpc_peering_connection_id = rediscloud_active_active_subscription_peering.test.aws_peering_id
	auto_accept               = true
}
`

const testAccResourceRedisCloudActiveActiveSubscriptionPeeringGCP = `
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

resource "rediscloud_subscription" "example" {
  name = "%s"
  payment_method_id = data.rediscloud_payment_method.card.id
  cloud_provider = "GCP"
  creation_plan {
	  memory_limit_in_gb = 1
	  quantity = 1
	  support_oss_cluster_api=false
	  region {
		  region = "us-east-1"
		  networking_deployment_cidr = "192.168.0.0/24"
		  write_operations_per_second = 1000
		  read_operations_per_second = 1000
	  }
	  region {
		  region = "us-east-2"
		  networking_deployment_cidr = "10.0.1.0/24"
		  write_operations_per_second = 1000
		  read_operations_per_second = 1000
	  }
  }
}

resource "rediscloud_active_active_subscription_peering" "test" {
  subscription_id = rediscloud_subscription.example.id
  provider_name = "GCP"
  gcp_project_id = "%s"
  gcp_network_name = "%s"
}
`
