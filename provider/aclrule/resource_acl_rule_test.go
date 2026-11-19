package aclrule_test

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

const aclRuleResourceName = "rediscloud_acl_rule.test"

func TestAccResourceRedisCloudAclRule_CRUDI(t *testing.T) {
	name := utils.RandomWithPrefix() + "-test-rule"
	const rule = "+@all"

	nameUpdated := name + "_updated_name"
	const ruleUpdated = rule + " -@dangerous ~*"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, envchecks.RedisCloudCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testAccCheckAclRuleDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/resource_basic.tf"),
				ConfigVariables: config.Variables{
					"name": config.StringVariable(name),
					"rule": config.StringVariable(rule),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(aclRuleResourceName, "id", regexp.MustCompile(`^\d+$`)),
					resource.TestCheckResourceAttr(aclRuleResourceName, "name", name),
					resource.TestCheckResourceAttr(aclRuleResourceName, "rule", rule),
					func(s *terraform.State) error {
						r := s.RootModule().Resources[aclRuleResourceName]

						id, err := strconv.Atoi(r.Primary.ID)
						if err != nil {
							return fmt.Errorf("couldn't parse the rule ID: %s", r.Primary.ID)
						}

						apiClient := client.MustTestClient(t)
						apiRule, err := apiClient.Client.RedisRules.Get(context.TODO(), id)
						if err != nil {
							return err
						}

						if redis.StringValue(apiRule.Name) != name {
							return fmt.Errorf("unexpected name value: %s", redis.StringValue(apiRule.Name))
						}
						if redis.StringValue(apiRule.ACL) != rule {
							return fmt.Errorf("unexpected rule value: %s", redis.StringValue(apiRule.ACL))
						}
						return nil
					},
				),
			},
			{
				ConfigFile: config.StaticFile("testdata/resource_basic.tf"),
				ConfigVariables: config.Variables{
					"name": config.StringVariable(name),
					"rule": config.StringVariable(ruleUpdated),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(aclRuleResourceName, "name", name),
					resource.TestCheckResourceAttr(aclRuleResourceName, "rule", ruleUpdated),
				),
			},
			{
				ConfigFile: config.StaticFile("testdata/resource_basic.tf"),
				ConfigVariables: config.Variables{
					"name": config.StringVariable(nameUpdated),
					"rule": config.StringVariable(ruleUpdated),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(aclRuleResourceName, "name", nameUpdated),
					resource.TestCheckResourceAttr(aclRuleResourceName, "rule", ruleUpdated),
				),
			},
			{
				ConfigFile: config.StaticFile("testdata/resource_basic.tf"),
				ConfigVariables: config.Variables{
					"name": config.StringVariable(name),
					"rule": config.StringVariable(rule),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(aclRuleResourceName, "name", name),
					resource.TestCheckResourceAttr(aclRuleResourceName, "rule", rule),
				),
			},
			{
				ConfigFile: config.StaticFile("testdata/resource_basic.tf"),
				ConfigVariables: config.Variables{
					"name": config.StringVariable(name),
					"rule": config.StringVariable(rule),
				},
				ResourceName:      aclRuleResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// State written by the last SDKv2 release must plan cleanly with the Framework implementation.
// Both steps address the provider by its registry source so that the SDKv2 state and the
// Framework provider share the same provider address.
func TestAccResourceRedisCloudAclRule_sdkStateCompatibility(t *testing.T) {
	t.Setenv("TF_ACC_PROVIDER_NAMESPACE", "redislabs")

	name := utils.RandomWithPrefix() + "-test-rule"
	const rule = "+@all"

	resource.Test(t, resource.TestCase{
		PreCheck:     envchecks.ComposePreChecks(t, envchecks.RedisCloudCheck),
		CheckDestroy: testAccCheckAclRuleDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/resource_sdk_v2_20_0.tf"),
				ConfigVariables: config.Variables{
					"name": config.StringVariable(name),
					"rule": config.StringVariable(rule),
				},
			},
			{
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				ConfigFile:               config.StaticFile("testdata/resource_registry_source.tf"),
				ConfigVariables: config.Variables{
					"name": config.StringVariable(name),
					"rule": config.StringVariable(rule),
				},
				PlanOnly: true,
			},
		},
	})
}

func testAccCheckAclRuleDestroy(s *terraform.State) error {
	apiClient, err := client.GetTestClient()
	if err != nil {
		return err
	}

	for _, r := range s.RootModule().Resources {
		if r.Type != "rediscloud_acl_rule" {
			continue
		}

		id, err := strconv.Atoi(r.Primary.ID)
		if err != nil {
			return err
		}

		rules, err := apiClient.Client.RedisRules.List(context.TODO())
		if err != nil {
			return err
		}

		for _, rule := range rules {
			if redis.IntValue(rule.ID) == id {
				return fmt.Errorf("rule %d still exists", id)
			}
		}
	}

	return nil
}
