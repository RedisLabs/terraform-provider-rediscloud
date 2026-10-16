package agentmemory_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAgentMemoryAPIKeyTimeoutLifecycle(t *testing.T) {
	for _, scenario := range []string{"configure-and-change", "add-to-existing"} {
		t.Run(scenario, func(t *testing.T) {
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
			prefix := testProviderConfig(t, server.URL)
			config := prefix + `resource "rediscloud_agent_memory_api_key" "runtime" {
 store_id="store-1"
name="runtime"
%s
}`
			initialTimeouts := ""
			if scenario == "configure-and-change" {
				initialTimeouts = `timeouts {
 create="2m"
read="3m"
delete="4m"
}`
			}
			initial := fmt.Sprintf(config, initialTimeouts)
			updated := fmt.Sprintf(config, `timeouts {
 create="7m"
read="7m"
delete="7m"
}`)

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
					{Config: initial, Check: check},
					{Config: updated, Check: resource.ComposeTestCheckFunc(check, resource.TestCheckResourceAttr("rediscloud_agent_memory_api_key.runtime", "timeouts.read", "7m"))},
					{Config: updated, PlanOnly: true},
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
