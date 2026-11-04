package peering_test

import (
	"net"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

// lastSDKProviderVersion is the last release that served the peering
// resources from the SDKv2 provider.
const lastSDKProviderVersion = "2.20.0"

func TestAccResourceRedisCloudSubscriptionPeering_aws(t *testing.T) {
	resourceTags, resourceTagsCheck := envchecks.BYOCResourceTagsValueAndCheck()
	cloudAccountName, cloudAccountCheck := envchecks.AWSBYOCValueAndCheck()
	awsPeering, awsPeeringCheck := envchecks.AwsPeeringValueAndCheck()
	name := utils.RandomWithPrefix()
	subCidrRange := subscriptionCidrAvoiding(awsPeering.VpcCidr)

	const resourceName = "rediscloud_subscription_peering.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: envchecks.ComposePreChecks(t, resourceTagsCheck,
			envchecks.RedisCloudCheck,
			awsPeeringCheck,
			cloudAccountCheck,
			awsPeeringFormatCheck(awsPeering),
		),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/resource_subscription_peering_aws.tf"),
				ConfigVariables: config.Variables{
					"team":               config.StringVariable(resourceTags.Team),
					"owner":              config.StringVariable(resourceTags.Owner),
					"cloud_account_name": config.StringVariable(cloudAccountName),
					"subscription_name":  config.StringVariable(name),
					"subscription_cidr":  config.StringVariable(subCidrRange),
					"aws_region":         config.StringVariable(awsPeering.Region),
					"aws_account_id":     config.StringVariable(awsPeering.AccountId),
					"vpc_id":             config.StringVariable(awsPeering.VpcId),
					"vpc_cidr":           config.StringVariable(awsPeering.VpcCidr),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(resourceName, "id", regexp.MustCompile(`^\d*/\d*$`)),
					resource.TestCheckResourceAttrSet(resourceName, "status"),
					resource.TestCheckResourceAttr(resourceName, "provider_name", "AWS"),
					resource.TestCheckResourceAttrSet(resourceName, "aws_account_id"),
					resource.TestCheckResourceAttrSet(resourceName, "vpc_id"),
					resource.TestCheckResourceAttr(resourceName, "vpc_cidr", awsPeering.VpcCidr),
					resource.TestCheckResourceAttr(resourceName, "vpc_cidrs.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "vpc_cidrs.0", awsPeering.VpcCidr),
					resource.TestCheckResourceAttrSet(resourceName, "region"),
					resource.TestCheckResourceAttrSet(resourceName, "aws_peering_id"),
				),
			},
			{
				ConfigFile: config.StaticFile("./testdata/resource_subscription_peering_aws.tf"),
				ConfigVariables: config.Variables{
					"team":               config.StringVariable(resourceTags.Team),
					"owner":              config.StringVariable(resourceTags.Owner),
					"cloud_account_name": config.StringVariable(cloudAccountName),
					"subscription_name":  config.StringVariable(name),
					"subscription_cidr":  config.StringVariable(subCidrRange),
					"aws_region":         config.StringVariable(awsPeering.Region),
					"aws_account_id":     config.StringVariable(awsPeering.AccountId),
					"vpc_id":             config.StringVariable(awsPeering.VpcId),
					"vpc_cidr":           config.StringVariable(awsPeering.VpcCidr),
				},
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccResourceRedisCloudSubscriptionPeering_gcp(t *testing.T) {
	gcpVpcProject, gcpVpcProjectCheck := envchecks.ValueAndCheck("GCP_VPC_PROJECT")
	gcpVpcId, gcpVpcIdCheck := envchecks.ValueAndCheck("GCP_VPC_ID")
	name := utils.RandomWithPrefix()

	const resourceName = "rediscloud_subscription_peering.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, envchecks.RedisCloudCheck, gcpVpcProjectCheck, gcpVpcIdCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/resource_subscription_peering_gcp.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(name),
					"gcp_project_id":    config.StringVariable(gcpVpcProject),
					"gcp_network_name":  config.StringVariable(gcpVpcId),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(resourceName, "id", regexp.MustCompile(`^\d*/\d*$`)),
					resource.TestCheckResourceAttr(resourceName, "provider_name", "GCP"),
					resource.TestCheckResourceAttrSet(resourceName, "status"),
					resource.TestCheckResourceAttrSet(resourceName, "gcp_project_id"),
					resource.TestCheckResourceAttrSet(resourceName, "gcp_network_name"),
					resource.TestCheckResourceAttrSet(resourceName, "gcp_redis_project_id"),
					resource.TestCheckResourceAttrSet(resourceName, "gcp_redis_network_name"),
					resource.TestCheckResourceAttrSet(resourceName, "gcp_peering_id"),
				),
			},
			{
				ConfigFile: config.StaticFile("./testdata/resource_subscription_peering_gcp.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(name),
					"gcp_project_id":    config.StringVariable(gcpVpcProject),
					"gcp_network_name":  config.StringVariable(gcpVpcId),
				},
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccResourceRedisCloudSubscriptionPeering_sdkStateCompatibility applies with
// the last SDKv2 release, then expects the Framework provider to plan no change.
func TestAccResourceRedisCloudSubscriptionPeering_sdkStateCompatibility(t *testing.T) {
	resourceTags, resourceTagsCheck := envchecks.BYOCResourceTagsValueAndCheck()
	cloudAccountName, cloudAccountCheck := envchecks.AWSBYOCValueAndCheck()
	awsPeering, awsPeeringCheck := envchecks.AwsPeeringValueAndCheck()
	name := utils.RandomWithPrefix()
	subCidrRange := subscriptionCidrAvoiding(awsPeering.VpcCidr)

	// The external provider needs the generated required_providers block, which
	// plugin-testing only adds to raw Config, not to ConfigFile.
	sdkConfig, err := os.ReadFile("./testdata/resource_subscription_peering_aws.tf")
	if err != nil {
		t.Fatal(err)
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: envchecks.ComposePreChecks(t, resourceTagsCheck,
			envchecks.RedisCloudCheck,
			awsPeeringCheck,
			cloudAccountCheck,
			awsPeeringFormatCheck(awsPeering),
		),
		CheckDestroy: testhelpers.CheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"rediscloud": {
						Source:            "RedisLabs/rediscloud",
						VersionConstraint: lastSDKProviderVersion,
					},
				},
				Config: string(sdkConfig),
				ConfigVariables: config.Variables{
					"team":               config.StringVariable(resourceTags.Team),
					"owner":              config.StringVariable(resourceTags.Owner),
					"cloud_account_name": config.StringVariable(cloudAccountName),
					"subscription_name":  config.StringVariable(name),
					"subscription_cidr":  config.StringVariable(subCidrRange),
					"aws_region":         config.StringVariable(awsPeering.Region),
					"aws_account_id":     config.StringVariable(awsPeering.AccountId),
					"vpc_id":             config.StringVariable(awsPeering.VpcId),
					"vpc_cidr":           config.StringVariable(awsPeering.VpcCidr),
				},
			},
			{
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				ConfigFile:               config.StaticFile("./testdata/resource_subscription_peering_aws.tf"),
				ConfigVariables: config.Variables{
					"team":               config.StringVariable(resourceTags.Team),
					"owner":              config.StringVariable(resourceTags.Owner),
					"cloud_account_name": config.StringVariable(cloudAccountName),
					"subscription_name":  config.StringVariable(name),
					"subscription_cidr":  config.StringVariable(subCidrRange),
					"aws_region":         config.StringVariable(awsPeering.Region),
					"aws_account_id":     config.StringVariable(awsPeering.AccountId),
					"vpc_id":             config.StringVariable(awsPeering.VpcId),
					"vpc_cidr":           config.StringVariable(awsPeering.VpcCidr),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// subscriptionCidrAvoiding picks a subscription CIDR that does not overlap the
// peer VPC. Offline the env is empty and the test skips at the TF_ACC gate, so
// an invalid value is tolerated here and rejected by awsPeeringFormatCheck.
func subscriptionCidrAvoiding(vpcCidr string) string {
	const subCidrRange = "10.0.0.0/24"
	if overlap, err := cidrRangesOverlap(subCidrRange, vpcCidr); err == nil && overlap {
		return "172.16.0.0/24"
	}
	return subCidrRange
}

func awsPeeringFormatCheck(awsPeering envchecks.AWSPeering) func(t *testing.T) bool {
	return func(t *testing.T) bool {
		if _, err := cidrRangesOverlap("10.0.0.0/24", awsPeering.VpcCidr); err != nil {
			t.Errorf("AWS_VPC_CIDR is not a valid CIDR range %q: %s", awsPeering.VpcCidr, err)
			return false
		}
		matchesRegex(t, awsPeering.Region, `^[a-z]+-[a-z]+-\d+$`)
		matchesRegex(t, awsPeering.AccountId, `^\d+$`)
		matchesRegex(t, awsPeering.VpcId, `^vpc-[a-z\d]+$`)
		return true
	}
}

func matchesRegex(t *testing.T, value string, regex string) {
	if !regexp.MustCompile(regex).MatchString(value) {
		t.Fatalf("%s doesn't match regex %s", value, regex)
	}
}

func cidrRangesOverlap(cidr1 string, cidr2 string) (bool, error) {
	_, first, err := net.ParseCIDR(cidr1)
	if err != nil {
		return false, err
	}
	_, second, err := net.ParseCIDR(cidr2)
	if err != nil {
		return false, err
	}

	return first.Contains(second.IP) || second.Contains(first.IP), nil
}
