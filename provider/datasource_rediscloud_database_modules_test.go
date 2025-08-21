package provider

import (
	"testing"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceRedisCloudDatabaseModules_basic(t *testing.T) {

	utils.TestAccRequiresEnvVar(t, "EXECUTE_TESTS")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { utils.TestAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      nil, // database modules isn't a 'real' resource
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceRedisCloudDatabaseModules,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckTypeSetElemNestedAttrs("data.rediscloud_database_modules.foo", "modules.*", map[string]string{
						"name": "RedisBloom",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.rediscloud_database_modules.foo", "modules.*", map[string]string{
						"name": "RediSearch",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.rediscloud_database_modules.foo", "modules.*", map[string]string{
						"name": "RedisJSON",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.rediscloud_database_modules.foo", "modules.*", map[string]string{
						"name": "RedisTimeSeries",
					}),
				),
			},
		},
	})
}

const testAccDataSourceRedisCloudDatabaseModules = `
data "rediscloud_database_modules" "foo" {
}
`
