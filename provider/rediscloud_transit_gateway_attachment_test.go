package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"os"
	"testing"
)

func TestAccResourceRedisCloudTransitGatewayAttachment_Pro(t *testing.T) {

	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")
	testTgwId := "tgw-0b92afdae97faaef8" // TODO Get from env
	baseName := acctest.RandomWithPrefix(testResourcePrefix) + "pro-tgwa"

	resourceName := "rediscloud_transit_gateway_attachment.test"
	datasourceName := "data.rediscloud_transit_gateway_attachment.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudTransitGatewayPro, testCloudAccountName, baseName, testTgwId),
				Check: resource.ComposeTestCheckFunc(
					// Test the datasource before attachment
					resource.TestCheckResourceAttrSet(datasourceName, "id"),
					resource.TestCheckResourceAttr(datasourceName, "aws_tgw_uid", "tgw-0b92afdae97faaef8"),
					resource.TestCheckResourceAttr(datasourceName, "attachment_uid", ""),
					resource.TestCheckResourceAttr(datasourceName, "status", "available"),
					resource.TestCheckResourceAttr(datasourceName, "attachment_status", ""),
					resource.TestCheckResourceAttr(datasourceName, "aws_account_id", "620187402834"),
					resource.TestCheckResourceAttr(datasourceName, "cidrs.#", "0"),
				),
			},
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudTransitGatewayAttachmentPro, testCloudAccountName, baseName, testTgwId),
				Check: resource.ComposeTestCheckFunc(
					// Test the resource
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "aws_tgw_uid", "tgw-0b92afdae97faaef8"),
					resource.TestCheckResourceAttr(resourceName, "attachment_uid", ""),
					resource.TestCheckResourceAttr(resourceName, "status", "available"),
					resource.TestCheckResourceAttr(resourceName, "attachment_status", ""), // TODO Pending?
					resource.TestCheckResourceAttr(resourceName, "aws_account_id", "620187402834"),
					resource.TestCheckResourceAttr(resourceName, "cidrs.#", "0"),

					// And check for datasource changes
					resource.TestCheckResourceAttrSet(datasourceName, "id"),
					resource.TestCheckResourceAttr(datasourceName, "aws_tgw_uid", "tgw-0b92afdae97faaef8"),
					resource.TestCheckResourceAttr(datasourceName, "attachment_uid", ""),
					resource.TestCheckResourceAttr(datasourceName, "status", "available"),
					resource.TestCheckResourceAttr(datasourceName, "attachment_status", ""), // TODO Pending?
					resource.TestCheckResourceAttr(datasourceName, "aws_account_id", "620187402834"),
					resource.TestCheckResourceAttr(datasourceName, "cidrs.#", "0"),
				),
			},
		},
	})
}

const testAccResourceRedisCloudTransitGatewayPro = `
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
  payment_method = "credit-card"
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage = "ram"

  allowlist {
    cidrs = ["192.168.0.0/16"]
    security_group_ids = []
  }

  cloud_provider {
    provider = data.rediscloud_cloud_account.account.provider_type
    cloud_account_id = data.rediscloud_cloud_account.account.id
    region {
      region = "us-east-1"
      networking_deployment_cidr = "10.0.0.0/24"
      preferred_availability_zones = ["us-east-1a"]
    }
  }

  creation_plan {
    memory_limit_in_gb = 1
    quantity = 1
    replication=false
    support_oss_cluster_api=false
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 10000
    modules = []
  }
}

data "rediscloud_transit_gateway_attachment" "test" {
	subscription_id = rediscloud_subscription.example.id
	aws_tgw_uid = "%s"
}
`

const testAccResourceRedisCloudTransitGatewayAttachmentPro = `
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
  payment_method = "credit-card"
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage = "ram"

  allowlist {
    cidrs = ["192.168.0.0/16"]
    security_group_ids = []
  }

  cloud_provider {
    provider = data.rediscloud_cloud_account.account.provider_type
    cloud_account_id = data.rediscloud_cloud_account.account.id
    region {
      region = "us-east-1"
      networking_deployment_cidr = "10.0.0.0/24"
      preferred_availability_zones = ["us-east-1a"]
    }
  }

  creation_plan {
    memory_limit_in_gb = 1
    quantity = 1
    replication=false
    support_oss_cluster_api=false
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 10000
    modules = []
  }
}

data "rediscloud_transit_gateway_attachment" "test" {
	subscription_id = rediscloud_subscription.example.id
	aws_tgw_uid = "%s"
}

resource "rediscloud_transit_gateway_attachment" "test" {
	subscription_id = rediscloud_subscription.example.id
	tgw_id = data.rediscloud_transit_gateway_attachment.test.tgw_id
}
`
