package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"regexp"
	"testing"
)

func TestAccDataSourceRedisCloudPaymentMethod_basic(t *testing.T) {

	testAccRequiresEnvVar(t, "EXECUTE_TEST_PAYMENT_METHOD")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      nil, // payment method isn't managed by this provider
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceRedisCloudPaymentMethod,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(
						"data.rediscloud_payment_method.foo", "id", regexp.MustCompile("^\\d*$")),
					resource.TestMatchResourceAttr(
						"data.rediscloud_payment_method.foo", "card_type", regexp.MustCompile("^\\w*$")),
					resource.TestMatchResourceAttr(
						"data.rediscloud_payment_method.foo", "last_four_numbers", regexp.MustCompile("^\\d*$")),
				),
			},
		},
	})
}

const testAccDataSourceRedisCloudPaymentMethod = `
data "rediscloud_payment_method" "foo" {
  card_type = "Visa"
}
`
