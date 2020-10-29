package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceRedisCloudSubscription(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceRedisCloudSubsctiption,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"rediscloud_subscription.foo", "sample_attribute", regexp.MustCompile("^ba")),
				),
			},
		},
	})
}

const testAccResourceRedisCloudSubsctiption = `
resource "rediscloud_subscription" "foo" {
  sample_attribute = "bar"
}
`
