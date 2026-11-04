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

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func reservedDetectorConfig(name, dependency, builtinID string) string {
	builtin := ""
	if builtinID != "" {
		builtin = fmt.Sprintf("built_in_detectors {\nenabled=true\ndetectors {\nid=%q\nenabled=true\naction=\"redact\"\n}\n}\n", builtinID)
	}
	return dependency + fmt.Sprintf(`
resource "rediscloud_agent_memory" "example" {
 name="reserved-detector-store"
 database_id=123
 short_term_ttl_seconds=86400
 long_term_ttl_seconds=31536000
 extraction_cadence_seconds=300
 long_term_memory_exclusions {
  enabled=true
  %s
  custom_detectors {
   enabled=true
   detectors {
    name=%s
    enabled=true
    action="redact"
    matcher {
     kind="regex"
     regex { pattern="ticket-[0-9]+" }
    }
   }
  }
 }
}
`, builtin, name)
}

func TestAgentMemoryDetectorNameValidationByAPI(t *testing.T) {
	cases := []struct{ name, expression, builtinID, errorPattern string }{
		{name: "custom-name", expression: `"preference"`},
		{name: "memory-type-name-not-detector-id", expression: `"semantic"`},
		{name: "maximum-length", expression: fmt.Sprintf("%q", strings.Repeat("a", 64))},
		{name: "above-former-length-deferred-to-api", expression: fmt.Sprintf("%q", strings.Repeat("a", 65))},
		{name: "empty-name", expression: `""`},
		{name: "null-name", expression: "null", errorPattern: "Missing Configuration for Required Attribute|Missing required argument"},
		{name: "format-deferred-to-api", expression: `"1preference"`},
	}
	for _, name := range []string{"credit-card", "email", "ip-address", "phone", "us-ssn"} {
		cases = append(cases,
			struct{ name, expression, builtinID, errorPattern string }{name: "reserved-" + name, expression: fmt.Sprintf("%q", name)},
			struct{ name, expression, builtinID, errorPattern string }{name: "custom-suffix-" + name, expression: fmt.Sprintf("%q", name+"_custom")},
			// Known catalogue entries and colliding custom names are checked by the API.
			struct{ name, expression, builtinID, errorPattern string }{name: "builtin-selection-" + name, expression: `"ticket"`, builtinID: name},
		)
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
			builtinID := c.builtinID
			config := testProviderConfig(t, server.URL) + reservedDetectorConfig(c.expression, "", builtinID)
			step := resource.TestStep{Config: config, PlanOnly: true, ExpectNonEmptyPlan: true}
			if c.errorPattern != "" {
				step.ExpectError = regexp.MustCompile(c.errorPattern)
			}
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{step}})
		})
	}
}

func TestAgentMemoryDetectorNameLifecycleByAPI(t *testing.T) {
	for _, name := range []string{"preference", "email_custom", "email"} {
		t.Run(name, func(t *testing.T) {
			var mu sync.Mutex
			var store map[string]any
			creates, deletes := 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
					if err := json.NewDecoder(r.Body).Decode(&store); err != nil {
						t.Error(err)
						w.WriteHeader(400)
						return
					}
					exclusions, _ := store["longTermMemoryExclusions"].(map[string]any)
					custom, _ := exclusions["customDetectors"].(map[string]any)
					items, ok := custom["detectors"].([]any)
					if !ok || len(items) != 1 {
						t.Error("expected one custom detector")
						w.WriteHeader(400)
						return
					}
					item, ok := items[0].(map[string]any)
					if !ok || item["name"] != name {
						t.Error("resolved valid name was not preserved")
						w.WriteHeader(400)
						return
					}
					creates++
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
					t.Errorf("fixture cleanup failed: creates=%d deletes=%d remaining=%t", creates, deletes, store != nil)
				}
			}()
			testProviderConfig(t, server.URL)
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
				{
					ConfigFile:      config.StaticFile("testdata/lifecycle/detectors/detector_deferred_name.tf"),
					ConfigVariables: config.Variables{"name": config.StringVariable(name)},
					Check:           resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"), resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "long_term_memory_exclusions.0.custom_detectors.0.detectors.0.name", name)),
				},
				{
					ConfigFile:      config.StaticFile("testdata/lifecycle/detectors/detector_deferred_name.tf"),
					ConfigVariables: config.Variables{"name": config.StringVariable(name)},
					PlanOnly:        true,
				},
			}})
		})
	}
}

func TestAgentMemoryDetectorNameDisabledByAPI(t *testing.T) {
	for _, c := range []struct{ name, from, to string }{
		{"detector", "    enabled=true", "    enabled=false"},
		{"group", "  custom_detectors {\n   enabled=true", "  custom_detectors {\n   enabled=false"},
		{"master", " long_term_memory_exclusions {\n  enabled=true", " long_term_memory_exclusions {\n  enabled=false"},
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
					t.Error("disabled reserved detector reached HTTP")
				}
			}()
			config := testProviderConfig(t, server.URL) + reservedDetectorConfig(`"email"`, "", "")
			if !strings.Contains(config, c.from) {
				t.Fatal("disabled fixture target not found")
			}
			config = strings.Replace(config, c.from, c.to, 1)
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
				{Config: config, PlanOnly: true, ExpectNonEmptyPlan: true},
			}})
		})
	}
}
