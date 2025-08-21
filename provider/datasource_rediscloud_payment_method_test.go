package provider

import (
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"regexp"
	"testing"
)

func TestAccDataSourceRedisCloudPaymentMethod_basic(t *testing.T) {

	utils.TestAccRequiresEnvVar(t, "EXECUTE_TESTS")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { utils.TestAccPreCheck(t) },
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
data "rediscloud_payment_method" "card" {
	card_type = "Visa"
	last_four_numbers = "5556"
}
`
