package regions_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

const regionsDataSource = "data.rediscloud_regions.example"

func TestAccDataSourceRedisCloudRegions_all(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testhelpers.BasicPreCheck(t) },
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
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
		PreCheck:                 func() { testhelpers.BasicPreCheck(t) },
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
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
		PreCheck:                 func() { testhelpers.BasicPreCheck(t) },
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
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
