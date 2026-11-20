package acceptance

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

const address = "rediscloud_iris_agent_memory.test"
const basePath = "/iris/v1/agent-memory/stores/"

type fixture struct {
	api      *client.IrisServiceClient
	name, id string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	if os.Getenv("TF_ACC") != "1" {
		t.Skip("live acceptance requires TF_ACC=1")
	}
	endpoint := os.Getenv("REDISCLOUD_IRIS_AGENT_MEMORY_URL")
	if endpoint == "" {
		t.Fatal("set REDISCLOUD_IRIS_AGENT_MEMORY_URL")
	}
	api, err := client.NewIrisServiceClient(client.IrisServiceConfig{URL: endpoint, Token: os.Getenv("REDISCLOUD_IRIS_AGENT_MEMORY_TOKEN"), APIKey: os.Getenv("REDISCLOUD_ACCESS_KEY"), SecretKey: os.Getenv("REDISCLOUD_SECRET_KEY")})
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{api: api, name: fmt.Sprintf("tf-iris-acc-%d", time.Now().UnixNano())}
	t.Cleanup(func() {
		if f.id != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			_, err := f.api.Call(ctx, "DELETE", basePath+f.id, map[string][]string{"flush": {"false"}}, nil)
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
		v, err := f.api.Call(ctx, "GET", basePath+f.id, nil, nil)
		if err != nil {
			return err
		}
		if v["storeId"] != f.id || v["state"] != "READY" {
			return fmt.Errorf("unexpected live store identity or readiness")
		}
		short, ok := v["shortMemory"].(map[string]any)
		if !ok || short["ttlSeconds"] != json.Number(fmt.Sprint(ttl)) {
			return fmt.Errorf("remote TTL did not match %d", ttl)
		}
		return nil
	}
}
func (f *fixture) destroy(_ *terraform.State) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	_, err := f.api.Call(ctx, "GET", basePath+f.id, nil, nil)
	if !client.IrisNotFound(err) {
		if err != nil {
			return fmt.Errorf("verify destruction of store %s: %w", f.id, err)
		}
		return fmt.Errorf("store %s still exists", f.id)
	}
	return nil
}
func (f *fixture) testCase(steps []resource.TestStep) resource.TestCase {
	return resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), CheckDestroy: f.destroy, Steps: steps}
}

func TestAccResourceIrisAgentMemory_CRUDI(t *testing.T) {
	f := newFixture(t)
	resource.Test(t, f.testCase([]resource.TestStep{
		{ConfigFile: config.StaticFile("testdata/store.tf"), ConfigVariables: config.Variables{"name": config.StringVariable(f.name)}, Check: f.remote(86400)},
		{ConfigFile: config.StaticFile("testdata/store.tf"), ConfigVariables: config.Variables{"name": config.StringVariable(f.name + "-updated"), "short_ttl": config.IntegerVariable(172800)}, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)}}, Check: f.remote(172800)},
		{ConfigFile: config.StaticFile("testdata/store.tf"), ConfigVariables: config.Variables{"name": config.StringVariable(f.name + "-updated"), "short_ttl": config.IntegerVariable(172800)}, PlanOnly: true},
		{ConfigFile: config.StaticFile("testdata/store.tf"), ConfigVariables: config.Variables{"name": config.StringVariable(f.name + "-updated"), "short_ttl": config.IntegerVariable(0)}, ExpectError: regexp.MustCompile("Invalid Iris JSON configuration")},
		{ConfigFile: config.StaticFile("testdata/store.tf"), ConfigVariables: config.Variables{"name": config.StringVariable(f.name + "-updated"), "short_ttl": config.IntegerVariable(172800)}, Check: f.remote(172800)},
		{ConfigFile: config.StaticFile("testdata/store.tf"), ConfigVariables: config.Variables{"name": config.StringVariable(f.name + "-updated"), "short_ttl": config.IntegerVariable(172800)}, ResourceName: address, ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"database_json", "custom_memory_types_json", "extraction_cadence_json", "long_term_memory_json", "long_term_memory_exclusions_json", "summarization_json", "flush_on_delete"}},
		{ConfigFile: config.StaticFile("testdata/store.tf"), ConfigVariables: config.Variables{"name": config.StringVariable(f.name + "-updated"), "short_ttl": config.IntegerVariable(172800)}, PreConfig: func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			_, err := f.api.Call(ctx, "DELETE", basePath+f.id, map[string][]string{"flush": {"false"}}, nil)
			if err != nil {
				t.Fatal(err)
			}
		}, Check: f.remote(172800)},
	}))
}
func TestAccResourceIrisAgentMemory_Replacement(t *testing.T) {
	f := newFixture(t)
	var previous string
	resource.Test(t, f.testCase([]resource.TestStep{
		{ConfigFile: config.StaticFile("testdata/store.tf"), ConfigVariables: config.Variables{"name": config.StringVariable(f.name)}, Check: func(s *terraform.State) error {
			if err := f.remote(86400)(s); err != nil {
				return err
			}
			previous = f.id
			return nil
		}},
		{ConfigFile: config.StaticFile("testdata/store.tf"), ConfigVariables: config.Variables{"name": config.StringVariable(f.name), "field_type": config.StringVariable("BOOLEAN")}, PlanOnly: true, ExpectNonEmptyPlan: true, ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionDestroyBeforeCreate)}}},
		{ConfigFile: config.StaticFile("testdata/store.tf"), ConfigVariables: config.Variables{"name": config.StringVariable(f.name), "field_type": config.StringVariable("BOOLEAN")}, Check: func(s *terraform.State) error {
			if err := f.remote(86400)(s); err != nil {
				return err
			}
			if f.id == previous {
				return fmt.Errorf("store was not replaced")
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			_, err := f.api.Call(ctx, "GET", basePath+previous, nil, nil)
			if !client.IrisNotFound(err) {
				return fmt.Errorf("superseded store was not deleted")
			}
			return nil
		}},
	}))
}
func TestAccResourceIrisAgentMemory_CustomerModels(t *testing.T) {
	llm, embedding := os.Getenv("AGENT_MEMORY_LLM_API_KEY"), os.Getenv("AGENT_MEMORY_EMBEDDING_API_KEY")
	if llm == "" || embedding == "" {
		t.Skip("requires AGENT_MEMORY_LLM_API_KEY and AGENT_MEMORY_EMBEDDING_API_KEY")
	}
	f := newFixture(t)
	model := func(name, key string) map[string]any {
		return map[string]any{"provider": "openai", "model": name, "credentials": map[string]any{"type": "apiKey", "apiKey": key}}
	}
	data, _ := json.Marshal(map[string]any{"llm": model("gpt-4o-mini", llm), "embedding": model("text-embedding-3-small", embedding), "ttlSeconds": 31536000})
	t.Setenv("TF_VAR_models_json", string(data))
	resource.Test(t, f.testCase([]resource.TestStep{
		{ConfigFile: config.StaticFile("testdata/store.tf"), ConfigVariables: config.Variables{"name": config.StringVariable(f.name)}, Check: f.remote(86400)},
		{ConfigFile: config.StaticFile("testdata/store.tf"), ConfigVariables: config.Variables{"name": config.StringVariable(f.name)}, ResourceName: address, ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"database_json", "custom_memory_types_json", "extraction_cadence_json", "long_term_memory_json", "long_term_memory_exclusions_json", "summarization_json", "flush_on_delete"}},
	}))
}
