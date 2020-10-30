package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"regexp"
	"testing"
)

func TestAccDataSourceRedisCloudPaymentMethod(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceRedisCloudPaymentMethod,
				Check: resource.ComposeTestCheckFunc(
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
