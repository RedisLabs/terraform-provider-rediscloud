package agentmemory_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"

	agentmemoryapi "github.com/RedisLabs/rediscloud-go-api/service/agentmemory"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

// This opt-in acceptance test owns only its uniquely named store and runtime key.
// The existing database is supplied by the operator and is never destroyed here.
// Import verification uses the harness's temporary import state; the original
// test state remains the sole owner responsible for cleanup.
func TestAccAgentMemory_ProductionReadiness_CRUDI(t *testing.T) {
	if os.Getenv("TF_ACC") == "" || os.Getenv("AGENT_MEMORY_TEST_DATABASE_ID") == "" {
		t.Skip("set TF_ACC and explicit Agent Memory QA fixture inputs to run")
	}
	databaseID, err := strconv.Atoi(os.Getenv("AGENT_MEMORY_TEST_DATABASE_ID"))
	require.NoError(t, err, "AGENT_MEMORY_TEST_DATABASE_ID must be a positive existing QA database ID")
	require.Positive(t, databaseID)
	prefix := os.Getenv("AGENT_MEMORY_TEST_PREFIX")
	require.Regexp(t, regexp.MustCompile(`^tfacc-[a-z0-9-]{8,40}$`), prefix)
	configuredClient, err := client.NewClient()
	require.NoError(t, err)
	api := configuredClient.Client
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	existing, listErr := api.AgentMemory.List(ctx)
	if listErr != nil {
		t.Fatal(client.SafeAPIError(listErr))
	}
	for _, store := range existing {
		require.NotEqual(t, prefix, store.Name, "a fixture with this name already exists; reconcile it before retrying")
	}
	const storeAddress = "rediscloud_agent_memory.readiness"
	const keyAddress = "rediscloud_agent_memory_api_key.readiness"
	var storeID, firstKeyID string
	config := func(ttl int, keyName string) string {
		return fmt.Sprintf(`
resource "rediscloud_agent_memory" "readiness" {
  name                       = %q
  database_id                = %d
  short_term_ttl_seconds     = %d
  long_term_ttl_seconds      = 7776000
  extraction_cadence_seconds = 300
}
resource "rediscloud_agent_memory_api_key" "readiness" {
  store_id = rediscloud_agent_memory.readiness.id
  name     = %q
  lifecycle {
    create_before_destroy = true
  }
}
`, prefix, databaseID, ttl, keyName)
	}
	verify := func(ttl int, rotated bool) resource.TestCheckFunc {
		return func(state *terraform.State) error {
			store, storeOK := state.RootModule().Resources[storeAddress]
			key, keyOK := state.RootModule().Resources[keyAddress]
			if !storeOK || !keyOK || store.Primary.ID == "" || key.Primary.ID == "" {
				return errors.New("expected managed store and runtime key with IDs")
			}
			if storeID == "" {
				storeID, firstKeyID = store.Primary.ID, key.Primary.ID
			}
			if store.Primary.ID != storeID {
				return errors.New("store ID changed unexpectedly")
			}
			if key.Primary.Attributes["api_key"] == "" {
				return errors.New("runtime key secret was not retained in state")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			remote, readErr := api.AgentMemory.Get(ctx, storeID)
			if readErr != nil {
				return errors.New(client.SafeAPIError(readErr))
			}
			if remote.ShortMemory == nil || remote.ShortMemory.TTLSeconds != ttl || remote.Status != agentmemoryapi.StatusReady {
				return errors.New("remote TTL or READY status did not converge")
			}
			if rotated {
				if key.Primary.ID == firstKeyID {
					return errors.New("runtime key replacement retained the old ID")
				}
				_, readErr = api.AgentMemory.GetAPIKey(ctx, storeID, firstKeyID)
				var missing *agentmemoryapi.APIKeyNotFound
				if !errors.As(readErr, &missing) {
					return errors.New("old runtime key is still listed or its removal could not be verified")
				}
			} else if key.Primary.ID != firstKeyID {
				return errors.New("runtime key changed during store maintenance")
			}
			return nil
		}
	}
	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy: func(_ *terraform.State) error {
			if storeID == "" {
				return nil
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			_, readErr := api.AgentMemory.Get(ctx, storeID)
			var missing *agentmemoryapi.NotFound
			if !errors.As(readErr, &missing) {
				return errors.New("owned readiness store remains or cleanup could not be verified")
			}
			return nil
		},
		Steps: []resource.TestStep{
			{Config: config(86400, prefix+"-key"), Check: verify(86400, false)},
			{
				ResourceName: storeAddress, ImportState: true, ImportStateVerify: true,
				// Import adopts advanced API defaults omitted from the create config.
				// Verify that block against the API, and compare every other field
				// against the original state using ImportStateVerify.
				ImportStateVerifyIgnore: []string{"summarization"},
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return errors.New("expected one imported store")
					}
					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
					defer cancel()
					remote, readErr := api.AgentMemory.Get(ctx, storeID)
					if readErr != nil {
						return errors.New(client.SafeAPIError(readErr))
					}
					attrs := states[0].Attributes
					if remote.Summarization == nil {
						if attrs["summarization.enabled"] != "" {
							return errors.New("import invented summarization absent from the API")
						}
						return nil
					}
					expected := map[string]string{
						"summarization.enabled":          strconv.FormatBool(remote.Summarization.Enabled),
						"summarization.trigger_strategy": remote.Summarization.TriggerStrategy,
					}
					if remote.Summarization.EventCount != nil {
						expected["summarization.event_count.threshold"] = strconv.Itoa(remote.Summarization.EventCount.Threshold)
						expected["summarization.event_count.retain_count"] = strconv.Itoa(remote.Summarization.EventCount.RetainCount)
					}
					for field, want := range expected {
						if attrs[field] != want {
							return fmt.Errorf("imported %s differs from the API", field)
						}
					}
					return nil
				},
			},
			{
				ResourceName: keyAddress, ImportState: true, ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"api_key"},
				ImportStateIdFunc: func(_ *terraform.State) (string, error) {
					return storeID + "/" + firstKeyID, nil
				},
			},
			{Config: config(43200, prefix+"-key"), Check: verify(43200, false)},
			{
				PreConfig: func() {
					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
					defer cancel()
					updateErr := api.AgentMemory.Update(ctx, storeID, agentmemoryapi.UpdateStore{
						ShortMemory: &agentmemoryapi.ShortMemoryConfig{TTLSeconds: 21600},
					})
					if updateErr != nil {
						t.Fatal(client.SafeAPIError(updateErr))
					}
					remote, readErr := api.AgentMemory.Get(ctx, storeID)
					if readErr != nil {
						t.Fatal(client.SafeAPIError(readErr))
					}
					require.NotNil(t, remote.ShortMemory)
					require.Equal(t, 21600, remote.ShortMemory.TTLSeconds, "out-of-band drift was not established")
				},
				Config: config(43200, prefix+"-key"), Check: verify(43200, false),
			},
			{Config: config(43200, prefix+"-rotated"), Check: verify(43200, true)},
			{Config: config(43200, prefix+"-rotated"), PlanOnly: true, Check: verify(43200, true)},
		},
	})
}
