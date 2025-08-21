package provider

import (
	"testing"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceRedisCloudDataPersistence_basic(t *testing.T) {

	utils.TestAccRequiresEnvVar(t, "EXECUTE_TESTS")

	const dataPersistenceFoo = "data.rediscloud_data_persistence.foo"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { utils.TestAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      nil, // data persistence isn't a 'real' resource
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceRedisCloudDataPersistence,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckTypeSetElemNestedAttrs(dataPersistenceFoo, "data_persistence.*", map[string]string{
						"name": "snapshot-every-12-hours",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(dataPersistenceFoo, "data_persistence.*", map[string]string{
						"name": "snapshot-every-6-hours",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(dataPersistenceFoo, "data_persistence.*", map[string]string{
						"name": "snapshot-every-1-hour",
					}),
				),
			},
		},
	})
}

const testAccDataSourceRedisCloudDataPersistence = `
data "rediscloud_data_persistence" "foo" {
}
`
