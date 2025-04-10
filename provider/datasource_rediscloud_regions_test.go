package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceRedisCloudRegions_all(t *testing.T) {

	testAccRequiresEnvVar(t, "EXECUTE_TEST_REGIONS")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      nil, // regions isn't a 'real' resource
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceRedisCloudRegions,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckTypeSetElemNestedAttrs("data.rediscloud_regions.foo", "regions.*", map[string]string{
						"name": "europe-west1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.rediscloud_regions.foo", "regions.*", map[string]string{
						"name": "us-west1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.rediscloud_regions.foo", "regions.*", map[string]string{
						"name": "us-west2",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.rediscloud_regions.foo", "regions.*", map[string]string{
						"name": "eu-west-1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.rediscloud_regions.foo", "regions.*", map[string]string{
						"name": "us-east-1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.rediscloud_regions.foo", "regions.*", map[string]string{
						"name": "us-east-2",
					}),
				),
			},
		},
	})
}

func TestAccDataSourceRedisCloudRegions_AWS(t *testing.T) {

	testAccRequiresEnvVar(t, "EXECUTE_TEST_REGIONS")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      nil, // regions isn't a 'real' resource
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceRedisCloudRegionsAWS,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckTypeSetElemNestedAttrs("data.rediscloud_regions.foo", "regions.*", map[string]string{
						"name": "eu-west-1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.rediscloud_regions.foo", "regions.*", map[string]string{
						"name": "us-east-1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.rediscloud_regions.foo", "regions.*", map[string]string{
						"name": "us-east-2",
					}),
				),
			},
		},
	})
}

func TestAccDataSourceRedisCloudRegions_GCP(t *testing.T) {

	testAccRequiresEnvVar(t, "EXECUTE_TEST_REGIONS")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      nil, // regions isn't a 'real' resource
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceRedisCloudRegionsGCP,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckTypeSetElemNestedAttrs("data.rediscloud_regions.foo", "regions.*", map[string]string{
						"name": "europe-west1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.rediscloud_regions.foo", "regions.*", map[string]string{
						"name": "us-west1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.rediscloud_regions.foo", "regions.*", map[string]string{
						"name": "us-west2",
					}),
				),
			},
		},
	})
}

const testAccDataSourceRedisCloudRegions = `
data "rediscloud_regions" "foo" {
}
`

const testAccDataSourceRedisCloudRegionsAWS = `
data "rediscloud_regions" "foo" {
	provider_name = "AWS"
}
`

const testAccDataSourceRedisCloudRegionsGCP = `
data "rediscloud_regions" "foo" {
	provider_name = "GCP"
}
`
