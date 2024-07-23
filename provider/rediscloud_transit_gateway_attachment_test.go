package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"os"
	"regexp"
	"testing"
)

func TestAccResourceRedisCloudTransitGatewayAttachment_Pro(t *testing.T) {

	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")
	testTgwId := os.Getenv("AWS_TEST_TGW_ID")
	baseName := acctest.RandomWithPrefix(testResourcePrefix) + "-pro-tgwa"

	resourceName := "rediscloud_transit_gateway_attachment.test"
	datasourceName := "data.rediscloud_transit_gateway.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingTgwCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudTransitGatewayPro, testCloudAccountName, baseName, testTgwId),
				Check: resource.ComposeTestCheckFunc(
					// Test the datasource before attachment
					resource.TestCheckResourceAttrSet(datasourceName, "id"),
					resource.TestCheckResourceAttr(datasourceName, "aws_tgw_uid", testTgwId),
					resource.TestCheckResourceAttr(datasourceName, "attachment_uid", ""),
					resource.TestCheckResourceAttr(datasourceName, "status", "available"),
					resource.TestCheckResourceAttr(datasourceName, "attachment_status", ""),
					resource.TestCheckResourceAttrSet(datasourceName, "aws_account_id"),
					resource.TestCheckResourceAttr(datasourceName, "cidrs.#", "0"),
				),
			},
			{
				Config:      fmt.Sprintf(testAccResourceRedisCloudTransitGatewayAttachmentProWithCidrs, testCloudAccountName, baseName, testTgwId),
				ExpectError: regexp.MustCompile("Attachment cannot be created with Cidrs provided, it must be accepted first\\. This resource may then be updated with Cidrs\\."),
			},
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudTransitGatewayAttachmentPro, testCloudAccountName, baseName, testTgwId),
				Check: resource.ComposeTestCheckFunc(
					// Test the resource
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "aws_tgw_uid", testTgwId),
					resource.TestCheckResourceAttrSet(resourceName, "attachment_uid"),
					resource.TestCheckResourceAttr(resourceName, "status", "available"),
					resource.TestCheckResourceAttr(resourceName, "attachment_status", "pending-acceptance"),
					resource.TestCheckResourceAttrSet(resourceName, "aws_account_id"),
					resource.TestCheckResourceAttr(resourceName, "cidrs.#", "0"),
				),
			},
			{
				Config:      fmt.Sprintf(testAccResourceRedisCloudTransitGatewayAttachmentProWithCidrs, testCloudAccountName, baseName, testTgwId),
				ExpectError: regexp.MustCompile("Transit Gateway attachment is not active"),
			},
		},
	})
}

func TestAccResourceRedisCloudTransitGatewayAttachment_ActiveActive(t *testing.T) {

	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")
	testTgwId := os.Getenv("AWS_TEST_TGW_ID")
	baseName := acctest.RandomWithPrefix(testResourcePrefix) + "-aa-tgwa"

	subName := baseName + "-sub"
	dbName := baseName + "-db"

	resourceName := "rediscloud_active_active_transit_gateway_attachment.test"
	datasourceName := "data.rediscloud_active_active_transit_gateway.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingTgwCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudTransitGatewayActiveActive, testCloudAccountName, subName, dbName, testTgwId),
				Check: resource.ComposeTestCheckFunc(
					// Test the datasource before attachment
					resource.TestCheckResourceAttrSet(datasourceName, "id"),
					resource.TestCheckResourceAttr(datasourceName, "aws_tgw_uid", testTgwId),
					resource.TestCheckResourceAttr(datasourceName, "attachment_uid", ""),
					resource.TestCheckResourceAttr(datasourceName, "status", "available"),
					resource.TestCheckResourceAttr(datasourceName, "attachment_status", ""),
					resource.TestCheckResourceAttrSet(datasourceName, "aws_account_id"),
					resource.TestCheckResourceAttr(datasourceName, "cidrs.#", "0"),
				),
			},
			{
				Config:      fmt.Sprintf(testAccResourceRedisCloudTransitGatewayAttachmentActiveActiveWithCidrs, testCloudAccountName, subName, dbName, testTgwId),
				ExpectError: regexp.MustCompile("Attachment cannot be created with Cidrs provided, it must be accepted first\\. This resource may then be updated with Cidrs\\."),
			},
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudTransitGatewayAttachmentActiveActive, testCloudAccountName, subName, dbName, testTgwId),
				Check: resource.ComposeTestCheckFunc(
					// Test the resource
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "aws_tgw_uid", testTgwId),
					resource.TestCheckResourceAttrSet(resourceName, "attachment_uid"),
					resource.TestCheckResourceAttr(resourceName, "status", "available"),
					resource.TestCheckResourceAttr(resourceName, "attachment_status", "pending-acceptance"),
					resource.TestCheckResourceAttrSet(resourceName, "aws_account_id"),
					resource.TestCheckResourceAttr(resourceName, "cidrs.#", "0"),
				),
			},
			{
				Config:      fmt.Sprintf(testAccResourceRedisCloudTransitGatewayAttachmentActiveActiveWithCidrs, testCloudAccountName, subName, dbName, testTgwId),
				ExpectError: regexp.MustCompile("Transit Gateway attachment is not active"),
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

data "rediscloud_transit_gateway" "test" {
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

data "rediscloud_transit_gateway" "test" {
	subscription_id = rediscloud_subscription.example.id
	aws_tgw_uid = "%s"
}

resource "rediscloud_transit_gateway_attachment" "test" {
	subscription_id = rediscloud_subscription.example.id
	tgw_id = data.rediscloud_transit_gateway.test.tgw_id
}
`

const testAccResourceRedisCloudTransitGatewayAttachmentProWithCidrs = `
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

data "rediscloud_transit_gateway" "test" {
	subscription_id = rediscloud_subscription.example.id
	aws_tgw_uid = "%s"
}

resource "rediscloud_transit_gateway_attachment" "test" {
	subscription_id = rediscloud_subscription.example.id
	tgw_id = data.rediscloud_transit_gateway.test.tgw_id
	cidrs = ["10.10.20.0/24"]
}
`

const testAccResourceRedisCloudTransitGatewayActiveActive = `
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type = "AWS"
  name = "%s"
}

resource "rediscloud_active_active_subscription" "example" {
  name = "%s"
  payment_method_id = data.rediscloud_payment_method.card.id
  cloud_provider = "AWS"

  creation_plan {
    memory_limit_in_gb = 1
    quantity = 1
    region {
      region = "us-east-1"
      networking_deployment_cidr = "10.0.18.0/24"
      write_operations_per_second = 1000
      read_operations_per_second = 1000
    }
    region {
      region = "us-east-2"
      networking_deployment_cidr = "10.0.19.0/24"
      write_operations_per_second = 1000
      read_operations_per_second = 1000
    }
  }
}

resource "rediscloud_active_active_subscription_database" "example" {
  subscription_id = rediscloud_active_active_subscription.example.id
  name = "%s"
  memory_limit_in_gb = 1
  support_oss_cluster_api = false
  external_endpoint_for_oss_cluster_api = false
}

resource "rediscloud_active_active_subscription_regions" "example" {
  subscription_id = rediscloud_active_active_subscription.example.id
  delete_regions = false
  region {
    region = "us-east-1"
    networking_deployment_cidr = "10.0.18.0/24"
    recreate_region = false
    database {
      database_id = rediscloud_active_active_subscription_database.example.db_id
      database_name = rediscloud_active_active_subscription_database.example.name
      local_write_operations_per_second = 1000
      local_read_operations_per_second = 1000
    }
  }
  region {
    region = "us-east-2"
    networking_deployment_cidr = "10.0.19.0/24"
    recreate_region = false
    database {
      database_id = rediscloud_active_active_subscription_database.example.db_id
      database_name = rediscloud_active_active_subscription_database.example.name
      local_write_operations_per_second = 1000
      local_read_operations_per_second = 1000
    }
  }
}

data "rediscloud_active_active_transit_gateway" "test" {
	subscription_id = rediscloud_active_active_subscription.example.id
	region_id = tolist(rediscloud_active_active_subscription_regions.example.region)[0].region == "us-east-1" ? tolist(rediscloud_active_active_subscription_regions.example.region)[0].region_id : tolist(rediscloud_active_active_subscription_regions.example.region)[1].region_id
	aws_tgw_uid = "%s"
}
`

const testAccResourceRedisCloudTransitGatewayAttachmentActiveActive = `
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type = "AWS"
  name = "%s"
}

resource "rediscloud_active_active_subscription" "example" {
  name = "%s"
  payment_method_id = data.rediscloud_payment_method.card.id
  cloud_provider = "AWS"

  creation_plan {
    memory_limit_in_gb = 1
    quantity = 1
    region {
      region = "us-east-1"
      networking_deployment_cidr = "10.0.18.0/24"
      write_operations_per_second = 1000
      read_operations_per_second = 1000
    }
    region {
      region = "us-east-2"
      networking_deployment_cidr = "10.0.19.0/24"
      write_operations_per_second = 1000
      read_operations_per_second = 1000
    }
  }
}

resource "rediscloud_active_active_subscription_database" "example" {
  subscription_id = rediscloud_active_active_subscription.example.id
  name = "%s"
  memory_limit_in_gb = 1
  support_oss_cluster_api = false
  external_endpoint_for_oss_cluster_api = false
}

resource "rediscloud_active_active_subscription_regions" "example" {
  subscription_id = rediscloud_active_active_subscription.example.id
  delete_regions = false
  region {
    region = "us-east-1"
    networking_deployment_cidr = "10.0.18.0/24"
    recreate_region = false
    database {
      database_id = rediscloud_active_active_subscription_database.example.db_id
      database_name = rediscloud_active_active_subscription_database.example.name
      local_write_operations_per_second = 1000
      local_read_operations_per_second = 1000
    }
  }
  region {
    region = "us-east-2"
    networking_deployment_cidr = "10.0.19.0/24"
    recreate_region = false
    database {
      database_id = rediscloud_active_active_subscription_database.example.db_id
      database_name = rediscloud_active_active_subscription_database.example.name
      local_write_operations_per_second = 1000
      local_read_operations_per_second = 1000
    }
  }
}

data "rediscloud_active_active_transit_gateway" "test" {
	subscription_id = rediscloud_active_active_subscription.example.id
	region_id = tolist(rediscloud_active_active_subscription_regions.example.region)[0].region == "us-east-1" ? tolist(rediscloud_active_active_subscription_regions.example.region)[0].region_id : tolist(rediscloud_active_active_subscription_regions.example.region)[1].region_id
	aws_tgw_uid = "%s"
}

resource "rediscloud_active_active_transit_gateway_attachment" "test" {
	subscription_id = rediscloud_active_active_subscription.example.id
	region_id = tolist(rediscloud_active_active_subscription_regions.example.region)[0].region == "us-east-1" ? tolist(rediscloud_active_active_subscription_regions.example.region)[0].region_id : tolist(rediscloud_active_active_subscription_regions.example.region)[1].region_id
	tgw_id = data.rediscloud_active_active_transit_gateway.test.tgw_id
}
`

const testAccResourceRedisCloudTransitGatewayAttachmentActiveActiveWithCidrs = `
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type = "AWS"
  name = "%s"
}

resource "rediscloud_active_active_subscription" "example" {
  name = "%s"
  payment_method_id = data.rediscloud_payment_method.card.id
  cloud_provider = "AWS"

  creation_plan {
    memory_limit_in_gb = 1
    quantity = 1
    region {
      region = "us-east-1"
      networking_deployment_cidr = "10.0.18.0/24"
      write_operations_per_second = 1000
      read_operations_per_second = 1000
    }
    region {
      region = "us-east-2"
      networking_deployment_cidr = "10.0.19.0/24"
      write_operations_per_second = 1000
      read_operations_per_second = 1000
    }
  }
}

resource "rediscloud_active_active_subscription_database" "example" {
  subscription_id = rediscloud_active_active_subscription.example.id
  name = "%s"
  memory_limit_in_gb = 1
  support_oss_cluster_api = false
  external_endpoint_for_oss_cluster_api = false
}

resource "rediscloud_active_active_subscription_regions" "example" {
  subscription_id = rediscloud_active_active_subscription.example.id
  delete_regions = false
  region {
    region = "us-east-1"
    networking_deployment_cidr = "10.0.18.0/24"
    recreate_region = false
    database {
      database_id = rediscloud_active_active_subscription_database.example.db_id
      database_name = rediscloud_active_active_subscription_database.example.name
      local_write_operations_per_second = 1000
      local_read_operations_per_second = 1000
    }
  }
  region {
    region = "us-east-2"
    networking_deployment_cidr = "10.0.19.0/24"
    recreate_region = false
    database {
      database_id = rediscloud_active_active_subscription_database.example.db_id
      database_name = rediscloud_active_active_subscription_database.example.name
      local_write_operations_per_second = 1000
      local_read_operations_per_second = 1000
    }
  }
}

data "rediscloud_active_active_transit_gateway" "test" {
	subscription_id = rediscloud_subscription.example.id
	region_id = tolist(rediscloud_active_active_subscription_regions.example.region)[0].region == "us-east-1" ? tolist(rediscloud_active_active_subscription_regions.example.region)[0].region_id : tolist(rediscloud_active_active_subscription_regions.example.region)[1].region_id
	aws_tgw_uid = "%s"
}

resource "rediscloud_active_active_transit_gateway_attachment" "test" {
	subscription_id = rediscloud_subscription.example.id
	region_id = tolist(rediscloud_active_active_subscription_regions.example.region)[0].region == "us-east-1" ? tolist(rediscloud_active_active_subscription_regions.example.region)[0].region_id : tolist(rediscloud_active_active_subscription_regions.example.region)[1].region_id
	tgw_id = data.rediscloud_active_active_transit_gateway.test.tgw_id
	cidrs = ["10.10.20.0/24"]
}
`
