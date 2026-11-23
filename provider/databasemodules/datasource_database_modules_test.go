package databasemodules_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAccDataSourceRedisCloudDatabaseModules_basic(t *testing.T) {
	const dataSourceName = "data.rediscloud_database_modules.example"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, envchecks.RedisCloudCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/database_modules_data_source.tf"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "id", "ALL"),
					resource.TestCheckTypeSetElemNestedAttrs(dataSourceName, "modules.*", map[string]string{
						"name": "RedisBloom",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(dataSourceName, "modules.*", map[string]string{
						"name": "RediSearch",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(dataSourceName, "modules.*", map[string]string{
						"name": "RedisJSON",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(dataSourceName, "modules.*", map[string]string{
						"name": "RedisTimeSeries",
					}),
				),
			},
		},
	})
}
