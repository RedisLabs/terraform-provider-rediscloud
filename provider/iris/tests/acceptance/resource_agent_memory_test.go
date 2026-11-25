package acceptance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

const address = "rediscloud_iris_agent_memory.test"
const basePath = "/iris/v1/agent-memory/stores/"

type fixture struct {
	api       *client.IrisServiceClient
	name, id  string
	directory string
}

func newFixture(t *testing.T, storeConfig string) *fixture {
	t.Helper()
	if os.Getenv("TF_ACC") != "1" {
		t.Skip("live acceptance requires TF_ACC=1")
	}
	endpoint := os.Getenv("REDISCLOUD_IRIS_AGENT_MEMORY_URL")
	if endpoint == "" {
		t.Skip("requires REDISCLOUD_IRIS_AGENT_MEMORY_URL")
	}
	api, err := client.NewIrisServiceClient(client.IrisServiceConfig{URL: endpoint, Token: os.Getenv("REDISCLOUD_IRIS_AGENT_MEMORY_TOKEN"), APIKey: os.Getenv("REDISCLOUD_ACCESS_KEY"), SecretKey: os.Getenv("REDISCLOUD_SECRET_KEY")})
	if err != nil {
		t.Fatal(err)
	}
	prefix := os.Getenv("TEST_RESOURCE_PREFIX")
	if prefix == "" {
		prefix = "tf-test"
	}
	f := &fixture{api: api, name: fmt.Sprintf("%s-iris-acc-%d", prefix, time.Now().UnixNano())}
	f.directory = t.TempDir()
	for _, file := range []string{"infrastructure/database.tf", storeConfig} {
		content, err := os.ReadFile(filepath.Join("testdata", file))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(f.directory, filepath.Base(file)), content, 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		if f.id != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			err := f.api.Do(ctx, "DELETE", basePath+f.id, map[string][]string{"flush": {"false"}}, nil, nil)
			if err != nil && !client.IrisNotFound(err) {
				t.Errorf("cleanup test store %s: %v", f.id, err)
			}
		}
	})
	return f
}
func (f *fixture) remember(state *terraform.State) error {
	r := state.RootModule().Resources[address]
	if r == nil || r.Primary == nil {
		return fmt.Errorf("store missing from state")
	}
	f.id = r.Primary.ID
	return nil
}
func (f *fixture) remote(ttl int64) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		if err := f.remember(state); err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		var current struct {
			StoreID     string `json:"storeId"`
			State       string `json:"state"`
			ShortMemory struct {
				TTLSeconds int64 `json:"ttlSeconds"`
			} `json:"shortMemory"`
		}
		if err := f.api.Do(ctx, "GET", basePath+f.id, nil, nil, &current); err != nil {
			return err
		}
		if current.StoreID != f.id || current.State != "READY" {
			return fmt.Errorf("unexpected live store identity or readiness")
		}
		if current.ShortMemory.TTLSeconds != ttl {
			return fmt.Errorf("remote TTL did not match %d", ttl)
		}
		return nil
	}
}
func (f *fixture) destroy(state *terraform.State) error {
	if f.id == "" {
		if err := f.remember(state); err != nil {
			return err
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	err := f.api.Do(ctx, "GET", basePath+f.id, nil, nil, nil)
	if !client.IrisNotFound(err) {
		if err != nil {
			return fmt.Errorf("verify destruction of store %s: %w", f.id, err)
		}
		return fmt.Errorf("store %s still exists", f.id)
	}
	return nil
}
func (f *fixture) testCase(t *testing.T, steps []resource.TestStep) resource.TestCase {
	t.Helper()
	return resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, testhelpers.PreCheckNoEssentialsSubscription),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             resource.ComposeTestCheckFunc(f.destroy, testhelpers.CheckEssentialsSubscriptionDestroy),
		Steps:                    steps,
	}
}

func TestAccResourceIrisAgentMemory_CRUDI(t *testing.T) {
	f := newFixture(t, "store.tf")
	resource.Test(t, f.testCase(t, []resource.TestStep{
		{ConfigDirectory: config.StaticDirectory(f.directory), ConfigVariables: config.Variables{"name": config.StringVariable(f.name)}, Check: f.remote(86400)},
		{ConfigDirectory: config.StaticDirectory(f.directory), ConfigVariables: config.Variables{"name": config.StringVariable(f.name), "store_name": config.StringVariable(f.name + "-updated"), "short_ttl": config.IntegerVariable(172800)}, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)}}, Check: f.remote(172800)},
		{ConfigDirectory: config.StaticDirectory(f.directory), ConfigVariables: config.Variables{"name": config.StringVariable(f.name), "store_name": config.StringVariable(f.name + "-updated"), "short_ttl": config.IntegerVariable(172800)}, PlanOnly: true},
		{ConfigDirectory: config.StaticDirectory(f.directory), ConfigVariables: config.Variables{"name": config.StringVariable(f.name), "store_name": config.StringVariable(f.name + "-updated"), "short_ttl": config.IntegerVariable(0)}, ExpectError: regexp.MustCompile("Invalid Attribute Value")},
		{ConfigDirectory: config.StaticDirectory(f.directory), ConfigVariables: config.Variables{"name": config.StringVariable(f.name), "store_name": config.StringVariable(f.name + "-updated"), "short_ttl": config.IntegerVariable(172800)}, Check: f.remote(172800)},
		{ConfigDirectory: config.StaticDirectory(f.directory), ConfigVariables: config.Variables{"name": config.StringVariable(f.name), "store_name": config.StringVariable(f.name + "-updated"), "short_ttl": config.IntegerVariable(172800)}, ResourceName: address, ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"custom_memory_type"}},
		{ConfigDirectory: config.StaticDirectory(f.directory), ConfigVariables: config.Variables{"name": config.StringVariable(f.name), "store_name": config.StringVariable(f.name + "-updated"), "short_ttl": config.IntegerVariable(172800)}, PreConfig: func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			err := f.api.Do(ctx, "DELETE", basePath+f.id, map[string][]string{"flush": {"false"}}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
		}, Check: f.remote(172800)},
	}))
}
func TestAccResourceIrisAgentMemory_Replacement(t *testing.T) {
	f := newFixture(t, "store.tf")
	var previous string
	resource.Test(t, f.testCase(t, []resource.TestStep{
		{ConfigDirectory: config.StaticDirectory(f.directory), ConfigVariables: config.Variables{"name": config.StringVariable(f.name)}, Check: func(s *terraform.State) error {
			if err := f.remote(86400)(s); err != nil {
				return err
			}
			previous = f.id
			return nil
		}},
		{ConfigDirectory: config.StaticDirectory(f.directory), ConfigVariables: config.Variables{"name": config.StringVariable(f.name), "field_type": config.StringVariable("BOOLEAN")}, PlanOnly: true, ExpectNonEmptyPlan: true, ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionDestroyBeforeCreate)}}},
		{ConfigDirectory: config.StaticDirectory(f.directory), ConfigVariables: config.Variables{"name": config.StringVariable(f.name), "field_type": config.StringVariable("BOOLEAN")}, Check: func(s *terraform.State) error {
			if err := f.remote(86400)(s); err != nil {
				return err
			}
			if f.id == previous {
				return fmt.Errorf("store was not replaced")
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			err := f.api.Do(ctx, "GET", basePath+previous, nil, nil, nil)
			if !client.IrisNotFound(err) {
				return fmt.Errorf("superseded store was not deleted")
			}
			return nil
		}},
	}))
}
func TestAccResourceIrisAgentMemory_CustomerModels(t *testing.T) {
	llm, embedding := os.Getenv("AGENT_MEMORY_MODEL_API_KEY"), os.Getenv("AGENT_MEMORY_MODEL_API_KEY")
	if llm == "" {
		llm, embedding = os.Getenv("AGENT_MEMORY_LLM_API_KEY"), os.Getenv("AGENT_MEMORY_EMBEDDING_API_KEY")
	}
	if llm == "" || embedding == "" {
		t.Skip("requires AGENT_MEMORY_MODEL_API_KEY (or both legacy LLM and embedding key variables)")
	}
	f := newFixture(t, "store.tf")
	t.Setenv("TF_VAR_customer_models", "true")
	t.Setenv("TF_VAR_llm_api_key", llm)
	t.Setenv("TF_VAR_embedding_api_key", embedding)
	resource.Test(t, f.testCase(t, []resource.TestStep{
		{ConfigDirectory: config.StaticDirectory(f.directory), ConfigVariables: config.Variables{"name": config.StringVariable(f.name)}, Check: f.remote(86400)},
		{ConfigDirectory: config.StaticDirectory(f.directory), ConfigVariables: config.Variables{"name": config.StringVariable(f.name)}, ResourceName: address, ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"custom_memory_type", "llm_api_key_version", "embedding_api_key_version"}},
	}))
}
