package provider_test

import (
	"flag"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

var essentialsMarketplaceFlag = flag.Bool("essentialsMarketplace", false,
	"Add this flag '-essentialsMarketplace' to run tests for marketplace associated accounts")

func TestAccResourceRedisCloudEssentialsSubscription_Free_CRUDI(t *testing.T) {

	subscriptionName := testRandomWithPrefix()
	subscriptionNameUpdated := subscriptionName + "-updated"

	const resourceName = "rediscloud_essentials_subscription.example"

	resource.Test(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, testhelpers.PreCheckNoEssentialsSubscription),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckEssentialsSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudFreeEssentialsSubscription, subscriptionName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Test the resource
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", subscriptionName),
					resource.TestCheckResourceAttr(resourceName, "status", "active"),
					resource.TestCheckResourceAttrSet(resourceName, "plan_id"),
					resource.TestCheckResourceAttr(resourceName, "payment_method_id", "0"),
					resource.TestCheckResourceAttrSet(resourceName, "creation_date"),
				),
			},
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudFreeEssentialsSubscription, subscriptionNameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Test the resource
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", subscriptionNameUpdated),
					resource.TestCheckResourceAttr(resourceName, "status", "active"),
					resource.TestCheckResourceAttrSet(resourceName, "plan_id"),
					resource.TestCheckResourceAttr(resourceName, "payment_method_id", "0"),
					resource.TestCheckResourceAttrSet(resourceName, "creation_date"),
				),
			},
			{
				Config:            fmt.Sprintf(testAccResourceRedisCloudFreeEssentialsSubscription, subscriptionNameUpdated),
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccResourceRedisCloudEssentialsSubscription_Paid_CreditCard_CRUDI(t *testing.T) {

	subscriptionName := testRandomWithPrefix()
	subscriptionNameUpdated := subscriptionName + "-updated"

	const resourceName = "rediscloud_essentials_subscription.example"

	resource.Test(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, testhelpers.PreCheckNoEssentialsSubscription),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckEssentialsSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudPaidCreditCardEssentialsSubscription, subscriptionName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Test the resource
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", subscriptionName),
					resource.TestCheckResourceAttr(resourceName, "status", "active"),
					resource.TestCheckResourceAttrSet(resourceName, "plan_id"),
					resource.TestCheckResourceAttrSet(resourceName, "payment_method"),
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
					resource.TestCheckResourceAttrSet(resourceName, "creation_date"),
				),
			},
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudPaidCreditCardEssentialsSubscription, subscriptionNameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Test the resource
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", subscriptionNameUpdated),
					resource.TestCheckResourceAttr(resourceName, "status", "active"),
					resource.TestCheckResourceAttrSet(resourceName, "plan_id"),
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
					resource.TestCheckResourceAttrSet(resourceName, "creation_date"),
				),
			},
			{
				Config:            fmt.Sprintf(testAccResourceRedisCloudPaidCreditCardEssentialsSubscription, subscriptionNameUpdated),
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccResourceRedisCloudEssentialsSubscription_Paid_NoPaymentType_CRUDI(t *testing.T) {

	subscriptionName := testRandomWithPrefix()
	subscriptionNameUpdated := subscriptionName + "-updated"

	const resourceName = "rediscloud_essentials_subscription.example"

	resource.Test(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, testhelpers.PreCheckNoEssentialsSubscription),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckEssentialsSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudPaidNoPaymentTypeEssentialsSubscription, subscriptionName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Test the resource
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", subscriptionName),
					resource.TestCheckResourceAttr(resourceName, "status", "active"),
					resource.TestCheckResourceAttrSet(resourceName, "plan_id"),
					resource.TestCheckResourceAttrSet(resourceName, "payment_method"),
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
					resource.TestCheckResourceAttrSet(resourceName, "creation_date"),
				),
			},
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudPaidNoPaymentTypeEssentialsSubscription, subscriptionNameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Test the resource
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", subscriptionNameUpdated),
					resource.TestCheckResourceAttr(resourceName, "status", "active"),
					resource.TestCheckResourceAttrSet(resourceName, "plan_id"),
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
					resource.TestCheckResourceAttrSet(resourceName, "creation_date"),
				),
			},
			{
				Config:            fmt.Sprintf(testAccResourceRedisCloudPaidNoPaymentTypeEssentialsSubscription, subscriptionNameUpdated),
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccResourceRedisCloudEssentialsSubscription_Paid_Marketplace_CRUDI(t *testing.T) {
	// Only the qa environment has access to the marketplace, so this test will normally fail.
	// Leaving this in the test suite for manual runs

	if !*essentialsMarketplaceFlag {
		t.Skip("The '-essentialsMarketplace' parameter wasn't provided in the test command.")
	}

	subscriptionName := testRandomWithPrefix()
	subscriptionNameUpdated := subscriptionName + "-updated"

	const resourceName = "rediscloud_essentials_subscription.example"
	const datasourceName = "data.rediscloud_essentials_subscription.example"

	resource.Test(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, testhelpers.PreCheckNoEssentialsSubscription),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckEssentialsSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudPaidMarketplaceEssentialsSubscription, subscriptionName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Test the resource
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", subscriptionName),
					resource.TestCheckResourceAttr(resourceName, "status", "active"),
					resource.TestCheckResourceAttrSet(resourceName, "plan_id"),
					resource.TestCheckNoResourceAttr(datasourceName, "payment_method_id"),
					//resource.TestCheckResourceAttr(resourceName, "payment_method", "marketplace"), // empty from API?
					resource.TestCheckResourceAttrSet(resourceName, "creation_date"),

					// Test the datasource
					resource.TestCheckResourceAttrSet(datasourceName, "id"),
					resource.TestCheckResourceAttr(datasourceName, "name", subscriptionName),
					resource.TestCheckResourceAttr(datasourceName, "status", "active"),
					resource.TestCheckResourceAttrSet(datasourceName, "plan_id"),
					resource.TestCheckNoResourceAttr(datasourceName, "payment_method_id"),
					resource.TestCheckResourceAttrSet(datasourceName, "creation_date"),
				),
			},
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudPaidMarketplaceEssentialsSubscription, subscriptionNameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Test the resource
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", subscriptionNameUpdated),
					resource.TestCheckResourceAttr(resourceName, "status", "active"),
					resource.TestCheckResourceAttrSet(resourceName, "plan_id"),
					resource.TestCheckResourceAttr(resourceName, "payment_method", "credit-card"),
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
					resource.TestCheckResourceAttrSet(resourceName, "creation_date"),

					// Test the datasource
					resource.TestCheckResourceAttrSet(datasourceName, "id"),
					resource.TestCheckResourceAttr(datasourceName, "name", subscriptionNameUpdated),
					resource.TestCheckResourceAttr(datasourceName, "status", "active"),
					resource.TestCheckResourceAttrSet(datasourceName, "plan_id"),
					resource.TestCheckResourceAttrSet(datasourceName, "payment_method_id"),
					resource.TestCheckResourceAttrSet(datasourceName, "creation_date"),
				),
			},
			{
				Config:            fmt.Sprintf(testAccResourceRedisCloudPaidMarketplaceEssentialsSubscription, subscriptionNameUpdated),
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccResourceRedisCloudEssentialsSubscription_Incorrect_PaymentIdForType(t *testing.T) {

	subscriptionName := testRandomWithPrefix()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, envchecks.RedisCloudCheck),
		CheckDestroy:             testhelpers.CheckEssentialsSubscriptionDestroy,
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccResourceRedisCloudPaidIncorrectPaymentTypeEssentialsSubscription, subscriptionName),
				ExpectError: regexp.MustCompile("payment methods aside from credit-card cannot have a payment ID"),
			},
		},
	})
}

const testAccResourceRedisCloudFreeEssentialsSubscription = `
data "rediscloud_essentials_plan" "example" {
	name = "30MB"
	cloud_provider = "AWS"
	region = "us-east-1"
}

data "rediscloud_payment_method" "card" {
	card_type = "Visa"
	last_four_numbers = "5556"
}

resource "rediscloud_essentials_subscription" "example" {
	name = "%s"
	plan_id = data.rediscloud_essentials_plan.example.id
	# payment_method = "credit-card"
	# payment_method_id = data.rediscloud_payment_method.card.id
}
`

const testAccResourceRedisCloudPaidMarketplaceEssentialsSubscription = `
data "rediscloud_essentials_plan" "example" {
	name = "250MB"
	cloud_provider = "AWS"
	region = "us-east-1"
}

resource "rediscloud_essentials_subscription" "example" {
	name = "%s"
	plan_id = data.rediscloud_essentials_plan.example.id
	payment_method = "marketplace"
}

data "rediscloud_essentials_subscription" "example" {
	name = rediscloud_essentials_subscription.example.name
}
`

const testAccResourceRedisCloudPaidCreditCardEssentialsSubscription = `
data "rediscloud_payment_method" "card" {
	card_type = "Visa"
	last_four_numbers = "5556"
}

data "rediscloud_essentials_plan" "example" {
	name = "250MB"
	cloud_provider = "AWS"
	region = "us-east-1"
}

resource "rediscloud_essentials_subscription" "example" {
	name = "%s"
	plan_id = data.rediscloud_essentials_plan.example.id
	payment_method_id = data.rediscloud_payment_method.card.id
	payment_method = "credit-card"
}
`

// doesn't contain credit-card, tests for default
const testAccResourceRedisCloudPaidNoPaymentTypeEssentialsSubscription = `
data "rediscloud_payment_method" "card" {
	card_type = "Visa"
	last_four_numbers = "5556"
}

data "rediscloud_essentials_plan" "example" {
	name = "250MB"
	cloud_provider = "AWS"
	region = "us-east-1"
}

resource "rediscloud_essentials_subscription" "example" {
	name = "%s"
	plan_id = data.rediscloud_essentials_plan.example.id
	payment_method_id = data.rediscloud_payment_method.card.id
}
`

const testAccResourceRedisCloudPaidIncorrectPaymentTypeEssentialsSubscription = `
data "rediscloud_essentials_plan" "example" {
	name = "250MB"
	cloud_provider = "AWS"
	region = "us-east-1"
}

resource "rediscloud_essentials_subscription" "example" {
	name = "%s"
	plan_id = data.rediscloud_essentials_plan.example.id
	payment_method = "marketplace"
	payment_method_id = 999999999
}
`
