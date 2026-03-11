package regions_test

import (
	"os"
	"testing"

	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

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
	for _, name := range []string{rediscloudApi.AccessKeyEnvVar, rediscloudApi.SecretKeyEnvVar} {
		if _, ok := os.LookupEnv(name); !ok {
			t.Fatalf("Missing `%s` environment variable", name)
		}
	}
}

const regionsDataSource = "data.rediscloud_regions.example"

func TestAccDataSourceRedisCloudRegions_all(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             nil,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/datasource_all.tf"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(regionsDataSource, "regions.0.region_id"),
					resource.TestCheckTypeSetElemNestedAttrs(regionsDataSource, "regions.*", map[string]string{
						"name": "europe-west1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(regionsDataSource, "regions.*", map[string]string{
						"name": "us-west1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(regionsDataSource, "regions.*", map[string]string{
						"name": "us-west2",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(regionsDataSource, "regions.*", map[string]string{
						"name": "eu-west-1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(regionsDataSource, "regions.*", map[string]string{
						"name": "us-east-1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(regionsDataSource, "regions.*", map[string]string{
						"name": "us-east-2",
					}),
				),
			},
		},
	})
}

func TestAccDataSourceRedisCloudRegions_AWS(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             nil,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/datasource_filtered.tf"),
				ConfigVariables: config.Variables{
					"provider_name": config.StringVariable("AWS"),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(regionsDataSource, "regions.0.region_id"),
					resource.TestCheckTypeSetElemNestedAttrs(regionsDataSource, "regions.*", map[string]string{
						"name": "eu-west-1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(regionsDataSource, "regions.*", map[string]string{
						"name": "us-east-1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(regionsDataSource, "regions.*", map[string]string{
						"name": "us-east-2",
					}),
				),
			},
		},
	})
}

func TestAccDataSourceRedisCloudRegions_GCP(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             nil,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/datasource_filtered.tf"),
				ConfigVariables: config.Variables{
					"provider_name": config.StringVariable("GCP"),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(regionsDataSource, "regions.0.region_id"),
					resource.TestCheckTypeSetElemNestedAttrs(regionsDataSource, "regions.*", map[string]string{
						"name": "europe-west1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(regionsDataSource, "regions.*", map[string]string{
						"name": "us-west1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(regionsDataSource, "regions.*", map[string]string{
						"name": "us-west2",
					}),
				),
			},
		},
	})
}
