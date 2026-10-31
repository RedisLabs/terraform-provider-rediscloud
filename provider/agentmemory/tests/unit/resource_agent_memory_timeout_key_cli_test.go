package unit_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAgentMemoryAPIKeyTimeoutLifecycle(t *testing.T) {
	for _, scenario := range []struct {
		name, initial, updated string
		variables              config.Variables
	}{
		{
			name:    "configure-and-change",
			initial: "testdata/lifecycle/api_keys/api_key_timeouts.tf",
			updated: "testdata/lifecycle/api_keys/api_key_timeouts.tf",
			variables: config.Variables{
				"create": config.StringVariable("7m"),
				"read":   config.StringVariable("7m"),
				"delete": config.StringVariable("7m"),
			},
		},
		{
			name:    "add-to-existing",
			initial: "testdata/lifecycle/api_keys/api_key_without_timeouts.tf",
			updated: "testdata/lifecycle/api_keys/api_key_timeouts.tf",
			variables: config.Variables{
				"create": config.StringVariable("7m"),
				"read":   config.StringVariable("7m"),
				"delete": config.StringVariable("7m"),
			},
		},
		{
			name:    "dynamic-presence",
			initial: "testdata/lifecycle/api_keys/api_key_dynamic_timeouts_on.tf",
			updated: "testdata/lifecycle/api_keys/api_key_dynamic_timeouts_off.tf",
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			var mu sync.Mutex
			exists := false
			creates, deletes := 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1/api-keys":
					keys := []map[string]any{}
					if exists {
						key := map[string]any{"apiKeyId": "key-1", "keyName": "runtime", "obfuscatedToken": "masked", "createdAt": 123}

						keys = append(keys, key)
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"apiKeys": keys})
				case r.Method == http.MethodPost && r.URL.Path == "/memory-stores/store-1/api-keys":
					if exists {
						t.Error("replacement or duplicate creation was attempted")
					}
					exists = true
					creates++
					_, _ = w.Write([]byte(`{"apiKey":"FAKE-CREATED-SECRET"}`))
				case r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-1/api-keys/key-1":
					exists = false
					deletes++
					w.WriteHeader(http.StatusNoContent)
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			testProviderConfig(t, server.URL)
			timeoutCheck := resource.TestCheckResourceAttr("rediscloud_agent_memory_api_key.runtime", "timeouts.0.read", "7m")
			if scenario.name == "dynamic-presence" {
				timeoutCheck = resource.TestCheckNoResourceAttr("rediscloud_agent_memory_api_key.runtime", "timeouts.0.read")
			}

			check := func(state *terraform.State) error {
				mu.Lock()
				defer mu.Unlock()
				key := state.RootModule().Resources["rediscloud_agent_memory_api_key.runtime"]
				if key == nil || key.Primary.ID != "key-1" || key.Primary.Attributes["name"] != "runtime" || key.Primary.Attributes["api_key"] != "FAKE-CREATED-SECRET" || creates != 1 || deletes != 0 {
					return errors.New("metadata recovery changed the tracked key or its original secret")
				}
				return nil
			}
			resource.UnitTest(t, resource.TestCase{
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				Steps: []resource.TestStep{
					{
						ConfigFile: config.StaticFile(scenario.initial),
						Check:      check,
					},
					{

						ConfigFile:      config.StaticFile(scenario.updated),
						ConfigVariables: scenario.variables,

						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{
								plancheck.ExpectKnownValue("rediscloud_agent_memory_api_key.runtime", tfjsonpath.New("id"), knownvalue.StringExact("key-1")),
							},
						},

						Check: resource.ComposeTestCheckFunc(check, timeoutCheck),
					},
					{
						ConfigFile:      config.StaticFile(scenario.updated),
						ConfigVariables: scenario.variables,
						PlanOnly:        true,
					},
				},
				CheckDestroy: func(_ *terraform.State) error {
					mu.Lock()
					defer mu.Unlock()
					if exists || creates != 1 || deletes != 1 {
						return errors.New("local API key fixture was not cleaned up exactly once")
					}
					return nil
				},
			})
		})
	}
}
