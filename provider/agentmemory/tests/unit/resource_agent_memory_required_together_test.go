package unit_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func pairedModelsConfig(llm, embedding, unknown bool) string {
	config := `resource "rediscloud_agent_memory" "example" {
 name="paired-model-store"
 database_id=123
 short_term_ttl_seconds=86400
 long_term_ttl_seconds=31536000
 extraction_cadence_seconds=300
`
	for _, model := range []struct {
		name, model string
		include     bool
	}{
		{"llm", "gpt-4o-mini", llm}, {"embedding", "text-embedding-3-small", embedding},
	} {
		if !model.include {
			continue
		}
		provider, selector, key := `"openai"`, fmt.Sprintf("%q", model.model), `"FAKE-MODEL-KEY"`
		if unknown {
			provider = "terraform_data.value.output"
			selector = provider
			key = provider
		}
		config += fmt.Sprintf(`
%s {
 provider=%s
 model=%s
 credentials {
  type="apiKey"
  api_key=%s
   api_key_version = 1
 }
}
`, model.name, provider, selector, key)
	}
	config += "}\n"
	if unknown {
		config += `resource "terraform_data" "value" { input="FAKE-RESOLVED" }`
	}
	return config
}

func TestAgentMemoryModelBlocksRequiredTogether(t *testing.T) {
	for _, unknown := range []bool{false, true} {
		for _, llm := range []bool{false, true} {
			for _, embedding := range []bool{false, true} {
				t.Run(fmt.Sprintf("llm=%t/embedding=%t/unknown=%t", llm, embedding, unknown), func(t *testing.T) {
					var calls atomic.Int32
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls.Add(1)
						http.Error(w, "validation must not call the API", 500)
					}))
					defer server.Close()
					step := resource.TestStep{Config: testProviderConfig(t, server.URL) + pairedModelsConfig(llm, embedding, unknown), PlanOnly: true, ExpectNonEmptyPlan: true}
					if llm != embedding {
						step.ExpectError = regexp.MustCompile("These attributes must be configured together")
					}
					resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{step}})
					assert.Zero(t, calls.Load(), "configuration validation must run without API calls")
				})
			}
		}
	}
}

func TestAgentMemoryPartialModelBlockRemovalPreservesStore(t *testing.T) {
	for _, tc := range []struct {
		name           string
		llm, embedding bool
		pattern        string
	}{
		{name: "embedding removed", llm: true, pattern: "These attributes must be configured together"},
		{name: "llm removed", embedding: true, pattern: "These attributes must be configured together"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			creates, patches, deletes := 0, 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
					creates++
					_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
				case r.Method == http.MethodPatch:
					patches++
					http.Error(w, "rejected removal must not update the API", 400)
				case r.Method == http.MethodGet && r.URL.Path == "/tasks/fixture-task":
					_, _ = w.Write([]byte(`{"taskId":"fixture-task","status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
				case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1":
					writeStoreResponseWithModelConfig(t, w, "paired-model-store", "gpt-4o-mini", "text-embedding-3-small")
				case r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-1":
					deletes++
					_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			prefix := testProviderConfig(t, server.URL)
			initial := prefix + pairedModelsConfig(true, true, false)
			check := func(state *terraform.State) error {
				mu.Lock()
				defer mu.Unlock()
				if creates != 1 || patches != 0 || deletes != 0 {
					return fmt.Errorf("rejected change mutated store: create/update/delete=%d/%d/%d", creates, patches, deletes)
				}
				if state.RootModule().Resources["rediscloud_agent_memory.example"].Primary.Tainted {
					return fmt.Errorf("store tainted")
				}
				return resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1")(state)
			}
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
				{Config: initial, Check: check},
				{Config: prefix + pairedModelsConfig(tc.llm, tc.embedding, false), ExpectError: regexp.MustCompile(tc.pattern)},
				{Config: initial, Check: check},
				{Config: initial, PlanOnly: true},
			}})
			mu.Lock()
			defer mu.Unlock()
			assert.Equal(t, 1, creates)
			assert.Zero(t, patches)
			assert.Equal(t, 1, deletes, "only explicit fixture cleanup may delete the store")
		})
	}
}
