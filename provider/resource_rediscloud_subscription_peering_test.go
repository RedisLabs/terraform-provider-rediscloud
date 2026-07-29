package provider_test

import (
	"fmt"
	"net"
	"regexp"
	"testing"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccResourceRedisCloudSubscriptionPeering_aws(t *testing.T) {

	name := testRandomWithPrefix()

	cloudAccountName, cloudAccountCheck := envchecks.AWSBYOCValueAndCheck()
	awsPeering, awsPeeringCheck := envchecks.AwsPeeringValueAndCheck()

	// Chose a CIDR range for the subscription that's unlikely to overlap with the peer VPC.
	// Env format is validated in PreCheck; offline the env is empty and the test skips at the
	// TF_ACC gate, so tolerate an empty/invalid AWS_VPC_CIDR here rather than failing early.
	subCidrRange := "10.0.0.0/24"
	if overlap, err := cidrRangesOverlap(subCidrRange, awsPeering.VpcCidr); err == nil && overlap {
		subCidrRange = "172.16.0.0/24"
	}

	tf := fmt.Sprintf(testAccResourceRedisCloudSubscriptionPeeringAWS,
		cloudAccountName,
		name,
		subCidrRange,
		awsPeering.Region,
		awsPeering.AccountId,
		awsPeering.VpcId,
		awsPeering.VpcCidr,
	)
	const resourceName = "rediscloud_subscription_peering.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: envchecks.ComposePreChecks(t,
			envchecks.RedisCloudCheck,
			awsPeeringCheck,
			cloudAccountCheck,
			// Validate env-var formats here (after the TF_ACC gate) so the test skips
			// cleanly offline instead of t.Fatal-ing in the test body.
			func(t *testing.T) bool {
				if _, err := cidrRangesOverlap("10.0.0.0/24", awsPeering.VpcCidr); err != nil {
					t.Errorf("AWS_VPC_CIDR is not a valid CIDR range %q: %s", awsPeering.VpcCidr, err)
					return false
				}
				matchesRegex(t, awsPeering.Region, "^[a-z]+-[a-z]+-\\d+$")
				matchesRegex(t, awsPeering.AccountId, "^\\d+$")
				matchesRegex(t, awsPeering.VpcId, "^vpc-[a-z\\d]+$")
				return true
			},
		),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: tf,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(resourceName, "id", regexp.MustCompile("^\\d*/\\d*$")),
					resource.TestCheckResourceAttrSet(resourceName, "status"),
					resource.TestCheckResourceAttrSet(resourceName, "provider_name"),
					resource.TestCheckResourceAttrSet(resourceName, "aws_account_id"),
					resource.TestCheckResourceAttrSet(resourceName, "vpc_id"),
					resource.TestCheckResourceAttr(resourceName, "vpc_cidr", awsPeering.VpcCidr),
					resource.TestCheckResourceAttr(resourceName, "vpc_cidrs.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "vpc_cidrs.0", awsPeering.VpcCidr),
					resource.TestCheckResourceAttrSet(resourceName, "region"),
					resource.TestCheckResourceAttrSet(resourceName, "aws_peering_id"),
				),
			},
		},
	})
}

func TestAccResourceRedisCloudSubscriptionPeering_gcp(t *testing.T) {

	name := testRandomWithPrefix()
	gcpVpcProject, gcpVpcProjectCheck := envchecks.ValueAndCheck("GCP_VPC_PROJECT")
	gcpVpcId, gcpVpcIdCheck := envchecks.ValueAndCheck("GCP_VPC_ID")

	tf := fmt.Sprintf(testAccResourceRedisCloudSubscriptionPeeringGCP,
		name,
		gcpVpcProject,
		gcpVpcId,
	)
	const resourceName = "rediscloud_subscription_peering.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, envchecks.RedisCloudCheck, gcpVpcProjectCheck, gcpVpcIdCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testAccCheckProSubscriptionDestroy,
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

func matchesRegex(t *testing.T, value string, regex string) {
	if !regexp.MustCompile(regex).MatchString(value) {
		t.Fatalf("%s doesn't match regex %s", value, regex)
	}
}

func cidrRangesOverlap(cidr1 string, cidr2 string) (bool, error) {
	_, first, err := net.ParseCIDR(cidr1)
	if err != nil {
		return false, err
	}
	_, second, err := net.ParseCIDR(cidr2)
	if err != nil {
		return false, err
	}

	overlaps := first.Contains(second.IP) || second.Contains(first.IP)

	return overlaps, nil
}

const testAccResourceRedisCloudSubscriptionPeeringAWS = `
data "rediscloud_payment_method" "card" {
	card_type = "Visa"
	last_four_numbers = "5556"
}

data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type = "AWS" 
  name = "%s"
}

resource "rediscloud_subscription" "example" {
  name = "%s"
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage = "ram"

  cloud_provider {
    provider = data.rediscloud_cloud_account.account.provider_type
    cloud_account_id = data.rediscloud_cloud_account.account.id
    region {
      region = "eu-west-1"
      networking_deployment_cidr = "%s"
      preferred_availability_zones = ["eu-west-1a"]
    }
  }

  creation_plan {
    memory_limit_in_gb = 1
    quantity = 1
    replication=false
    support_oss_cluster_api=false
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 10000
  }
}

resource "rediscloud_subscription_peering" "test" {
  subscription_id = rediscloud_subscription.example.id
  provider_name = "AWS"
  region = "%s"
  aws_account_id = "%s"
  vpc_id = "%s"
  vpc_cidrs = ["%s"]
}
`

const testAccResourceRedisCloudSubscriptionPeeringGCP = `
data "rediscloud_payment_method" "card" {
	card_type = "Visa"
	last_four_numbers = "5556"
}

resource "rediscloud_subscription" "example" {
  name = "%s"
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage = "ram"

  cloud_provider {
    provider = "GCP"
    cloud_account_id = 1
    region {
      region = "europe-west1"
      networking_deployment_cidr = "192.168.0.0/24"
      preferred_availability_zones = []
    }
  }

  creation_plan {
    memory_limit_in_gb = 1
    quantity = 1
    replication=false
    support_oss_cluster_api=false
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 1000
    modules = []
  }
}

resource "rediscloud_subscription_peering" "test" {
  subscription_id = rediscloud_subscription.example.id
  provider_name = "GCP"
  gcp_project_id = "%s"
  gcp_network_name = "%s"
}
`
