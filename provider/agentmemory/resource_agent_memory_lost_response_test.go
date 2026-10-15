package agentmemory_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	agentmemoryapi "github.com/RedisLabs/rediscloud-go-api/service/agentmemory"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

// Exercise documented reconciliation, not an automatic idempotency guarantee.
// The loopback proxy drops one successful create response only after the real
// service accepted it. It never logs bodies, headers, credentials or tokens.
func TestAccAgentMemory_AcceptedLostResponseRecovery(t *testing.T) {
	if os.Getenv("TF_ACC") == "" || os.Getenv("AGENT_MEMORY_TEST_DATABASE_ID") == "" {
		t.Skip("set TF_ACC and explicit Agent Memory QA fixture inputs to run")
	}
	databaseID, err := strconv.Atoi(os.Getenv("AGENT_MEMORY_TEST_DATABASE_ID"))
	require.NoError(t, err)
	require.Positive(t, databaseID)
	basePrefix := os.Getenv("AGENT_MEMORY_TEST_PREFIX")
	require.Regexp(t, regexp.MustCompile(`^tfacc-[a-z0-9-]{8,40}$`), basePrefix)
	admin, err := client.NewClient()
	require.NoError(t, err)
	baseURL := os.Getenv("REDISCLOUD_URL")
	if baseURL == "" {
		baseURL = "https://api.redislabs.com/v1"
	}
	target, err := url.Parse(baseURL)
	require.NoError(t, err)

	for _, caseName := range []string{"store", "key", "task-result"} {
		loseKey := caseName == "key"
		loseTask := caseName == "task-result"
		t.Run(caseName, func(t *testing.T) {
			prefix := basePrefix + "-lost-" + caseName
			keyName := prefix + "-runtime"
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			existing, listErr := admin.Client.AgentMemory.List(ctx)
			if listErr != nil {
				t.Fatal(client.SafeAPIError(listErr))
			}
			for _, store := range existing {
				require.NotEqual(t, prefix, store.Name, "fixture already exists; reconcile before retrying")
			}
			var injected atomic.Bool
			var storeCreates, keyCreates atomic.Int64
			var createTaskID atomic.Value
			proxy := &httputil.ReverseProxy{
				Rewrite: func(req *httputil.ProxyRequest) {
					// SetURL also rewrites Host for the upstream API gateway.
					req.SetURL(target)
					// Let the proxy transport negotiate and decode compression;
					// forwarding the SDK's header leaves gzip bytes in ModifyResponse.
					req.Out.Header.Del("Accept-Encoding")
				},
			}
			proxy.ModifyResponse = func(resp *http.Response) error {
				if resp.StatusCode < 200 || resp.StatusCode >= 300 {
					return nil
				}
				storePost := resp.Request.Method == http.MethodPost && strings.TrimSuffix(resp.Request.URL.Path, "/") == strings.TrimSuffix(target.Path, "/")+"/memory-stores"
				keyPost := resp.Request.Method == http.MethodPost && strings.HasSuffix(resp.Request.URL.Path, "/api-keys")
				if storePost {
					storeCreates.Add(1)
				}
				if keyPost {
					keyCreates.Add(1)
				}
				if loseTask {
					taskID, _ := createTaskID.Load().(string)
					taskGet := taskID != "" && resp.Request.Method == http.MethodGet && resp.Request.URL.Path == strings.TrimSuffix(target.Path, "/")+"/tasks/"+url.PathEscape(taskID)
					if storePost || taskGet {
						body, err := io.ReadAll(resp.Body)
						_ = resp.Body.Close()
						if err != nil {
							return errors.New("could not inspect owned fixture task response")
						}
						var task struct {
							ID     string `json:"taskId"`
							Status string `json:"status"`
						}
						if err := json.Unmarshal(body, &task); err != nil {
							return errors.New("owned fixture task response was not JSON")
						}
						if storePost {
							createTaskID.Store(task.ID)
						}
						if taskGet && task.Status == "processing-completed" && injected.CompareAndSwap(false, true) {
							body = []byte(`{"status":"processing-completed","response":{}}`)
						}
						replaceRecoveryResponseBody(resp, string(body))
					}
					return nil
				}
				if ((loseKey && keyPost) || (!loseKey && storePost)) && injected.CompareAndSwap(false, true) {
					_, _ = io.Copy(io.Discard, resp.Body)
					_ = resp.Body.Close()
					body := `{"message":"controlled accepted-but-lost response"}`
					resp.StatusCode = http.StatusServiceUnavailable
					resp.Status = "503 Service Unavailable"
					resp.Body = io.NopCloser(strings.NewReader(body))
					resp.ContentLength = int64(len(body))
					resp.Header.Set("Content-Length", strconv.Itoa(len(body)))
					resp.Header.Set("Content-Type", "application/json")
					resp.Header.Del("Content-Encoding")
				}
				return nil
			}
			server := httptest.NewServer(proxy)
			defer server.Close()
			t.Setenv("REDISCLOUD_URL", server.URL)
			config := fmt.Sprintf(`
resource "rediscloud_agent_memory" "recovery" {
  name                       = %q
  database_id                = %d
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 7776000
  extraction_cadence_seconds = 300
  summarization {
    enabled          = true
    trigger_strategy = "event_count"
    event_count {
      threshold    = 20
      retain_count = 10
    }
  }
}
resource "rediscloud_agent_memory_api_key" "recovery" {
  store_id = rediscloud_agent_memory.recovery.id
  name     = %q
}
`, prefix, databaseID, keyName)
			var recoveredID, revokedKeyID string
			reconcile := func() {
				require.True(t, injected.Load(), "the response-loss injection did not occur")
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				defer cancel()
				for {
					stores, listErr := admin.Client.AgentMemory.List(ctx)
					if listErr != nil {
						t.Fatal(client.SafeAPIError(listErr))
					}
					matches := 0
					for _, store := range stores {
						if store.Name == prefix && store.DatabaseID == strconv.Itoa(databaseID) {
							matches++
							if store.Status == agentmemoryapi.StatusReady {
								recoveredID = store.StoreID
							}
						}
					}
					require.LessOrEqual(t, matches, 1, "ambiguous fixture ownership; do not blindly retry")
					if recoveredID != "" {
						break
					}
					select {
					case <-ctx.Done():
						t.Fatal("accepted store did not become discoverable/READY; reconcile fixture prefix")
					case <-time.After(time.Second):
					}
				}
				if loseKey {
					keys, listErr := admin.Client.AgentMemory.ListAPIKeys(ctx, recoveredID)
					if listErr != nil {
						t.Fatal(client.SafeAPIError(listErr))
					}
					require.Len(t, keys, 1, "only the failed request's key may be revoked")
					require.Equal(t, keyName, keys[0].KeyName)
					revokedKeyID = keys[0].APIKeyID
					// The lost secret cannot be imported. Revoke the positively
					// identified fixture key before retrying key creation.
					if deleteErr := admin.Client.AgentMemory.DeleteAPIKey(ctx, recoveredID, revokedKeyID); deleteErr != nil {
						t.Fatal(client.SafeAPIError(deleteErr))
					}
				}
			}
			expectedError := "HTTP 503"
			if loseTask {
				expectedError = "Failed to create Agent Memory service"
			}
			steps := []resource.TestStep{{Config: config, ExpectError: regexp.MustCompile(expectedError)}}
			if !loseKey {
				steps = append(steps, resource.TestStep{
					PreConfig: reconcile, ResourceName: "rediscloud_agent_memory.recovery",
					ImportState: true, ImportStatePersist: true,
					ImportStateIdFunc: func(_ *terraform.State) (string, error) { return recoveredID, nil },
				})
			}
			check := func(state *terraform.State) error {
				store := state.RootModule().Resources["rediscloud_agent_memory.recovery"]
				key := state.RootModule().Resources["rediscloud_agent_memory_api_key.recovery"]
				if store == nil || key == nil || store.Primary.ID != recoveredID || key.Primary.ID == "" || key.Primary.ID == revokedKeyID || key.Primary.Attributes["api_key"] == "" {
					return errors.New("reconciliation lost resource identity or did not replace the unrecoverable secret")
				}
				expectedKeys := int64(1)
				if loseKey {
					expectedKeys = 2
				}
				if storeCreates.Load() != 1 || keyCreates.Load() != expectedKeys {
					return errors.New("unexpected repeated store/key creation after reconciliation")
				}
				return nil
			}
			step := resource.TestStep{Config: config, Check: check}
			if loseKey {
				step.PreConfig = reconcile
			}
			steps = append(steps, step, resource.TestStep{Config: config, PlanOnly: true})
			resource.Test(t, resource.TestCase{
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				Steps:                    steps,
				CheckDestroy: func(_ *terraform.State) error {
					ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
					defer cancel()
					stores, listErr := admin.Client.AgentMemory.List(ctx)
					if listErr != nil {
						return errors.New(client.SafeAPIError(listErr))
					}
					for _, store := range stores {
						if store.Name == prefix {
							return errors.New("recovery fixture still exists; reconcile logged run prefix")
						}
					}
					return nil
				},
			})
		})
	}
}

func replaceRecoveryResponseBody(resp *http.Response, body string) {
	resp.Body = io.NopCloser(strings.NewReader(body))
	resp.ContentLength = int64(len(body))
	resp.Header.Set("Content-Length", strconv.Itoa(len(body)))
	resp.Header.Set("Content-Type", "application/json")
	resp.Header.Del("Content-Encoding")
}
