package paymentmethod_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func TestAccDataSourceRedisCloudPaymentMethod_basic(t *testing.T) {
	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testhelpers.BasicPreCheck(t) },
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             nil, // payment method isn't managed by this provider
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/datasource_basic.tf"),
				ConfigVariables: config.Variables{
					"card_type":         config.StringVariable("Visa"),
					"last_four_numbers": config.StringVariable("5556"),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(
						"data.rediscloud_payment_method.card", "id", regexp.MustCompile(`^\d+$`)),
					resource.TestMatchResourceAttr(
						"data.rediscloud_payment_method.card", "card_type", regexp.MustCompile(`^\w+$`)),
					resource.TestMatchResourceAttr(
						"data.rediscloud_payment_method.card", "last_four_numbers", regexp.MustCompile(`^\d{4}$`)),
				),
			},
		},
	})
}
