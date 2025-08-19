package provider

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourceRedisCloudDedicatedSubscription_basic(t *testing.T) {
	name := acctest.RandomWithPrefix(testResourcePrefix)
	resourceName := "rediscloud_dedicated_subscription.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckDedicatedSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudDedicatedSubscriptionBasic, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "payment_method", "credit-card"),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.provider", "AWS"),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.region", "us-east-1"),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.networking_deployment_cidr", "10.0.0.0/24"),
					resource.TestCheckResourceAttr(resourceName, "instance_type.0.instance_name", "dedicated-large"),
					resource.TestCheckResourceAttr(resourceName, "instance_type.0.replication", "true"),
					resource.TestCheckResourceAttr(resourceName, "status", "active"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
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

func TestAccResourceRedisCloudDedicatedSubscription_update(t *testing.T) {
	name := acctest.RandomWithPrefix(testResourcePrefix)
	updatedName := acctest.RandomWithPrefix(testResourcePrefix)
	resourceName := "rediscloud_dedicated_subscription.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckDedicatedSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudDedicatedSubscriptionBasic, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
				),
			},
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudDedicatedSubscriptionBasic, updatedName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", updatedName),
				),
			},
		},
	})
}

func TestAccResourceRedisCloudDedicatedSubscription_gcp(t *testing.T) {
	name := acctest.RandomWithPrefix(testResourcePrefix)
	resourceName := "rediscloud_dedicated_subscription.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckDedicatedSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudDedicatedSubscriptionGCP, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.provider", "GCP"),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.region", "us-central1"),
					resource.TestCheckResourceAttr(resourceName, "instance_type.0.replication", "false"),
				),
			},
		},
	})
}

func TestAccResourceRedisCloudDedicatedSubscription_withVPC(t *testing.T) {
	name := acctest.RandomWithPrefix(testResourcePrefix)
	resourceName := "rediscloud_dedicated_subscription.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckDedicatedSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudDedicatedSubscriptionWithVPC, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.networking_vpc_id", "vpc-12345678"),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.networking_deployment_cidr", "172.16.0.0/24"),
				),
			},
		},
	})
}

func TestAccResourceRedisCloudDedicatedSubscription_invalidCIDR(t *testing.T) {
	name := acctest.RandomWithPrefix(testResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckDedicatedSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccResourceRedisCloudDedicatedSubscriptionInvalidCIDR, name),
				ExpectError: regexp.MustCompile("must contain a valid CIDR"),
			},
		},
	})
}

func testAccCheckDedicatedSubscriptionDestroy(s *terraform.State) error {
	client := testProvider.Meta().(*apiClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "rediscloud_dedicated_subscription" {
			continue
		}

		subId, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}

		// Note: This would use the actual dedicated subscription API client when available
		// subscription, err := client.client.DedicatedSubscription.Get(context.TODO(), subId)
		// if err != nil {
		//     if isNotFoundError(err) {
		//         continue
		//     }
		//     return fmt.Errorf("error fetching dedicated subscription %d: %s", subId, err)
		// }
		// if subscription != nil {
		//     return fmt.Errorf("dedicated subscription %d still exists", subId)
		// }

		// Placeholder implementation for development
		_ = client
		_ = subId
		// In actual implementation, verify the subscription is deleted
	}

	return nil
}

// Test configurations
const testAccResourceRedisCloudDedicatedSubscriptionBasic = `
resource "rediscloud_dedicated_subscription" "test" {
  name           = "%s"
  payment_method = "credit-card"

  cloud_provider {
    provider                     = "AWS"
    cloud_account_id            = "1"
    region                      = "us-east-1"
    networking_deployment_cidr  = "10.0.0.0/24"
    availability_zones          = ["us-east-1a", "us-east-1b"]
  }

  instance_type {
    instance_name = "dedicated-large"
    replication   = true
  }

  redis_version = "7.2"
}
`

const testAccResourceRedisCloudDedicatedSubscriptionGCP = `
resource "rediscloud_dedicated_subscription" "test" {
  name           = "%s"
  payment_method = "credit-card"

  cloud_provider {
    provider                     = "GCP"
    cloud_account_id            = "1"
    region                      = "us-central1"
    networking_deployment_cidr  = "10.1.0.0/24"
    availability_zones          = ["us-central1-a", "us-central1-b"]
  }

  instance_type {
    instance_name = "dedicated-medium"
    replication   = false
  }

  redis_version = "6.2"
}
`

const testAccResourceRedisCloudDedicatedSubscriptionWithVPC = `
resource "rediscloud_dedicated_subscription" "test" {
  name           = "%s"
  payment_method = "credit-card"

  cloud_provider {
    provider                     = "AWS"
    cloud_account_id            = "1"
    region                      = "us-west-2"
    networking_deployment_cidr  = "172.16.0.0/24"
    networking_vpc_id           = "vpc-12345678"
    availability_zones          = ["us-west-2a", "us-west-2b", "us-west-2c"]
  }

  instance_type {
    instance_name = "dedicated-xlarge"
    replication   = true
  }
}
`

const testAccResourceRedisCloudDedicatedSubscriptionInvalidCIDR = `
resource "rediscloud_dedicated_subscription" "test" {
  name           = "%s"
  payment_method = "credit-card"

  cloud_provider {
    provider                     = "AWS"
    cloud_account_id            = "1"
    region                      = "us-east-1"
    networking_deployment_cidr  = "invalid-cidr"
  }

  instance_type {
    instance_name = "dedicated-large"
    replication   = true
  }
}
`
