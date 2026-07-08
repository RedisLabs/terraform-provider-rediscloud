package provider_test

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

// TestAccResourceRedisCloudProSubscription_CMK is a semi-automated test that requires the user to pause midway through
// to give the CMK the necessary permissions.
// TODO: integrate the GCP provider and set up these permissions automatically
func TestAccResourceRedisCloudProSubscription_CMK(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")
	utils.AccRequiresEnvVar(t, "GCP_CMK_RESOURCE_NAME")

	name := testRandomWithPrefix()
	const resourceName = "rediscloud_subscription.example"
	gcpCmkResourceName := os.Getenv("GCP_CMK_RESOURCE_NAME")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./pro/testdata/cmk_gcp_step1.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(name),
				},
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttrSet(resourceName, "customer_managed_key_redis_service_account"),
					resource.TestCheckResourceAttr(resourceName, "payment_method", "credit-card"),
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
					resource.TestCheckResourceAttr(resourceName, "memory_storage", "ram"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.provider"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.region.#"), // number of regions
					resource.TestCheckResourceAttrSet(resourceName, "creation_plan.0.dataset_size_in_gb"),
					resource.TestCheckResourceAttr(resourceName, "customer_managed_key_enabled", "true"),
				),
			},
			{
				ConfigFile: config.StaticFile("./pro/testdata/cmk_gcp_step1.tf"),
				ConfigVariables: config.Variables{
					"subscription_name":     config.StringVariable(name),
					"gcp_cmk_resource_name": config.StringVariable(gcpCmkResourceName),
				},
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttrSet(resourceName, "customer_managed_key_redis_service_account"),
					resource.TestCheckResourceAttr(resourceName, "payment_method", "credit-card"),
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
					resource.TestCheckResourceAttr(resourceName, "memory_storage", "ram"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.provider"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.region.#"),
					resource.TestCheckResourceAttrSet(resourceName, "creation_plan.0.dataset_size_in_gb"),
					resource.TestCheckResourceAttr(resourceName, "customer_managed_key_enabled", "true"),
				),
			},
		},
	})
}

// TestAccResourceRedisCloudProSubscription_CMK_AWS is a fully automated AWS CMK test.
// It uses the hashicorp/aws external provider to create the KMS key and key policy
// in-fixture, removing the need for a pre-existing AWS_CMK_KEY_ARN.
func TestAccResourceRedisCloudProSubscription_CMK_AWS(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := testRandomWithPrefix() + "-pro-cmk-aws"
	const resourceName = "rediscloud_subscription.example"

	configVars := config.Variables{
		"name": config.StringVariable(name),
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccAwsPreExistingCloudAccountPreCheck(t)
			testAccAwsApiCredsPreCheck(t)
		},
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		ExternalProviders: map[string]resource.ExternalProvider{
			"aws": {
				Source:            "hashicorp/aws",
				VersionConstraint: "~> 5.0",
			},
		},
		CheckDestroy: testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				// Step 1: subscription enters encryption_key_pending; KMS key policy
				// references the role ARN returned by the subscription.
				ConfigFile:         config.StaticFile("./pro/testdata/cmk_aws_step1.tf"),
				ConfigVariables:    configVars,
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttrSet(resourceName, "customer_managed_key_aws_role_arn"),
					resource.TestCheckResourceAttr(resourceName, "payment_method", "credit-card"),
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
					resource.TestCheckResourceAttr(resourceName, "memory_storage", "ram"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.provider"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.region.#"),
					resource.TestCheckResourceAttrSet(resourceName, "creation_plan.0.dataset_size_in_gb"),
					resource.TestCheckResourceAttr(resourceName, "customer_managed_key_enabled", "true"),
				),
			},
			{
				// Step 2: subscription supplies the KMS key ARN and transitions to active.
				ConfigFile:         config.StaticFile("./pro/testdata/cmk_aws_step2.tf"),
				ConfigVariables:    configVars,
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttrSet(resourceName, "customer_managed_key_aws_role_arn"),
					resource.TestCheckResourceAttr(resourceName, "payment_method", "credit-card"),
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
					resource.TestCheckResourceAttr(resourceName, "memory_storage", "ram"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.provider"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.region.#"),
					resource.TestCheckResourceAttrSet(resourceName, "creation_plan.0.dataset_size_in_gb"),
					resource.TestCheckResourceAttr(resourceName, "customer_managed_key_enabled", "true"),
				),
			},
		},
	})
}
