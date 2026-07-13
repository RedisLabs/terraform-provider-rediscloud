package datapersistence_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAccDataSourceRedisCloudDataPersistence_basic(t *testing.T) {

	const dataPersistenceFoo = "data.rediscloud_data_persistence.foo"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, envchecks.RedisCloudCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             nil, // data persistence isn't a 'real' resource
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
