package paymentmethod_test

import (
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

var protoV5ProviderFactories = map[string]func() (tfprotov5.ProviderServer, error){
	"rediscloud": func() (tfprotov5.ProviderServer, error) {
		muxServer, err := provider.MuxProviderServerCreator(
			provider.NewSdkProvider("dev")(),
			provider.NewFrameworkProvider("dev")(),
		)
		if err != nil {
			return nil, err
		}
		return muxServer(), nil
	},
}

func testAccPreCheck(t *testing.T) {
	for _, name := range []string{"REDISCLOUD_URL", "REDISCLOUD_ACCESS_KEY", "REDISCLOUD_SECRET_KEY"} {
		if _, ok := os.LookupEnv(name); !ok {
			t.Fatalf("Missing `%s` environment variable", name)
		}
	}
}

func TestAccDataSourceRedisCloudPaymentMethod_basic(t *testing.T) {
	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             nil, // payment method isn't managed by this provider
		Steps: []resource.TestStep{
			{
				Config: utils.RenderTestConfig(t, "./testdata/datasource_basic.tf", map[string]string{
					"__CARD_TYPE__":         "Visa",
					"__LAST_FOUR_NUMBERS__": "5556",
				}),
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
