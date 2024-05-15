package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceRedisCloudSubscriptionPeerings_basic(t *testing.T) {
	name := acctest.RandomWithPrefix(testResourcePrefix)

	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	// Chose a CIDR range for the subscription that's unlikely to overlap with any VPC CIDR
	subCidrRange := "10.0.0.0/24"

	awsAccountId := os.Getenv("AWS_ACCOUNT_ID")
	awsVPCId := os.Getenv("AWS_VPC_ID")
	awsVPCCidr := os.Getenv("AWS_VPC_CIDR")
	awsRegion := os.Getenv("AWS_PEERING_REGION")
	tf := fmt.Sprintf(testAccDatasourceRedisCloudSubscriptionPeeringsDataSource,
		testCloudAccountName,
		name,
		subCidrRange,
		awsRegion,
		awsAccountId,
		awsVPCId,
		awsVPCCidr,
	)
	dataSourceName := "data.rediscloud_subscription_peerings.example"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPeeringPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckFlexibleSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: tf,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchTypeSetElemNestedAttrs(dataSourceName, "peerings.*", map[string]*regexp.Regexp{
						"provider_name":  regexp.MustCompile("AWS"),
						"aws_account_id": regexp.MustCompile(awsAccountId),
						"vpc_id":         regexp.MustCompile(awsVPCId),
						"vpc_cidr":       regexp.MustCompile(awsVPCCidr),
						"aws_peering_id": regexp.MustCompile("^pcx-"),
						"region":         regexp.MustCompile(awsRegion),
					}),
				),
			},
		},
	})
}

const testAccDatasourceRedisCloudSubscriptionPeeringsDataSource = `
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type = "AWS" 
  name = "%s"
}

resource "rediscloud_flexible_subscription" "example" {
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
    memory_limit_in_gb           = 1
    quantity                     = 1
    replication                  = false
    support_oss_cluster_api      = false
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 1000
    modules = []
  }
}

resource "rediscloud_subscription_peering" "test" {
  subscription_id = rediscloud_flexible_subscription.example.id
  region = "%s"
  aws_account_id = "%s"
  vpc_id = "%s"
  vpc_cidr = "%s"
}

data "rediscloud_subscription_peerings" "example" {
  subscription_id = rediscloud_subscription_peering.test.subscription_id
}
`
