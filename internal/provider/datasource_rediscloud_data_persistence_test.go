package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

func TestAccDataSourceRedisCloudDataPersistence_basic(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      nil, // data persistence isn't a 'real' resource
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceRedisCloudDataPersistence,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.rediscloud_data_persistence.foo", "data_persistence.#", "6"),
				),
			},
		},
	})
}

const testAccDataSourceRedisCloudDataPersistence = `
data "rediscloud_data_persistence" "foo" {
}
`
