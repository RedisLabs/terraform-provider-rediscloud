package provider

import (
	"context"
	"flag"
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var activeActiveContractFlag = flag.Bool("activeActiveContract", false,
	"Add this flag '-activeActiveContract' to run tests for contract associated accounts")

var activeActiveMarketplaceFlag = flag.Bool("activeActiveMarketplace", false,
	"Add this flag '-activeActiveMarketplace' to run tests for marketplace associated accounts")

// Checks CRUDI (CREATE,READ,UPDATE,IMPORT) operations on the subscription resource.
func TestAccResourceRedisCloudActiveActiveSubscription_CRUDI(t *testing.T) {

	name := acctest.RandomWithPrefix(testResourcePrefix)
	resourceName := "rediscloud_active_active_subscription.example"
	datasourceName := "data.rediscloud_active_active_subscription.example"

	var subId int

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudActiveActiveSubscription, name),
				Check: resource.ComposeTestCheckFunc(
					// Test the resource
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "payment_method", "credit-card"),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider", "AWS"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.memory_limit_in_gb", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.quantity", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.modules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.modules.0", "RedisJSON"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.0.write_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.0.read_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.1.write_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.1.read_operations_per_second", "1000"),

					resource.TestCheckResourceAttr(resourceName, "pricing.#", "2"),

					resource.TestCheckResourceAttr(resourceName, "pricing.0.type", "MinimumPrice"),
					resource.TestCheckResourceAttr(resourceName, "pricing.0.quantity", "1"),
					resource.TestCheckResourceAttr(resourceName, "pricing.0.quantity_measurement", "subscription"),
					resource.TestCheckResourceAttrSet(resourceName, "pricing.0.price_per_unit"),
					resource.TestCheckResourceAttr(resourceName, "pricing.0.price_currency", "USD"),
					resource.TestCheckResourceAttr(resourceName, "pricing.0.price_period", "hour"),

					resource.TestCheckResourceAttr(resourceName, "pricing.1.type", "MinimumPrice"),
					resource.TestCheckResourceAttr(resourceName, "pricing.1.quantity", "1"),
					resource.TestCheckResourceAttr(resourceName, "pricing.1.quantity_measurement", "subscription"),
					resource.TestCheckResourceAttrSet(resourceName, "pricing.1.price_per_unit"),
					resource.TestCheckResourceAttr(resourceName, "pricing.1.price_currency", "USD"),
					resource.TestCheckResourceAttr(resourceName, "pricing.1.price_period", "hour"),

					func(s *terraform.State) error {
						r := s.RootModule().Resources[resourceName].Primary
						region1 := r.Attributes["pricing.0.region"]
						region2 := r.Attributes["pricing.1.region"]

						match := (region1 == "us-east-1" && region2 == "us-east-2") ||
							(region2 == "us-east-1" && region1 == "us-east-2")

						if !match {
							return fmt.Errorf("regions within pricing response are incorrect. expected us-east-1 and us-east-2")
						}
						return nil
					},

					func(s *terraform.State) error {
						r := s.RootModule().Resources[resourceName]

						var err error
						subId, err = strconv.Atoi(r.Primary.ID)
						if err != nil {
							return err
						}

						client := testProvider.Meta().(*apiClient)
						sub, err := client.client.Subscription.Get(context.TODO(), subId)
						if err != nil {
							return err
						}

						if redis.StringValue(sub.Name) != name {
							return fmt.Errorf("unexpected name value: %s", redis.StringValue(sub.Name))
						}
						return nil
					},

					// Test the datasource
					resource.TestCheckResourceAttr(datasourceName, "name", name),
					resource.TestCheckResourceAttr(datasourceName, "payment_method", "credit-card"),
					resource.TestCheckResourceAttrSet(datasourceName, "payment_method_id"),
					resource.TestCheckResourceAttr(datasourceName, "cloud_provider", "AWS"),
					resource.TestCheckResourceAttr(datasourceName, "number_of_databases", "1"),
					resource.TestCheckResourceAttr(datasourceName, "status", "active"),

					resource.TestCheckResourceAttr(datasourceName, "pricing.#", "2"),

					resource.TestCheckResourceAttr(datasourceName, "pricing.0.type", "MinimumPrice"),
					resource.TestCheckResourceAttr(datasourceName, "pricing.0.quantity", "1"),
					resource.TestCheckResourceAttr(datasourceName, "pricing.0.quantity_measurement", "subscription"),
					resource.TestCheckResourceAttrSet(datasourceName, "pricing.0.price_per_unit"),
					resource.TestCheckResourceAttr(datasourceName, "pricing.0.price_currency", "USD"),
					resource.TestCheckResourceAttr(datasourceName, "pricing.0.price_period", "hour"),

					resource.TestCheckResourceAttr(datasourceName, "pricing.1.type", "MinimumPrice"),
					resource.TestCheckResourceAttr(datasourceName, "pricing.1.quantity", "1"),
					resource.TestCheckResourceAttr(datasourceName, "pricing.1.quantity_measurement", "subscription"),
					resource.TestCheckResourceAttrSet(datasourceName, "pricing.1.price_per_unit"),
					resource.TestCheckResourceAttr(datasourceName, "pricing.1.price_currency", "USD"),
					resource.TestCheckResourceAttr(datasourceName, "pricing.1.price_period", "hour"),
				),
			},
			{
				// Checks if the changes in the creation plan are ignored.
				Config: fmt.Sprintf(testAccResourceRedisCloudActiveActiveSubscriptionNoCreationPlan, name, "AWS"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider", "AWS"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.memory_limit_in_gb", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.quantity", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.modules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.modules.0", "RedisJSON"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.0.write_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.0.read_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.1.write_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.1.read_operations_per_second", "1000"),
				),
			},
			{
				// Checks if the changes to the payment_method are ignored.
				Config: fmt.Sprintf(testAccResourceRedisCloudActiveActiveSubscriptionChangedPaymentMethod, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "payment_method", "credit-card"),
				),
			},
			{
				// Checks if the payment_method and creation_plan block are ignored after the IMPORT operation.
				ResourceName: resourceName,
				ImportState:  true,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					paymentMethod, ok := states[0].Attributes["payment_method"]
					if ok && paymentMethod != "credit-card" {
						return fmt.Errorf("Unexpected payment_method block. Should be 'credit-card', instead of  %s", paymentMethod)
					}
					creationPlan, ok := states[0].Attributes["creation_plan.#"]
					if ok && creationPlan != "0" {
						return fmt.Errorf("Unexpected creation_plan block. Should be 0, instead of  %s", creationPlan)
					}
					return nil
				},
			},
			{
				// Checks if an error is raised when a ForceNew attribute is changed and the creation_plan block is not defined.
				Config:       fmt.Sprintf(testAccResourceRedisCloudActiveActiveSubscriptionNoCreationPlan, name, "GCP"),
				ResourceName: resourceName,
				ExpectError:  regexp.MustCompile(`Error: the "creation_plan" block is required`),
			},
		},
	})
}

func TestAccResourceRedisCloudActiveActiveSubscription_createUpdateContractPayment(t *testing.T) {

	if !*activeActiveContractFlag {
		t.Skip("The '-activeActiveContract' parameter wasn't provided in the test command.")
	}

	name := acctest.RandomWithPrefix(testResourcePrefix)
	updatedName := fmt.Sprintf("%v-updatedName", name)
	resourceName := "rediscloud_active_active_subscription.example"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudActiveActiveSubscriptionContractPayment, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.provider", "AWS"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.0.write_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.0.read_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.1.write_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.1.read_operations_per_second", "1000"),
				),
			},
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudActiveActiveSubscriptionContractPayment, updatedName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
					resource.TestCheckResourceAttr(resourceName, "name", updatedName),
				),
			},
		},
	})
}

func TestAccResourceRedisCloudActiveActiveSubscription_createUpdateMarketplacePayment(t *testing.T) {

	if !*activeActiveMarketplaceFlag {
		t.Skip("The '-activeActiveMarketplace' parameter wasn't provided in the test command.")
	}

	name := acctest.RandomWithPrefix(testResourcePrefix)
	updatedName := fmt.Sprintf("%v-updatedName", name)
	resourceName := "rediscloud_active_active_subscription.example"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudActiveActiveSubscriptionMarketplacePayment, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.provider", "AWS"),
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.0.write_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.0.read_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.1.write_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.1.read_operations_per_second", "1000"),
				),
			},
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudActiveActiveSubscriptionMarketplacePayment, updatedName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", updatedName),
				),
			},
		},
	})
}

func testAccCheckActiveActiveSubscriptionDestroy(s *terraform.State) error {
	client := testProvider.Meta().(*apiClient)

	for _, r := range s.RootModule().Resources {
		if r.Type != "rediscloud_active_active_subscription" {
			continue
		}

		subId, err := strconv.Atoi(r.Primary.ID)
		if err != nil {
			return err
		}

		subs, err := client.client.Subscription.List(context.TODO())
		if err != nil {
			return err
		}

		for _, sub := range subs {
			if redis.IntValue(sub.ID) == subId {
				return fmt.Errorf("subscription %d still exists", subId)
			}
		}
	}

	return nil
}

// TF config for provisioning a new subscription.
const testAccResourceRedisCloudActiveActiveSubscription = `
data "rediscloud_payment_method" "card" {
	card_type = "Visa"
}

resource "rediscloud_active_active_subscription" "example" {
	name = "%s"
	payment_method_id = data.rediscloud_payment_method.card.id
	cloud_provider = "AWS"

	creation_plan {
		memory_limit_in_gb = 1
		modules = ["RedisJSON"]
		quantity = 1
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

data "rediscloud_active_active_subscription" "example" {
	name = rediscloud_active_active_subscription.example.name
}
`

const testAccResourceRedisCloudActiveActiveSubscriptionNoCreationPlan = `
  
  data "rediscloud_payment_method" "card" {
	card_type = "Visa"
  }
  
  resource "rediscloud_active_active_subscription" "example" {
	name = "%s"
	payment_method_id = data.rediscloud_payment_method.card.id
	cloud_provider = "%s"
   
  }
`

const testAccResourceRedisCloudActiveActiveSubscriptionChangedPaymentMethod = `
data "rediscloud_payment_method" "card" {
	card_type = "Visa"
}

resource "rediscloud_active_active_subscription" "example" {
	name = "%s"
    payment_method = "marketplace"
	payment_method_id = data.rediscloud_payment_method.card.id
	cloud_provider = "AWS"

	creation_plan {
		memory_limit_in_gb = 1
		quantity = 1
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
`

const testAccResourceRedisCloudActiveActiveSubscriptionContractPayment = `

resource "rediscloud_active_active_subscription" "example" {
  	name = "%s"
	cloud_provider = "AWS"
  
	creation_plan {
	  	memory_limit_in_gb = 2
	  	quantity = 1
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
`

const testAccResourceRedisCloudActiveActiveSubscriptionMarketplacePayment = `

resource "rediscloud_active_active_subscription" "example" {
  name = "%s"
  payment_method = "marketplace"

  cloud_provider = "AWS"
  creation_plan {
    memory_limit_in_gb = 2
    quantity = 1
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
`
