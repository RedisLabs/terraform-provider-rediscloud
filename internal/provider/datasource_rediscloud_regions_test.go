package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

func TestAccDataSourceRedisCloudRegions_all(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      nil, // regions isn't a 'real' resource
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceRedisCloudRegions,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.rediscloud_regions.foo", "regions.#", "36"),
				),
			},
		},
	})
}

func TestAccDataSourceRedisCloudRegions_AWS(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      nil, // regions isn't a 'real' resource
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceRedisCloudRegionsAWS,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.rediscloud_regions.foo", "regions.#", "16"),
				),
			},
		},
	})
}

func TestAccDataSourceRedisCloudRegions_GCP(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      nil, // regions isn't a 'real' resource
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceRedisCloudRegionsGCP,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.rediscloud_regions.foo", "regions.#", "20"),
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
