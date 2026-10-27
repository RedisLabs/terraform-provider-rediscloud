package unit_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func duplicateDetectorBlock(name, pattern, action string, enabled bool) string {
	return fmt.Sprintf(`
 detectors {
  name=%s
  enabled=%t
  action=%q
  matcher {
   kind="regex"
   regex { pattern=%q }
  }
 }
`, name, enabled, action, pattern)
}

func duplicateDetectorConfig(detectors, dependency, extra string, masterEnabled, groupEnabled bool) string {
	return dependency + fmt.Sprintf(`
resource "rediscloud_agent_memory" "example" {
 name="detector-names-store"
 database_id=123
 short_term_ttl_seconds=86400
 long_term_ttl_seconds=31536000
 extraction_cadence_seconds=300
 long_term_memory_exclusions {
  enabled=%t
  custom_detectors {
   enabled=%t
   %s
  }
 }
 %s
}
`, masterEnabled, groupEnabled, detectors, extra)
}

func TestAgentMemoryCustomDetectorNameUniqueness(t *testing.T) {
	first := duplicateDetectorBlock(`"ticket"`, "ticket-[0-9]+", "redact", true)
	same := duplicateDetectorBlock(`"ticket"`, "ticket-[0-9]+", "redact", true)
	different := duplicateDetectorBlock(`"ticket"`, "account-[0-9]+", "drop", true)
	disabled := duplicateDetectorBlock(`"ticket"`, "account-[0-9]+", "redact", false)
	unique := duplicateDetectorBlock(`"account"`, "ticket-[0-9]+", "redact", true)
	unknown := duplicateDetectorBlock(`tostring(terraform_data.name.output)`, "other-[0-9]+", "redact", true)
	dependency := `resource "terraform_data" "name" { input="other" }`
	for _, c := range []struct {
		name, detectors, dependency, extra     string
		masterDisabled, groupDisabled, invalid bool
	}{
		{name: "empty-list"},
		{name: "one-detector", detectors: first},
		{name: "unique-names-same-pattern", detectors: first + unique},
		{name: "identical-detectors", detectors: first + same, invalid: true},
		{name: "same-name-different-definition", detectors: first + different, invalid: true},
		{name: "one-disabled", detectors: first + disabled, invalid: true},
		{name: "both-disabled", detectors: disabled + disabled, invalid: true},
		{name: "group-disabled", detectors: first + same, groupDisabled: true, invalid: true},
		{name: "master-disabled", detectors: first + same, masterDisabled: true, invalid: true},
		{name: "unknown-name", detectors: first + unknown, dependency: dependency},
		{name: "known-duplicate-before-unknown", detectors: first + same + unknown, dependency: dependency, invalid: true},
		{name: "known-duplicate-after-unknown", detectors: unknown + first + same, dependency: dependency, invalid: true},
		{name: "unknown-summarization-threshold", detectors: first + same, dependency: `resource "terraform_data" "threshold" { input=20 }`, extra: "summarization {\nenabled=true\nevent_count {\nthreshold=terraform_data.threshold.output\nretain_count=2\n}\n}", invalid: true},
	} {
		t.Run(c.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				http.Error(w, "validation must not call HTTP", 400)
			}))
			defer server.Close()
			defer func() {
				if calls.Load() != 0 {
					t.Error("plan reached HTTP")
				}
			}()
			config := testProviderConfig(t, server.URL) + duplicateDetectorConfig(c.detectors, c.dependency, c.extra, !c.masterDisabled, !c.groupDisabled)
			step := resource.TestStep{Config: config, PlanOnly: true, ExpectNonEmptyPlan: true}
			if c.invalid {
				step.ExpectError = regexp.MustCompile("Duplicate Agent Memory custom detector name")
			}
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{step}})
		})
	}
}

func TestAgentMemoryCustomDetectorNameUnknownDuplicate(t *testing.T) {
	for _, unknownFirst := range []bool{false, true} {
		t.Run(fmt.Sprint(unknownFirst), func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				http.Error(w, "duplicate must not call HTTP", 400)
			}))
			defer server.Close()
			defer func() {
				if calls.Load() != 0 {
					t.Error("resolved duplicate reached HTTP")
				}
			}()
			first := duplicateDetectorBlock(`"ticket"`, "ticket-[0-9]+", "redact", true)
			deferred := duplicateDetectorBlock(`tostring(terraform_data.name.output)`, "other-[0-9]+", "drop", true)
			detectors := first + deferred
			if unknownFirst {
				detectors = deferred + first
			}
			dependency := `resource "terraform_data" "name" { input="ticket" }`
			prefix := testProviderConfig(t, server.URL)
			config := prefix + duplicateDetectorConfig(detectors, dependency, "", true, true)
			restored := prefix + duplicateDetectorConfig(first, dependency, "", true, true)
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
				{Config: config, PlanOnly: true, ExpectNonEmptyPlan: true},
				{Config: config, ExpectError: regexp.MustCompile("Duplicate Agent Memory custom detector name")},
				{Config: restored, PlanOnly: true, ExpectNonEmptyPlan: true},
			}})
		})
	}
}

func TestAgentMemoryCustomDetectorNameLifecycle(t *testing.T) {
	var mu sync.Mutex
	var store map[string]any
	creates, deletes := 0, 0
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
			exclusions, ok := payload["longTermMemoryExclusions"].(map[string]any)
			if !ok {
				t.Error("missing exclusions")
				w.WriteHeader(400)
				return
			}
			custom, ok := exclusions["customDetectors"].(map[string]any)
			if !ok {
				t.Error("missing custom detectors")
				w.WriteHeader(400)
				return
			}
			items, ok := custom["detectors"].([]any)
			if !ok || len(items) != 2 {
				t.Error("expected two detectors")
				w.WriteHeader(400)
				return
			}
			names := map[string]bool{}
			for _, raw := range items {
				item, ok := raw.(map[string]any)
				if !ok {
					t.Error("invalid detector")
					w.WriteHeader(400)
					return
				}
				name, ok := item["name"].(string)
				if !ok || names[name] {
					t.Error("duplicate name reached HTTP")
					w.WriteHeader(400)
					return
				}
				names[name] = true
			}
			if !names["ticket"] || !names["account"] {
				t.Errorf("unexpected detector names: %v", names)
			}
			if r.Method == http.MethodPost {
				creates++
				store = payload
			} else {
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
		if creates != 1 || deletes != 1 || store != nil {
			t.Errorf("fixture not cleaned: creates=%d deletes=%d remaining=%t", creates, deletes, store != nil)
		}
	}()
	prefix := testProviderConfig(t, server.URL)
	first := duplicateDetectorBlock(`"ticket"`, "ticket-[0-9]+", "redact", true)
	config := func(input, expression string) string {
		return prefix + duplicateDetectorConfig(first+duplicateDetectorBlock(expression, "account-[0-9]+", "redact", true), fmt.Sprintf(`resource "terraform_data" "name" { input=%q }`, input), "", true, true)
	}
	expression := `split(":",tostring(terraform_data.name.output))[0]`
	initial, changed := config("account:before", expression), config("account:after", expression)
	duplicate := config("account:after", `"ticket"`)
	deferredDuplicate := config("ticket:after", expression)
	restored := config("ticket:after", `"account"`)
	check := resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"), resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "long_term_memory_exclusions.0.custom_detectors.0.detectors.1.name", "account"))
	resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
		{Config: initial, Check: check},
		{Config: changed, Check: check},
		{Config: changed, PlanOnly: true},
		{Config: duplicate, PlanOnly: true, ExpectError: regexp.MustCompile("Duplicate Agent Memory custom detector name")},
		{Config: deferredDuplicate, PlanOnly: true, ExpectNonEmptyPlan: true},
		{Config: deferredDuplicate, ExpectError: regexp.MustCompile("Duplicate Agent Memory custom detector name")},
		// Use a valid list for cleanup after the rejected update.
		{Config: restored, PlanOnly: true},
	}})
}
