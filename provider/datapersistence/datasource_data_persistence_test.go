package datapersistence_test

import (
	"os"
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

func TestAccDataSourceRedisCloudDataPersistence_basic(t *testing.T) {
	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	const dataPersistenceFoo = "data.rediscloud_data_persistence.foo"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
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
