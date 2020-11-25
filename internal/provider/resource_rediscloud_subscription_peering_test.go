package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"net"
	"os"
	"regexp"
	"testing"
)

func TestAccResourceRedisCloudSubscriptionPeering_basic(t *testing.T) {
	name := acctest.RandomWithPrefix(testResourcePrefix)
	password := acctest.RandString(20)

	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	cidrRange := os.Getenv("AWS_VPC_CIDR")
	// Chose a CIDR range for the subscription that's unlikely to overlap with any VPC CIDR
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

	tf := fmt.Sprintf(testAccResourceRedisCloudSubscriptionPeering,
		testCloudAccountName,
		name,
		subCidrRange,
		password,
		peeringRegion,
		accountId,
		vpcId,
		cidrRange,
	)
	resourceName := "rediscloud_subscription_peering.test"

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
				),
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

const testAccResourceRedisCloudSubscriptionPeering = `
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
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
  persistent_storage_encryption = false

  cloud_provider {
    provider = data.rediscloud_cloud_account.account.provider_type
    cloud_account_id = data.rediscloud_cloud_account.account.id
    region {
      region = "eu-west-1"
      networking_deployment_cidr = "%s"
      preferred_availability_zones = ["eu-west-1a"]
    }
  }

  database {
    name = "tf-database"
    protocol = "redis"
    memory_limit_in_gb = 1
    data_persistence = "none"
    throughput_measurement_by = "operations-per-second"
    password = "%s"
    throughput_measurement_value = 10000
  }
}

resource "rediscloud_subscription_peering" "test" {
  subscription_id = rediscloud_subscription.example.id
  region = "%s"
  aws_account_id = "%s"
  vpc_id = "%s"
  vpc_cidr = "%s"
}
`
