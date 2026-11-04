package unit_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	api "github.com/RedisLabs/rediscloud-go-api/service/agentmemory"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/stretchr/testify/assert"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/agentmemory"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAgentMemoryAPIKeyIdentityPlans(t *testing.T) {
	for _, changed := range []string{"name", "store"} {
		t.Run(changed, func(t *testing.T) {
			fake := testhelpers.NewFakeAPI()
			var key *api.APIKeyInfo
			var currentStore string
			creates, deletes := 0, 0
			fake.Handle("GET /memory-stores/{store}/api-keys", func(req *http.Request) (any, error) {
				keys := []api.APIKeyInfo{}
				if key != nil && req.PathValue("store") == currentStore {
					keys = append(keys, *key)
				}
				return api.ListAPIKeysResponse{APIKeys: keys}, nil
			})
			fake.Handle("POST /memory-stores/{store}/api-keys", func(req *http.Request) (any, error) {
				var payload api.CreateAPIKeyRequest
				if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
					return nil, err
				}
				if key != nil {
					return nil, fmt.Errorf("old key must be deleted before replacement")
				}
				creates++
				currentStore = req.PathValue("store")
				key = &api.APIKeyInfo{APIKeyID: redis.String(fmt.Sprintf("key-%d", creates)), KeyName: payload.KeyName}
				return api.CreateAPIKeyResponse{APIKey: redis.String(fmt.Sprintf("FAKE-SECRET-%d", creates))}, nil
			})
			fake.Handle("DELETE /memory-stores/{store}/api-keys/{key}", func(req *http.Request) (any, error) {
				if key == nil || req.PathValue("store") != currentStore || req.PathValue("key") != redis.StringValue(key.APIKeyID) {
					return nil, fmt.Errorf("delete must target the tracked key")
				}
				deletes++
				key = nil
				return map[string]any{}, nil
			})
			nextStore, nextName := "store-1", "runtime"
			if changed == "store" {
				nextStore = "store-2"
			} else {
				nextName = "renamed"
			}
			const address = "test_agent_memory_api_key.runtime"
			resource.UnitTest(t, resource.TestCase{
				ProtoV5ProviderFactories: testhelpers.FrameworkProviderFactories(testhelpers.NewAPIClient(t, fake), agentmemory.NewAgentMemoryAPIKeyResource),
				Steps: []resource.TestStep{
					{
						ConfigFile: config.StaticFile("testdata/api_key_identity.tf"),
						ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
							plancheck.ExpectUnknownValue(address, tfjsonpath.New("id")),
							plancheck.ExpectUnknownValue(address, tfjsonpath.New("api_key")),
						}},
					},
					{
						ConfigFile:      config.StaticFile("testdata/api_key_identity.tf"),
						ConfigVariables: config.Variables{"read_timeout": config.StringVariable("3s")},
						ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate),
							plancheck.ExpectKnownValue(address, tfjsonpath.New("id"), knownvalue.StringExact("key-1")),
							plancheck.ExpectResourceAction("terraform_data.key_reference", plancheck.ResourceActionNoop),
						}},
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr(address, "id", "key-1"),
							resource.TestCheckResourceAttr(address, "api_key", "FAKE-SECRET-1"),
						),
					},
					{
						ConfigFile: config.StaticFile("testdata/api_key_identity.tf"),
						ConfigVariables: config.Variables{
							"name": config.StringVariable(nextName), "store_id": config.StringVariable(nextStore),
							"read_timeout": config.StringVariable("3s"),
						},
						ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(address, plancheck.ResourceActionDestroyBeforeCreate),
							plancheck.ExpectUnknownValue(address, tfjsonpath.New("id")),
							plancheck.ExpectUnknownValue(address, tfjsonpath.New("api_key")),
						}},
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr(address, "id", "key-2"),
							resource.TestCheckResourceAttr(address, "api_key", "FAKE-SECRET-2"),
							resource.TestCheckResourceAttr("terraform_data.key_reference", "input", "key-2"),
						),
					},
				},
			})
			fake.WithHandlersPaused(func() {
				assert.Equal(t, 2, creates)
				assert.Equal(t, 2, deletes)
				assert.Nil(t, key)
			})
		})
	}
}
