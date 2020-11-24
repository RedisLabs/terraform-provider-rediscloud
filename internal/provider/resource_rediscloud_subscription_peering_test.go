package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestAccResourceRedisCloudSubscriptionPeering_basic(t *testing.T) {
	t.Skip("Required environment variables currently not available under CI")

	name := acctest.RandomWithPrefix(testResourcePrefix)
	password := acctest.RandString(20)

	cidrRange := os.Getenv("AWS_VPC_CIDR")

	if strings.HasPrefix(cidrRange, "192.168") {
		t.Fatal("VPC peering test has the subscription deployment CIDR using 192.168.x.x, so the peered VPC must be something else")
	}

	tf := fmt.Sprintf(testAccResourceRedisCloudSubscriptionPeering,
		name,
		password,
		os.Getenv("AWS_PEERING_REGION"),
		os.Getenv("AWS_ACCOUNT_ID"),
		os.Getenv("AWS_VPC_ID"),
		cidrRange,
	)
	resourceName := "rediscloud_subscription_peering.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPeeringPreCheck(t) },
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

const testAccResourceRedisCloudSubscriptionPeering = `
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type = "AWS" 
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
      networking_deployment_cidr = "192.168.0.0/24"
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
