package unit_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

var emptyStringStatePaths = []string{
	"summarization.0.trigger_strategy",
	"long_term_memory_exclusions.0.built_in_detectors.0.detectors.0.action",
	"long_term_memory_exclusions.0.custom_detectors.0.detectors.0.name",
	"long_term_memory_exclusions.0.custom_detectors.0.detectors.0.action",
	"long_term_memory_exclusions.0.custom_detectors.0.detectors.0.matcher.0.kind",
	"long_term_memory_exclusions.0.custom_detectors.0.detectors.0.matcher.0.regex.0.pattern",
	"custom_memory_types.0.name", "custom_memory_types.0.description",
	"custom_memory_types.0.fields.0.name", "custom_memory_types.0.fields.0.description",
	"custom_memory_types.0.fields.0.type", "custom_memory_types.0.extraction_strategy.0.prompt",
}

// Exact JSON-state checks distinguish a known empty string from null.
func emptyStringStateChecks() []statecheck.StateCheck {
	checks := make([]statecheck.StateCheck, 0, len(emptyStringStatePaths))
	for _, name := range emptyStringStatePaths {
		parts := strings.Split(name, ".")
		path := tfjsonpath.New(parts[0])
		for _, part := range parts[1:] {
			if index, err := strconv.Atoi(part); err == nil {
				path = path.AtSliceIndex(index)
			} else {
				path = path.AtMapKey(part)
			}
		}
		checks = append(checks, statecheck.ExpectKnownValue("rediscloud_agent_memory.example", path, knownvalue.StringExact("")))
	}
	return checks
}

type emptyStringFixture struct {
	t                          *testing.T
	mu                         sync.Mutex
	store                      map[string]any
	creates, patches, deletes  int
	rejectCreate, rejectUpdate bool
}

func (f *emptyStringFixture) serveHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/memory-stores", r.Method == http.MethodPatch && r.URL.Path == "/memory-stores/store-1":
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			f.t.Error(err)
			w.WriteHeader(400)
			return
		}
		if r.Method == http.MethodPost {
			f.creates++
			f.checkCreateStrings(payload)
		} else {
			f.patches++
		}
		if (r.Method == http.MethodPost && f.rejectCreate) || (r.Method == http.MethodPatch && f.rejectUpdate) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"message":"empty config rejected by API"}`))
			return
		}
		if r.Method == http.MethodPost {
			f.store = payload
		} else {
			f.applyUpdate(payload)
		}
		_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
	case r.Method == http.MethodGet && r.URL.Path == "/tasks/fixture-task":
		_, _ = w.Write([]byte(`{"taskId":"fixture-task","status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
	case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1" && f.store != nil:
		f.store["storeId"], f.store["databaseId"], f.store["status"] = "store-1", "123", "READY"
		f.store["extractionStrategy"] = "INSTRUCT"
		if err := json.NewEncoder(w).Encode(f.store); err != nil {
			f.t.Error(err)
		}
	case r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-1":
		f.deletes++
		f.store = nil
		_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
	default:
		http.NotFound(w, r)
	}
}

func (f *emptyStringFixture) checkCreateStrings(payload map[string]any) {
	for _, name := range []string{
		"summarization.triggerStrategy",
		"longTermMemoryExclusions.builtInDetectors.detectors.0.action",
		"longTermMemoryExclusions.customDetectors.detectors.0.name",
		"longTermMemoryExclusions.customDetectors.detectors.0.action",
		"longTermMemoryExclusions.customDetectors.detectors.0.matcher.kind",
		"longTermMemoryExclusions.customDetectors.detectors.0.matcher.regex.pattern",
		"customMemoryTypes.0.name", "customMemoryTypes.0.description",
		"customMemoryTypes.0.fields.0.name", "customMemoryTypes.0.fields.0.description",
		"customMemoryTypes.0.fields.0.type", "customMemoryTypes.0.extractionStrategy.prompt",
	} {
		var value any = payload
		for _, part := range strings.Split(name, ".") {
			if index, err := strconv.Atoi(part); err == nil {
				list, ok := value.([]any)
				if !assert.True(f.t, ok && index < len(list), name) {
					return
				}
				value = list[index]
			} else {
				object, ok := value.(map[string]any)
				if !assert.True(f.t, ok, name) {
					return
				}
				var present bool
				value, present = object[part]
				if !assert.True(f.t, present, "explicit string must be present: %s", name) {
					return
				}
			}
		}
		// Immutable definitions stay empty; mutable fields may contain a supplied value.
		assert.IsType(f.t, "", value, name)
	}
}

func (f *emptyStringFixture) applyUpdate(payload map[string]any) {
	for _, name := range []string{"summarization", "longTermMemoryExclusions"} {
		if value, present := payload[name]; present {
			f.store[name] = value
		}
	}
	if updates, ok := payload["updateCustomMemoryTypeStrategies"].([]any); ok {
		for _, entry := range updates {
			strategy := entry.(map[string]any)
			assert.Equal(f.t, "", strategy["typeName"], "the known empty type name must not be omitted")
			memoryType := f.store["customMemoryTypes"].([]any)[0].(map[string]any)
			current := memoryType["extractionStrategy"].(map[string]any)
			prompt, present := strategy["prompt"]
			assert.True(f.t, present, "explicit update prompt must not be omitted")
			current["prompt"] = prompt
		}
	}
}

func TestAgentMemoryEmptyStringsRoundTrip(t *testing.T) {
	for _, deferred := range []bool{false, true} {
		t.Run(fmt.Sprintf("deferred=%t", deferred), func(t *testing.T) {
			fixture := &emptyStringFixture{t: t}
			server := httptest.NewServer(http.HandlerFunc(fixture.serveHTTP))
			defer server.Close()
			resource.UnitTest(t, resource.TestCase{
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config:     testProviderConfig(t, server.URL),
						ConfigFile: config.StaticFile("testdata/empty_strings.tf"),
						ConfigVariables: config.Variables{
							"value":         config.StringVariable(""),
							"mutable_value": config.StringVariable(""),
							"deferred":      config.BoolVariable(deferred),
						},
						ConfigStateChecks: emptyStringStateChecks(),
					},
					{
						Config:     testProviderConfig(t, server.URL),
						ConfigFile: config.StaticFile("testdata/empty_strings.tf"),
						ConfigVariables: config.Variables{
							"value":         config.StringVariable(""),
							"mutable_value": config.StringVariable(""),
							"deferred":      config.BoolVariable(deferred),
						},
						PlanOnly: true,
					},
					{
						Config:     testProviderConfig(t, server.URL),
						ConfigFile: config.StaticFile("testdata/empty_strings.tf"),
						ConfigVariables: config.Variables{
							"value":         config.StringVariable(""),
							"mutable_value": config.StringVariable("updated"),
							"deferred":      config.BoolVariable(deferred),
						},
						Check: resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"),
					},
					{
						Config:     testProviderConfig(t, server.URL),
						ConfigFile: config.StaticFile("testdata/empty_strings.tf"),
						ConfigVariables: config.Variables{
							"value":         config.StringVariable(""),
							"mutable_value": config.StringVariable(""),
							"deferred":      config.BoolVariable(deferred),
						},
						ConfigStateChecks: emptyStringStateChecks(),
					},
					{
						Config:     testProviderConfig(t, server.URL),
						ConfigFile: config.StaticFile("testdata/empty_strings.tf"),
						ConfigVariables: config.Variables{
							"value":         config.StringVariable(""),
							"mutable_value": config.StringVariable(""),
							"deferred":      config.BoolVariable(deferred),
						},
						PlanOnly: true,
					},
				},
			})
			fixture.mu.Lock()
			defer fixture.mu.Unlock()
			assert.Equal(t, 1, fixture.creates)
			assert.Equal(t, 2, fixture.patches)
			assert.Equal(t, 1, fixture.deletes)
			assert.Nil(t, fixture.store)
		})
	}
}

func TestAgentMemoryEmptyStringsRejectedByAPI(t *testing.T) {
	for _, deferred := range []bool{false, true} {
		t.Run(fmt.Sprintf("deferred=%t", deferred), func(t *testing.T) {
			fixture := &emptyStringFixture{t: t, rejectCreate: true}
			server := httptest.NewServer(http.HandlerFunc(fixture.serveHTTP))
			defer server.Close()
			resource.UnitTest(t, resource.TestCase{
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config:     testProviderConfig(t, server.URL),
						ConfigFile: config.StaticFile("testdata/empty_strings.tf"),
						ConfigVariables: config.Variables{
							"value":         config.StringVariable(""),
							"mutable_value": config.StringVariable(""),
							"deferred":      config.BoolVariable(deferred),
						},
						PlanOnly:           true,
						ExpectNonEmptyPlan: true,
					},
					{
						Config:     testProviderConfig(t, server.URL),
						ConfigFile: config.StaticFile("testdata/empty_strings.tf"),
						ConfigVariables: config.Variables{
							"value":         config.StringVariable(""),
							"mutable_value": config.StringVariable(""),
							"deferred":      config.BoolVariable(deferred),
						},
						ExpectError: regexp.MustCompile(`empty\s+config\s+rejected\s+by\s+API`),
					},
				},
			})
			fixture.mu.Lock()
			defer fixture.mu.Unlock()
			require.Equal(t, 1, fixture.creates)
			assert.Zero(t, fixture.deletes)
			assert.Nil(t, fixture.store)
		})
	}
}

func TestAgentMemoryEmptyStringsUpdateRejectedByAPI(t *testing.T) {
	fixture := &emptyStringFixture{t: t}
	server := httptest.NewServer(http.HandlerFunc(fixture.serveHTTP))
	defer server.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:     testProviderConfig(t, server.URL),
				ConfigFile: config.StaticFile("testdata/empty_strings.tf"),
				ConfigVariables: config.Variables{
					"value":         config.StringVariable(""),
					"mutable_value": config.StringVariable("updated"),
					"deferred":      config.BoolVariable(false),
				},
			},
			{
				Config:     testProviderConfig(t, server.URL),
				ConfigFile: config.StaticFile("testdata/empty_strings.tf"),
				ConfigVariables: config.Variables{
					"value":         config.StringVariable(""),
					"mutable_value": config.StringVariable(""),
					"deferred":      config.BoolVariable(false),
				},
				PreConfig: func() {
					fixture.mu.Lock()
					fixture.rejectUpdate = true
					fixture.mu.Unlock()
				},
				ExpectError: regexp.MustCompile(`empty\s+config\s+rejected\s+by\s+API`),
			},
			{
				Config:     testProviderConfig(t, server.URL),
				ConfigFile: config.StaticFile("testdata/empty_strings.tf"),
				ConfigVariables: config.Variables{
					"value":         config.StringVariable(""),
					"mutable_value": config.StringVariable("updated"),
					"deferred":      config.BoolVariable(false),
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "summarization.0.trigger_strategy", "updated"),
				),
			},
		},
	})
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	assert.Equal(t, 1, fixture.creates)
	assert.Equal(t, 1, fixture.patches)
	assert.Equal(t, 1, fixture.deletes)
	assert.Nil(t, fixture.store)
}
