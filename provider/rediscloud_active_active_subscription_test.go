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

// Checks CRUDI (CREATE, READ, UPDATE, IMPORT) operations on the subscription resource.
// Also checks active-active subscription regions.
func TestAccResourceRedisCloudActiveActiveSubscription_CRUDI(t *testing.T) {

	testAccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := acctest.RandomWithPrefix(testResourcePrefix)
	const resourceName = "rediscloud_active_active_subscription.example"
	const datasourceSubscriptionName = "data.rediscloud_active_active_subscription.example"
	const datasourceRegionName = "data.rediscloud_active_active_subscription_regions.example"

	var subId int

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudActiveActiveSubscription, name, name),
				Check: resource.ComposeAggregateTestCheckFunc(
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

					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.mode", "manual"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.#", "2"),

					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.0.start_hour", "22"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.0.duration_in_hours", "8"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.0.days.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.0.days.0", "Monday"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.0.days.1", "Thursday"),

					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.1.start_hour", "12"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.1.duration_in_hours", "6"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.1.days.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.1.days.0", "Friday"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.1.days.1", "Saturday"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.1.days.2", "Sunday"),

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

					// Test the subscription datasource
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "name", name),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "payment_method", "credit-card"),
					resource.TestCheckResourceAttrSet(datasourceSubscriptionName, "payment_method_id"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "cloud_provider", "AWS"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "number_of_databases", "0"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "status", "active"),

					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.mode", "manual"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.#", "2"),

					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.0.start_hour", "22"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.0.duration_in_hours", "8"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.0.days.#", "2"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.0.days.0", "Monday"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.0.days.1", "Thursday"),

					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.1.start_hour", "12"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.1.duration_in_hours", "6"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.1.days.#", "3"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.1.days.0", "Friday"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.1.days.1", "Saturday"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.1.days.2", "Sunday"),

					resource.TestCheckResourceAttr(datasourceSubscriptionName, "pricing.#", "2"),

					resource.TestCheckResourceAttr(datasourceSubscriptionName, "pricing.0.type", "MinimumPrice"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "pricing.0.quantity", "1"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "pricing.0.quantity_measurement", "subscription"),
					resource.TestCheckResourceAttrSet(datasourceSubscriptionName, "pricing.0.price_per_unit"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "pricing.0.price_currency", "USD"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "pricing.0.price_period", "hour"),

					resource.TestCheckResourceAttr(datasourceSubscriptionName, "pricing.1.type", "MinimumPrice"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "pricing.1.quantity", "1"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "pricing.1.quantity_measurement", "subscription"),
					resource.TestCheckResourceAttrSet(datasourceSubscriptionName, "pricing.1.price_per_unit"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "pricing.1.price_currency", "USD"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "pricing.1.price_period", "hour"),

					// Test the region datasource

					resource.TestCheckResourceAttr(datasourceRegionName, "subscription_name", name),
					//resource.TestCheckResourceAttrSet(datasourceRegionName, "regions.0.regionId"),
					resource.TestCheckResourceAttr(datasourceRegionName, "regions.0.region", "us-east-1"),
					resource.TestCheckResourceAttr(datasourceRegionName, "regions.0.networking_deployment_cidr", "192.168.0.0/24"),
					resource.TestCheckResourceAttrSet(datasourceRegionName, "regions.0.vpc_id"),
					//resource.TestCheckResourceAttrSet(datasourceRegionName, "regions.1.regionId"),
					resource.TestCheckResourceAttr(datasourceRegionName, "regions.1.region", "us-east-2"),
					resource.TestCheckResourceAttr(datasourceRegionName, "regions.1.networking_deployment_cidr", "10.0.1.0/24"),
					resource.TestCheckResourceAttrSet(datasourceRegionName, "regions.1.vpc_id"),
				),
			},
			{
				// Checks if the changes in the creation plan are ignored.
				Config: fmt.Sprintf(testAccResourceRedisCloudActiveActiveSubscriptionNoCreationPlan, name, "AWS"),
				Check: resource.ComposeAggregateTestCheckFunc(
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

					// maintenance windows spec is omitted so should default back to automatic
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.mode", "automatic"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.#", "0"),

					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.mode", "automatic"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.#", "0"),
				),
			},
			{
				// Checks if the changes to the payment_method are ignored.
				Config: fmt.Sprintf(testAccResourceRedisCloudActiveActiveSubscriptionChangedPaymentMethod, name),
				Check: resource.ComposeAggregateTestCheckFunc(
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

	testAccRequiresEnvVar(t, "EXECUTE_TESTS")

	if !*activeActiveContractFlag {
		t.Skip("The '-activeActiveContract' parameter wasn't provided in the test command.")
	}

	name := acctest.RandomWithPrefix(testResourcePrefix)
	updatedName := fmt.Sprintf("%v-updatedName", name)
	const resourceName = "rediscloud_active_active_subscription.example"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudActiveActiveSubscriptionContractPayment, name),
				Check: resource.ComposeAggregateTestCheckFunc(
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
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
					resource.TestCheckResourceAttr(resourceName, "name", updatedName),
				),
			},
		},
	})
}

func TestAccResourceRedisCloudActiveActiveSubscription_createUpdateMarketplacePayment(t *testing.T) {

	testAccRequiresEnvVar(t, "EXECUTE_TESTS")

	if !*activeActiveMarketplaceFlag {
		t.Skip("The '-activeActiveMarketplace' parameter wasn't provided in the test command.")
	}

	name := acctest.RandomWithPrefix(testResourcePrefix)
	updatedName := fmt.Sprintf("%v-updatedName", name)
	const resourceName = "rediscloud_active_active_subscription.example"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudActiveActiveSubscriptionMarketplacePayment, name),
				Check: resource.ComposeAggregateTestCheckFunc(
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
				Check: resource.ComposeAggregateTestCheckFunc(
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
	last_four_numbers = "5556"
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

  maintenance_windows {
    mode = "manual"
    window {
      start_hour = 22
      duration_in_hours = 8
      days = ["Monday", "Thursday"]
    }
    window {
      start_hour = 12
      duration_in_hours = 6
      days = ["Friday", "Saturday", "Sunday"]
    }
  }
}

resource "rediscloud_active_active_subscription_database" "example" {
  subscription_id = rediscloud_active_active_subscription.example.id
  name = "%s"
  dataset_size_in_gb = 1
  global_data_persistence = "aof-every-1-second"
  global_password = "some-random-pass-2"
  global_source_ips = ["192.168.0.0/16"]
  global_alert {
    name = "dataset-size"
    value = 40
  }

  global_modules = ["RedisJSON"]

  override_region {
    name = "us-east-2"
    override_global_source_ips = ["192.10.0.0/16"]
  }

  override_region {
    name = "us-east-1"
    override_global_data_persistence = "none"
    override_global_password = "region-specific-password"
    override_global_alert {
      name = "dataset-size"
      value = 60
    }
  }

  tags = {
    "environment" = "production"
    "cost_center" = "0700"
  }
}


data "rediscloud_active_active_subscription" "example" {
	name = rediscloud_active_active_subscription.example.name
}

data "rediscloud_active_active_subscription_regions" "example" {
	subscription_name = rediscloud_active_active_subscription.example.name
}
`

const testAccResourceRedisCloudActiveActiveSubscriptionNoCreationPlan = `
  
data "rediscloud_payment_method" "card" {
	card_type = "Visa"
	last_four_numbers = "5556"
}

resource "rediscloud_active_active_subscription" "example" {
	name = "%s"
	payment_method_id = data.rediscloud_payment_method.card.id
	cloud_provider = "%s"

	maintenance_windows {
		mode = "automatic"
	}
}

data "rediscloud_active_active_subscription" "example" {
	name = rediscloud_active_active_subscription.example.name
}
`

const testAccResourceRedisCloudActiveActiveSubscriptionChangedPaymentMethod = `
data "rediscloud_payment_method" "card" {
	card_type = "Visa"
	last_four_numbers = "5556"
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
