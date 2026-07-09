package activeactive_test

import (
	"testing"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestActiveActiveSubscriptionResourceTags_OnCreate_CRUDI(t *testing.T) {

	const resourceName = "rediscloud_active_active_subscription.example"
	const datasourceName = "data.rediscloud_active_active_subscription.example"
	resourceTags := map[string]config.Variable{
		"env": config.StringVariable("dev"),
	}
	resourceTagsCheck := map[string]knownvalue.Check{
		"env": knownvalue.StringExact("dev"),
	}

	resourceTagsUpdate := map[string]config.Variable{
		"env": config.StringVariable("prod"),
	}
	resourceTagsUpdateCheck := map[string]knownvalue.Check{
		"env": knownvalue.StringExact("prod"),
	}

	subscriptionName := testRandomWithPrefix()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { envchecks.RedisCloudCheck(t) },
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             checkAASubscriptionDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create subscription with resource tags
			{
				ConfigFile: config.StaticFile("./testdata/aa_basic_subscription_with_resource_tags.tf"),
				ConfigVariables: config.Variables{
					"rediscloud_subscription_name":          config.StringVariable(subscriptionName),
					"rediscloud_subscription_resource_tags": config.MapVariable(resourceTags),
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("name"), knownvalue.StringExact(subscriptionName)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("resource_tags"), knownvalue.MapExact(resourceTagsCheck)),
					statecheck.ExpectKnownValue(datasourceName, tfjsonpath.New("resource_tags"), knownvalue.MapExact(resourceTagsCheck)),
				},
			},
			// Step 2: Update resource tags
			{
				ConfigFile: config.StaticFile("./testdata/aa_basic_subscription_with_resource_tags.tf"),
				ConfigVariables: config.Variables{
					"rediscloud_subscription_name":          config.StringVariable(subscriptionName),
					"rediscloud_subscription_resource_tags": config.MapVariable(resourceTagsUpdate),
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("name"), knownvalue.StringExact(subscriptionName)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("resource_tags"), knownvalue.MapExact(resourceTagsUpdateCheck)),
					statecheck.ExpectKnownValue(datasourceName, tfjsonpath.New("resource_tags"), knownvalue.MapExact(resourceTagsUpdateCheck)),
				},
			},
			// Step 3: Remove resource tags
			{
				ConfigFile: config.StaticFile("./testdata/aa_basic_subscription_no_resource_tags.tf"),
				ConfigVariables: config.Variables{
					"rediscloud_subscription_name": config.StringVariable(subscriptionName),
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("name"), knownvalue.StringExact(subscriptionName)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("resource_tags"), knownvalue.MapExact(make(map[string]knownvalue.Check))),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("resource_tags"), knownvalue.MapSizeExact(0)),
					statecheck.ExpectKnownValue(datasourceName, tfjsonpath.New("resource_tags"), knownvalue.MapExact(make(map[string]knownvalue.Check))),
					statecheck.ExpectKnownValue(datasourceName, tfjsonpath.New("resource_tags"), knownvalue.MapSizeExact(0)),
				},
			},
			{
				ConfigFile: config.StaticFile("./testdata/aa_basic_subscription_with_resource_tags.tf"),
				ConfigVariables: config.Variables{
					"rediscloud_subscription_name":          config.StringVariable(subscriptionName),
					"rediscloud_subscription_resource_tags": config.MapVariable(resourceTags),
				},
			},
			// Step 4: Import subscription and verify resource tags are imported
			{
				ConfigFile: config.StaticFile("./testdata/aa_basic_subscription_with_resource_tags.tf"),
				ConfigVariables: config.Variables{
					"rediscloud_subscription_name":          config.StringVariable(subscriptionName),
					"rediscloud_subscription_resource_tags": config.MapVariable(resourceTags),
				},
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"creation_plan"},
			},
		},
	})
}

func TestActiveActiveSubscriptionResourceTags_CRUDI(t *testing.T) {

	const resourceName = "rediscloud_active_active_subscription.example"
	const datasourceName = "data.rediscloud_active_active_subscription.example"
	resourceTags := map[string]config.Variable{
		"env": config.StringVariable("dev"),
	}
	resourceTagsCheck := map[string]knownvalue.Check{
		"env": knownvalue.StringExact("dev"),
	}

	subscriptionName := testRandomWithPrefix()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { envchecks.RedisCloudCheck(t) },
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             checkAASubscriptionDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create subscription with resource tags
			{
				ConfigFile: config.StaticFile("./testdata/aa_basic_subscription_no_resource_tags.tf"),
				ConfigVariables: config.Variables{
					"rediscloud_subscription_name": config.StringVariable(subscriptionName),
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("name"), knownvalue.StringExact(subscriptionName)),
					//Values need to check for null on Create due to Terraform SDKV2 bug
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("resource_tags"), knownvalue.Null()),
					statecheck.ExpectKnownValue(datasourceName, tfjsonpath.New("resource_tags"), knownvalue.MapExact(make(map[string]knownvalue.Check))),
					statecheck.ExpectKnownValue(datasourceName, tfjsonpath.New("resource_tags"), knownvalue.MapSizeExact(0)),
				},
			},
			// Step 2: Update resource tags
			{
				ConfigFile: config.StaticFile("./testdata/aa_basic_subscription_with_resource_tags.tf"),
				ConfigVariables: config.Variables{
					"rediscloud_subscription_name":          config.StringVariable(subscriptionName),
					"rediscloud_subscription_resource_tags": config.MapVariable(resourceTags),
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("name"), knownvalue.StringExact(subscriptionName)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("resource_tags"), knownvalue.MapExact(resourceTagsCheck)),
					statecheck.ExpectKnownValue(datasourceName, tfjsonpath.New("resource_tags"), knownvalue.MapExact(resourceTagsCheck)),
				},
			},
			// Step 3: Remove resource tags
			{
				ConfigFile: config.StaticFile("./testdata/aa_basic_subscription_no_resource_tags.tf"),
				ConfigVariables: config.Variables{
					"rediscloud_subscription_name": config.StringVariable(subscriptionName),
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("name"), knownvalue.StringExact(subscriptionName)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("resource_tags"), knownvalue.MapExact(make(map[string]knownvalue.Check))),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("resource_tags"), knownvalue.MapSizeExact(0)),
					statecheck.ExpectKnownValue(datasourceName, tfjsonpath.New("resource_tags"), knownvalue.MapExact(make(map[string]knownvalue.Check))),
					statecheck.ExpectKnownValue(datasourceName, tfjsonpath.New("resource_tags"), knownvalue.MapSizeExact(0)),
				},
			},
			{
				ConfigFile: config.StaticFile("./testdata/aa_basic_subscription_with_resource_tags.tf"),
				ConfigVariables: config.Variables{
					"rediscloud_subscription_name":          config.StringVariable(subscriptionName),
					"rediscloud_subscription_resource_tags": config.MapVariable(resourceTags),
				},
			},
			// Step 5: Import subscription and verify resource tags are imported
			{
				ConfigFile: config.StaticFile("./testdata/aa_basic_subscription_with_resource_tags.tf"),
				ConfigVariables: config.Variables{
					"rediscloud_subscription_name":          config.StringVariable(subscriptionName),
					"rediscloud_subscription_resource_tags": config.MapVariable(resourceTags),
				},
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"creation_plan"},
			},
		},
	})
}
