package provider_test

import (
	"testing"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourceRedisCloudDatabaseModules_basic(t *testing.T) {

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { envchecks.RedisCloudCheck(t) },
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             nil, // database modules isn't a 'real' resource
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
