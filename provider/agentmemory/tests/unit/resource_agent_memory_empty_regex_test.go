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

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func emptyRegexConfig(expression, dependency string, enabled bool) string {
	return dependency + fmt.Sprintf(`
resource "rediscloud_agent_memory" "example" {
 name="empty-regex-store"
 database_id=123
 short_term_ttl_seconds=86400
 long_term_ttl_seconds=31536000
 extraction_cadence_seconds=300
 long_term_memory_exclusions {
  enabled=true
  custom_detectors {
   enabled=true
   detectors {
    name="tenant-ticket"
    enabled=%t
    action="redact"
    matcher {
     kind="regex"
     regex { pattern=%s }
    }
   }
  }
 }
}
`, enabled, expression)
}

func TestAgentMemoryEmptyRegexValidation(t *testing.T) {
	cases := []struct {
		name, pattern string
		nullable      bool
	}{
		{"wildcard-star", ".*", true},
		{"literal-star", "a*", true},
		{"optional-literal", "a?", true},
		{"zero-repetition", "a{0}", true},
		{"optional-range", "a{0,3}", true},
		{"empty-group", "(?:)", true},
		{"anchors", "^$", true},
		{"text-anchors", `\A\z`, true},
		{"empty-alternative", "(?:ticket|)", true},
		{"optional-digits", "[0-9]*", true},
		{"flags-optional", "(?i)a*", true},
		{"dotall-optional", "(?s:.*)", true},
		{"literal", "ticket", false},
		{"mandatory-character", "a+", false},
		{"positive-range", "a{1,3}", false},
		{"ticket-pattern", "ticket-[0-9]+", false},
		{"mandatory-alternative", "(?:ticket|account)", false},
		{"escaped-metacharacters", `\.\*`, false},
		{"quoted-metacharacters", `\Q.*\E`, false},
		{"unicode-class", `\p{Greek}+`, false},
		// This RE2 construct is unsupported by Go; acceptance stays with the API.
		{"re2-byte-class-deferred", `\C`, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				http.Error(w, "plan must not call HTTP", 400)
			}))
			defer server.Close()
			defer func() {
				if calls.Load() != 0 {
					t.Error("plan reached HTTP")
				}
			}()
			config := testProviderConfig(t, server.URL) + emptyRegexConfig(fmt.Sprintf("%q", c.pattern), "", true)
			step := resource.TestStep{Config: config, PlanOnly: true, ExpectNonEmptyPlan: true}
			if c.nullable {
				step.ExpectError = regexp.MustCompile("Regex matches empty text")
			}
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{step}})
		})
	}
}

func TestAgentMemoryEmptyRegexBoundaries(t *testing.T) {
	for _, c := range []struct {
		name, expression, errorPattern string
		disabled                       bool
	}{
		{name: "empty-pattern", expression: `""`, errorPattern: "length"},
		{name: "former-maximum-length", expression: fmt.Sprintf("%q", strings.Repeat("a", 512))},
		{name: "above-former-maximum-deferred-to-api", expression: fmt.Sprintf("%q", strings.Repeat("a", 513))},
		{name: "null-pattern", expression: "null", errorPattern: "Invalid Attribute Combination"},
		{name: "disabled-detector", expression: `".*"`, errorPattern: "Regex matches empty text", disabled: true},
		// This validator does not impose Go's syntax rules on the API's RE2 compiler.
		{name: "syntax-validation-deferred", expression: `"["`},
	} {
		t.Run(c.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				http.Error(w, "plan must not call HTTP", 400)
			}))
			defer server.Close()
			defer func() {
				if calls.Load() != 0 {
					t.Error("plan reached HTTP")
				}
			}()
			config := testProviderConfig(t, server.URL) + emptyRegexConfig(c.expression, "", !c.disabled)
			step := resource.TestStep{Config: config, PlanOnly: true, ExpectNonEmptyPlan: true}
			if c.errorPattern != "" {
				step.ExpectError = regexp.MustCompile(c.errorPattern)
			}
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{step}})
		})
	}
}

func TestAgentMemoryEmptyRegexUnknown(t *testing.T) {
	for _, pattern := range []string{".*", "a?", "(?:ticket|)"} {
		t.Run(pattern, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				http.Error(w, "empty match must not reach HTTP", 400)
			}))
			defer server.Close()
			defer func() {
				if calls.Load() != 0 {
					t.Error("resolved empty-match pattern reached HTTP")
				}
			}()
			prefix := testProviderConfig(t, server.URL)
			dependency := fmt.Sprintf(`resource "terraform_data" "pattern" { input=%q }`, pattern)
			config := prefix + emptyRegexConfig("tostring(terraform_data.pattern.output)", dependency, true)
			restored := prefix + emptyRegexConfig(`"ticket-[0-9]+"`, dependency, true)
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
				{Config: config, PlanOnly: true, ExpectNonEmptyPlan: true},
				{Config: config, ExpectError: regexp.MustCompile("Regex matches empty text")},
				{Config: restored, PlanOnly: true, ExpectNonEmptyPlan: true},
			}})
		})
	}
}

func TestAgentMemoryEmptyRegexLifecycle(t *testing.T) {
	var mu sync.Mutex
	var store map[string]any
	creates, patches, deletes := 0, 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/memory-stores" || r.Method == http.MethodPatch && r.URL.Path == "/memory-stores/store-1":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
			exclusions, _ := payload["longTermMemoryExclusions"].(map[string]any)
			custom, _ := exclusions["customDetectors"].(map[string]any)
			items, ok := custom["detectors"].([]any)
			if !ok || len(items) != 1 {
				t.Error("expected one detector")
				w.WriteHeader(400)
				return
			}
			item, _ := items[0].(map[string]any)
			matcher, _ := item["matcher"].(map[string]any)
			regex, _ := matcher["regex"].(map[string]any)
			value, ok := regex["pattern"].(string)
			expected := "ticket-[0-9]+"
			if r.Method == http.MethodPatch {
				expected = "account-[0-9]+"
			}
			if !ok || value != expected {
				t.Errorf("unexpected pattern reached HTTP: %q", value)
				w.WriteHeader(400)
				return
			}
			if r.Method == http.MethodPost {
				creates++
				store = payload
			} else {
				patches++
				for k, v := range payload {
					store[k] = v
				}
			}
			_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/fixture-task":
			_, _ = w.Write([]byte(`{"taskId":"fixture-task","status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1":
			if store == nil {
				http.NotFound(w, r)
				return
			}
			store["storeId"] = "store-1"
			store["databaseId"] = "123"
			store["status"] = "READY"
			store["extractionStrategy"] = "INSTRUCT"
			store["createdAt"] = "2026-10-16T12:00:00Z"
			store["endpoint"] = "https://memory.example.test"
			store["endpoints"] = []map[string]any{{"url": "https://memory.example.test", "provider": "AWS", "region": "us-east-1", "egressIps": []string{}, "isAccessible": true}}
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
	defer func() {
		mu.Lock()
		defer mu.Unlock()
		if creates != 1 || patches != 1 || deletes != 1 || store != nil {
			t.Errorf("fixture cleanup/counts failed: creates=%d patches=%d deletes=%d remaining=%t", creates, patches, deletes, store != nil)
		}
	}()
	prefix := testProviderConfig(t, server.URL)
	config := func(value string) string {
		return prefix + emptyRegexConfig("tostring(terraform_data.pattern.output)", fmt.Sprintf(`resource "terraform_data" "pattern" { input=%q }`, value), true)
	}
	initial, changed, invalid := config("ticket-[0-9]+"), config("account-[0-9]+"), config(".*")
	restored := prefix + emptyRegexConfig(`"account-[0-9]+"`, `resource "terraform_data" "pattern" { input=".*" }`, true)
	resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
		{Config: initial},
		{Config: changed, Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"), resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "long_term_memory_exclusions.custom_detectors.detectors.0.matcher.regex.pattern", "account-[0-9]+"))},
		{Config: changed, PlanOnly: true},
		{Config: invalid, PlanOnly: true, ExpectNonEmptyPlan: true},
		{Config: invalid, ExpectError: regexp.MustCompile("Regex matches empty text")},
		// Use a valid pattern for cleanup after the rejected update.
		{Config: restored, PlanOnly: true},
	}})
}
