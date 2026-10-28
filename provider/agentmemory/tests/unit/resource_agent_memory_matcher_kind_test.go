package unit_test

import (
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
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func matcherKindConfig(expression, dependency string, withRegex bool) string {
	config := emptyRegexConfig(`"ticket-[0-9]+"`, dependency, true)
	config = strings.Replace(config, `kind="regex"`, "kind="+expression, 1)
	if !withRegex {
		config = strings.Replace(config, `regex { pattern="ticket-[0-9]+" }`, "", 1)
	}
	return config
}

func TestAgentMemoryMatcherKindValidation(t *testing.T) {
	for _, c := range []struct {
		name, expression, dependency, errorPattern string
		regex                                      bool
	}{
		{name: "regex-with-block", expression: `"regex"`, regex: true},
		{name: "regex-without-block", expression: `"regex"`, errorPattern: "Missing custom detector regex"},
		{name: "future-without-block", expression: `"future-kind"`},
		{name: "future-with-regex-block", expression: `"future-kind"`, regex: true},
		{name: "unknown-without-block", expression: "tostring(terraform_data.kind.output)", dependency: `resource "terraform_data" "kind" { input="regex" }`},
		{name: "unknown-with-block", expression: "tostring(terraform_data.kind.output)", dependency: `resource "terraform_data" "kind" { input="regex" }`, regex: true},
		{name: "empty", expression: `""`},
		{name: "null", expression: "null", errorPattern: "Missing Configuration for Required Attribute|Missing required argument"},
	} {
		t.Run(c.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				http.Error(w, "plan must not call API", 500)
			}))
			defer server.Close()
			defer func() { assert.Zero(t, calls.Load()) }()
			step := resource.TestStep{Config: testProviderConfig(t, server.URL) + matcherKindConfig(c.expression, c.dependency, c.regex), PlanOnly: true, ExpectNonEmptyPlan: true}
			if c.errorPattern != "" {
				step.ExpectError = regexp.MustCompile(c.errorPattern)
			}
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{step}})
		})
	}
}

func TestAgentMemoryMatcherKindResolvedInvalid(t *testing.T) {
	for _, c := range []struct{ name, input, errorPattern string }{
		{"regex-without-block", `"regex"`, "Missing custom detector regex"},
		{"null", "null", "Missing Configuration for Required Attribute|Missing required argument"},
	} {
		t.Run(c.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				http.Error(w, "invalid kind must not reach API", 500)
			}))
			defer server.Close()
			defer func() { assert.Zero(t, calls.Load()) }()
			dependency := fmt.Sprintf(`resource "terraform_data" "kind" { input={value=%s} }`, c.input)
			prefix := testProviderConfig(t, server.URL)
			config := prefix + matcherKindConfig("tostring(terraform_data.kind.output.value)", dependency, false)
			restored := prefix + matcherKindConfig(`"regex"`, dependency, true)
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
				{Config: config, PlanOnly: true, ExpectNonEmptyPlan: true},
				{Config: config, ExpectError: regexp.MustCompile(c.errorPattern)},
				{Config: restored, PlanOnly: true, ExpectNonEmptyPlan: true},
			}})
		})
	}
}

func TestAgentMemoryMatcherKindByAPI(t *testing.T) {
	for _, unknown := range []bool{false, true} {
		t.Run(fmt.Sprintf("unknown=%t", unknown), func(t *testing.T) {
			var mu sync.Mutex
			var store map[string]any
			creates, patches, deletes := 0, 0, 0
			var observed []string
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
					items := listLimitEntries(payload, "custom-detectors")
					if len(items) != 1 {
						t.Error("missing detector")
						w.WriteHeader(400)
						return
					}
					item, _ := items[0].(map[string]any)
					matcher, _ := item["matcher"].(map[string]any)
					kind, _ := matcher["kind"].(string)
					observed = append(observed, kind)
					if r.Method == http.MethodPost {
						creates++
					} else {
						patches++
					}
					if kind == "regex" {
						regex, _ := matcher["regex"].(map[string]any)
						assert.Equal(t, "ticket-[0-9]+", regex["pattern"])
					} else {
						_, present := matcher["regex"]
						assert.False(t, present, "unused regex must not be added")
					}
					if kind == "rejected-kind" {
						w.WriteHeader(400)
						_, _ = w.Write([]byte(`{"message":"matcher kind rejected by API"}`))
						return
					}
					// A future kind needing no additional fields can fit the existing SDK shape.
					if store == nil {
						store = payload
					} else {
						store["longTermMemoryExclusions"] = payload["longTermMemoryExclusions"]
					}
					_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
				case r.Method == http.MethodGet && r.URL.Path == "/tasks/fixture-task":
					_, _ = w.Write([]byte(`{"taskId":"fixture-task","status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
				case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1" && store != nil:
					store["storeId"] = "store-1"
					store["databaseId"] = "123"
					store["status"] = "READY"
					store["extractionStrategy"] = "INSTRUCT"
					_ = json.NewEncoder(w).Encode(store)
				case r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-1":
					deletes++
					store = nil
					_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			config := func(kind string, regex bool) string {
				expression, dependency := fmt.Sprintf("%q", kind), ""
				if unknown {
					dependency = fmt.Sprintf(`resource "terraform_data" "kind" { input=%q }`, kind)
					expression = "tostring(terraform_data.kind.output)"
				}
				return testProviderConfig(t, server.URL) + matcherKindConfig(expression, dependency, regex)
			}
			check := func(kind string) resource.TestCheckFunc {
				return func(state *terraform.State) error {
					mu.Lock()
					defer mu.Unlock()
					if creates != 1 || deletes != 0 || state.RootModule().Resources["rediscloud_agent_memory.example"].Primary.Tainted {
						return fmt.Errorf("store replaced or tainted: creates=%d deletes=%d", creates, deletes)
					}
					checks := []resource.TestCheckFunc{resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"), resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "long_term_memory_exclusions.0.custom_detectors.0.detectors.0.matcher.0.kind", kind)}
					if kind != "regex" {
						checks = append(checks, resource.TestCheckNoResourceAttr("rediscloud_agent_memory.example", "long_term_memory_exclusions.0.custom_detectors.0.detectors.0.matcher.0.regex.0.pattern"))
					}
					return resource.ComposeTestCheckFunc(checks...)(state)
				}
			}
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
				{Config: config("future-kind", false), Check: check("future-kind")},
				{Config: config("regex", true), Check: check("regex")},
				{Config: config("future-kind", false), Check: check("future-kind")},
				{Config: config("rejected-kind", false), ExpectError: regexp.MustCompile("Failed to update Agent Memory service")},
				{Config: config("future-kind", false), Check: check("future-kind")},
				{Config: config("future-kind", false), PlanOnly: true},
			}})
			mu.Lock()
			defer mu.Unlock()
			assert.Equal(t, []string{"future-kind", "regex", "future-kind", "rejected-kind"}, observed)
			assert.Equal(t, 1, creates)
			assert.Equal(t, 3, patches)
			assert.Equal(t, 1, deletes)
			assert.Nil(t, store)
		})
	}
}

func TestAgentMemoryMatcherKindCreateRejectedByAPI(t *testing.T) {
	var mu sync.Mutex
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		calls++
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/memory-stores", r.URL.Path)
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		items := listLimitEntries(payload, "custom-detectors")
		if len(items) != 1 {
			t.Error("missing detector")
		} else {
			item, _ := items[0].(map[string]any)
			matcher, _ := item["matcher"].(map[string]any)
			assert.Equal(t, "rejected-kind", matcher["kind"])
			_, present := matcher["regex"]
			assert.False(t, present)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"message":"matcher kind rejected by API"}`))
	}))
	defer server.Close()
	resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{{Config: testProviderConfig(t, server.URL) + matcherKindConfig(`"rejected-kind"`, "", false), ExpectError: regexp.MustCompile("Failed to create Agent Memory service")}}})
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, calls)
}

func TestAgentMemoryMatcherRegexPatternPresence(t *testing.T) {
	for _, c := range []struct{ name, block, dependency string }{
		{name: "empty-block", block: `regex {}`},
		{name: "null-pattern", block: `regex { pattern=null }`},
		{name: "unknown-null-pattern", block: `regex { pattern=tostring(terraform_data.pattern.output.value) }`, dependency: `resource "terraform_data" "pattern" { input={value=null} }`},
	} {
		t.Run(c.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				http.Error(w, "missing pattern must not reach API", 500)
			}))
			defer server.Close()
			defer func() { assert.Zero(t, calls.Load()) }()
			prefix := testProviderConfig(t, server.URL)
			config := matcherKindConfig(`"regex"`, c.dependency, true)
			config = strings.Replace(config, `regex { pattern="ticket-[0-9]+" }`, c.block, 1)
			step := resource.TestStep{Config: prefix + config, PlanOnly: true, ExpectNonEmptyPlan: true, ExpectError: regexp.MustCompile("Invalid Attribute Combination")}
			steps := []resource.TestStep{step}
			if c.dependency != "" {
				steps = []resource.TestStep{{Config: prefix + config, PlanOnly: true, ExpectNonEmptyPlan: true}, {Config: prefix + config, ExpectError: regexp.MustCompile("Invalid Attribute Combination")}, {Config: prefix + matcherKindConfig(`"regex"`, c.dependency, true), PlanOnly: true, ExpectNonEmptyPlan: true}}
			}
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: steps})
		})
	}
}
