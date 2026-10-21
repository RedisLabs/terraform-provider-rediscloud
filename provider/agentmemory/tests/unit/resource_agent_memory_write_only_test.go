package unit_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

var writeOnlyTestSecrets = []string{"FAKE-WO-LLM-FIRST", "FAKE-WO-LLM-ROTATED", "FAKE-WO-EMBED-FIRST", "FAKE-WO-EMBED-ROTATED"}

type writeOnlyPlanCheck struct{}

func (writeOnlyPlanCheck) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	data, err := json.Marshal(req.Plan)
	if err != nil {
		resp.Error = err
		return
	}
	for _, secret := range writeOnlyTestSecrets {
		if strings.Contains(string(data), secret) {
			resp.Error = fmt.Errorf("ephemeral model secret persisted in plan JSON")
			return
		}
	}
}

func writeOnlyModelConfig(llmVersion, embeddingVersion, ttl int) string {
	config := agentMemoryModelConfig("wo-store", "gpt-4o-mini", "text-embedding-3-small", "LLM_PLACEHOLDER", "EMBED_PLACEHOLDER")
	config = strings.Replace(config, "api_key = \"LLM_PLACEHOLDER\"\n   api_key_version = 1", fmt.Sprintf("api_key = var.llm_key\napi_key_version = %d", llmVersion), 1)
	config = strings.Replace(config, "api_key = \"EMBED_PLACEHOLDER\"\n   api_key_version = 1", fmt.Sprintf("api_key = var.embedding_key\napi_key_version = %d", embeddingVersion), 1)
	config = strings.Replace(config, "long_term_ttl_seconds      = 31536000", fmt.Sprintf("long_term_ttl_seconds      = %d", ttl), 1)
	return `
variable "llm_key" {
 type = string
 sensitive = true
 ephemeral = true
}
variable "embedding_key" {
 type = string
 sensitive = true
 ephemeral = true
}
` + config
}

func TestAgentMemoryWriteOnlyCredentialsLifecycle(t *testing.T) {
	for _, scenario := range []string{"create", "create read recovery"} {
		t.Run(scenario, func(t *testing.T) {
			t.Setenv("TF_VAR_llm_key", writeOnlyTestSecrets[0])
			t.Setenv("TF_VAR_embedding_key", writeOnlyTestSecrets[2])
			var mu sync.Mutex
			creates, deletes, patches := 0, 0, 0
			ttl := 31536000
			var observed [2][]string
			invalidRead := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodPost && r.URL.Path == "/memory-stores", r.Method == http.MethodPatch && r.URL.Path == "/memory-stores/store-1":
					var payload map[string]any
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
						t.Error(err)
						w.WriteHeader(400)
						return
					}
					llm, _ := payload["llm"].(map[string]any)
					memory, _ := payload["longTermMemory"].(map[string]any)
					embedding, _ := memory["embedding"].(map[string]any)
					for index, model := range []map[string]any{llm, embedding} {
						if model == nil {
							continue
						}
						credentials, _ := model["credentials"].(map[string]any)
						key, _ := credentials["apiKey"].(string)
						if key == "" {
							t.Error("model update sent empty credential")
							w.WriteHeader(400)
							return
						}
						observed[index] = append(observed[index], key)
					}
					if value, ok := memory["ttlSeconds"].(float64); ok {
						ttl = int(value)
					}
					if r.Method == http.MethodPost {
						creates++
						invalidRead = scenario == "create read recovery"
					} else {
						patches++
					}
					_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
				case r.Method == http.MethodGet && r.URL.Path == "/tasks/fixture-task":
					_, _ = w.Write([]byte(`{"taskId":"fixture-task","status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
				case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1":
					if invalidRead {
						invalidRead = false
						_, _ = w.Write([]byte(`{"storeId":"store-1","databaseId":"invalid"}`))
						return
					}
					// This endpoint returns metadata only, as the real API does for model credentials.
					response := httptest.NewRecorder()
					writeStoreResponseWithModelConfig(t, response, "wo-store", "gpt-4o-mini", "text-embedding-3-small")
					var store map[string]any
					if err := json.Unmarshal(response.Body.Bytes(), &store); err != nil {
						t.Error(err)
						w.WriteHeader(500)
						return
					}
					store["longTermMemory"].(map[string]any)["ttlSeconds"] = ttl
					_ = json.NewEncoder(w).Encode(store)
				case r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-1":
					deletes++
					_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			prefix := testProviderConfig(t, server.URL)
			config := func(lv, ev, ttl int) string { return prefix + writeOnlyModelConfig(lv, ev, ttl) }
			check := func(llmCount, embeddingCount int, lastLLM, lastEmbedding string) resource.TestCheckFunc {
				return func(state *terraform.State) error {
					mu.Lock()
					defer mu.Unlock()
					if creates != 1 || deletes != 0 {
						return fmt.Errorf("store replaced: creates=%d deletes=%d", creates, deletes)
					}
					if len(observed[0]) != llmCount || len(observed[1]) != embeddingCount {
						return fmt.Errorf("unexpected credential sends: llm=%d embedding=%d", len(observed[0]), len(observed[1]))
					}
					if observed[0][llmCount-1] != lastLLM || observed[1][embeddingCount-1] != lastEmbedding {
						return fmt.Errorf("incorrect credential sent to API")
					}
					data, err := json.Marshal(state)
					if err != nil {
						return err
					}
					for _, secret := range writeOnlyTestSecrets {
						if strings.Contains(string(data), secret) {
							return fmt.Errorf("write-only secret persisted in state")
						}
					}
					rs := state.RootModule().Resources["rediscloud_agent_memory.example"]
					if rs.Primary.Tainted {
						return fmt.Errorf("store tainted")
					}
					return resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1")(state)
				}
			}
			planChecks := resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{writeOnlyPlanCheck{}}}
			firstCount := 1
			var steps []resource.TestStep
			steps = append(steps,
				resource.TestStep{Config: config(1, 1, 31536000), ConfigPlanChecks: planChecks, Check: check(firstCount, firstCount, writeOnlyTestSecrets[0], writeOnlyTestSecrets[2])},
				resource.TestStep{Config: config(1, 1, 31536000), PlanOnly: true, ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPostRefresh: []plancheck.PlanCheck{writeOnlyPlanCheck{}}}},
				// A new ephemeral value alone cannot trigger rotation: its version must change.
				resource.TestStep{PreConfig: func() {
					t.Setenv("TF_VAR_llm_key", writeOnlyTestSecrets[1])
					t.Setenv("TF_VAR_embedding_key", writeOnlyTestSecrets[3])
				}, Config: config(1, 1, 31536000), PlanOnly: true, ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPostRefresh: []plancheck.PlanCheck{writeOnlyPlanCheck{}}}},
				resource.TestStep{Config: config(2, 1, 31536000), ConfigPlanChecks: planChecks, Check: check(firstCount+1, firstCount, writeOnlyTestSecrets[1], writeOnlyTestSecrets[2])},
				resource.TestStep{Config: config(2, 1, 172800), ConfigPlanChecks: planChecks, Check: check(firstCount+1, firstCount, writeOnlyTestSecrets[1], writeOnlyTestSecrets[2])},
				resource.TestStep{Config: config(2, 2, 172800), ConfigPlanChecks: planChecks, Check: check(firstCount+1, firstCount+1, writeOnlyTestSecrets[1], writeOnlyTestSecrets[3])},
				resource.TestStep{Config: config(2, 2, 172800), PlanOnly: true, ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPostRefresh: []plancheck.PlanCheck{writeOnlyPlanCheck{}}}},
				resource.TestStep{Config: prefix + `
removed {
 from = rediscloud_agent_memory.example
 lifecycle {
  destroy = false
 }
}
`},
				resource.TestStep{Config: config(2, 2, 172800), ResourceName: "rediscloud_agent_memory.example", ImportStateId: "store-1", ImportState: true, ImportStateVerify: false, ImportStatePersist: true, ImportStateCheck: func(states []*terraform.InstanceState) error {
					for _, state := range states {
						for name, value := range state.Attributes {
							if strings.HasSuffix(name, "api_key") {
								if value != "" {
									return fmt.Errorf("import populated model credential")
								}
							}
						}
					}
					return nil
				}},
				resource.TestStep{Config: config(2, 2, 172800), ConfigPlanChecks: planChecks, Check: check(firstCount+2, firstCount+2, writeOnlyTestSecrets[1], writeOnlyTestSecrets[3])},
				resource.TestStep{Config: config(2, 2, 172800), PlanOnly: true, ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPostRefresh: []plancheck.PlanCheck{writeOnlyPlanCheck{}}}},
				resource.TestStep{
					Config:           strings.Replace(config(3, 2, 172800), "api_key_version = 3", "api_key_version = terraform_data.version.output", 1) + `resource "terraform_data" "version" { input = 3 }`,
					ConfigPlanChecks: planChecks,
					Check:            check(firstCount+3, firstCount+2, writeOnlyTestSecrets[1], writeOnlyTestSecrets[3]),
				},
			)
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: steps})
			mu.Lock()
			defer mu.Unlock()
			if creates != 1 || deletes != 1 {
				t.Errorf("expected one creation and explicit cleanup, got %d/%d", creates, deletes)
			}
			expectedPatches := 5
			if patches != expectedPatches {
				t.Errorf("expected %d patches, got %d", expectedPatches, patches)
			}
		})
	}
}

func TestAgentMemoryWriteOnlyCredentialsValidation(t *testing.T) {
	for _, block := range []string{"llm", "embedding"} {
		for _, tc := range []struct{ name, attributes, pattern string }{
			{"missing version", `api_key="FAKE-WO"`, "Invalid Attribute Combination"},
			{"missing key", `api_key_version=1`, "Invalid Attribute Combination"},
			{"zero version", "api_key=\"FAKE-WO\"\napi_key_version=0", "at least 1"},
			{"blank key", "api_key=\"  \"\napi_key_version=1", "Missing Agent Memory model API key"},
		} {
			t.Run(block+"/"+tc.name, func(t *testing.T) {
				prefix := testProviderConfig(t, "http://127.0.0.1:1")
				cfg := unknownKeyConfig(block, "REPLACE_ME", "")
				cfg = strings.Replace(cfg, "api_key=REPLACE_ME\n   api_key_version = 1", tc.attributes, 1)
				resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{{Config: prefix + cfg, PlanOnly: true, ExpectError: regexp.MustCompile(tc.pattern)}}})
			})
		}
	}
}

func TestAgentMemoryWriteOnlyVersionUnknownInvalid(t *testing.T) {
	for _, block := range []string{"llm", "embedding"} {
		for _, tc := range []struct{ name, input, expression string }{
			{"zero", "0", "terraform_data.version.output"},
			{"null", "true", "terraform_data.version.output ? null : 1"},
		} {
			t.Run(block+"/"+tc.name, func(t *testing.T) {
				var requests atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					requests.Add(1)
					w.WriteHeader(http.StatusInternalServerError)
				}))
				defer server.Close()
				cfg := unknownKeyConfig(block, `"FAKE-KNOWN"`, fmt.Sprintf(`resource "terraform_data" "version" { input = %s }`, tc.input))
				cfg = strings.Replace(cfg, "api_key=\"FAKE-KNOWN\"\n   api_key_version = 1", "api_key=\"FAKE-KNOWN\"\n   api_key_version = "+tc.expression, 1)
				resource.UnitTest(t, resource.TestCase{
					ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
					Steps:                    []resource.TestStep{{Config: testProviderConfig(t, server.URL) + cfg, ExpectError: regexp.MustCompile("Invalid Attribute Combination|at least 1")}},
				})
				if requests.Load() != 0 {
					t.Fatalf("invalid version reached the API: %d requests", requests.Load())
				}
			})
		}
	}
}
