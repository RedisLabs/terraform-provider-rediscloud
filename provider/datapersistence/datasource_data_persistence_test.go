package datapersistence_test

import (
	"testing"

	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func testAccPreCheck(t *testing.T) {
	t.Helper()
	testhelpers.RequireEnvVars(t, provider.RedisCloudUrlEnvVar, rediscloudApi.AccessKeyEnvVar, rediscloudApi.SecretKeyEnvVar)
}

func TestAccDataSourceRedisCloudDataPersistence_basic(t *testing.T) {
	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	const dataPersistenceFoo = "data.rediscloud_data_persistence.foo"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
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
