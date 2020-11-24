package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

func TestAccDataSourceRedisCloudDatabaseModules_basic(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      nil, // database modules isn't a 'real' resource
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceRedisCloudDatabaseModules,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.rediscloud_database_modules.foo", "modules.#", "4"),
				),
			},
		},
	})
}

const testAccDataSourceRedisCloudDatabaseModules = `
data "rediscloud_database_modules" "foo" {
}
`
