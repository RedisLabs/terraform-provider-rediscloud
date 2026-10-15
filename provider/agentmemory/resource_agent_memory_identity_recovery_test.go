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

func TestAccAgentMemory_ReadIdentityRecovery(t *testing.T) {
	if os.Getenv("TF_ACC") == "" || os.Getenv("AGENT_MEMORY_TEST_DATABASE_ID") == "" {
		t.Skip("set TF_ACC and explicit Agent Memory QA fixture inputs to run")
	}
	databaseID, err := strconv.Atoi(os.Getenv("AGENT_MEMORY_TEST_DATABASE_ID"))
	require.NoError(t, err)
	require.Positive(t, databaseID)
	prefix := os.Getenv("AGENT_MEMORY_TEST_PREFIX")
	require.Regexp(t, regexp.MustCompile(`^tfacc-[a-z0-9-]{8,40}$`), prefix)
	prefix += "-read-id"
	admin, err := client.NewClient()
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	stores, err := admin.Client.AgentMemory.List(ctx)
	if err != nil {
		t.Fatal(client.SafeAPIError(err))
	}
	for _, store := range stores {
		require.NotEqual(t, prefix, store.Name, "fixture exists; reconcile before retry")
	}
	baseURL := os.Getenv("REDISCLOUD_URL")
	if baseURL == "" {
		baseURL = "https://api.redislabs.com/v1"
	}
	target, err := url.Parse(baseURL)
	require.NoError(t, err)
	storePath := strings.TrimSuffix(target.Path, "/") + "/memory-stores"
	var injected atomic.Bool
	var firstID atomic.Value
	var creates, deletes atomic.Int64
	proxy := &httputil.ReverseProxy{Rewrite: func(req *httputil.ProxyRequest) {
		req.SetURL(target)
		req.Out.Header.Del("Accept-Encoding")
	}}
	proxy.ModifyResponse = func(resp *http.Response) error {
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil
		}
		requestPath := strings.TrimSuffix(resp.Request.URL.Path, "/")
		if resp.Request.Method == http.MethodPost && requestPath == storePath {
			creates.Add(1)
		}
		storeGet := strings.HasPrefix(requestPath, storePath+"/") && !strings.Contains(strings.TrimPrefix(requestPath, storePath+"/"), "/")
		if resp.Request.Method == http.MethodDelete && storeGet {
			deletes.Add(1)
		}
		if resp.Request.Method != http.MethodGet || !storeGet || injected.Load() {
			return nil
		}
		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return errors.New("could not inspect owned fixture read response")
		}
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			return errors.New("owned fixture read response was not JSON")
		}
		id, ok := payload["storeId"].(string)
		if !ok || id == "" {
			return errors.New("owned fixture read already lacked its identity")
		}
		if injected.CompareAndSwap(false, true) {
			firstID.Store(id)
			delete(payload, "storeId")
			body, err = json.Marshal(payload)
			if err != nil {
				return errors.New("could not encode controlled fixture response")
			}
		}
		replaceRecoveryResponseBody(resp, string(body))
		return nil
	}
	server := httptest.NewServer(proxy)
	defer server.Close()
	t.Setenv("REDISCLOUD_URL", server.URL)
	config := fmt.Sprintf(`
resource "rediscloud_agent_memory" "identity" {
  name                       = %q
  database_id                = %d
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 7776000
  extraction_cadence_seconds = 300
  summarization {
    enabled = true
    trigger_strategy = "event_count"
    event_count {
      threshold = 20
      retain_count = 10
    }
  }
}
resource "rediscloud_agent_memory_api_key" "identity" {
  store_id = rediscloud_agent_memory.identity.id
  name = %q
}
`, prefix, databaseID, prefix+"-runtime")
	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{Config: config, ExpectError: regexp.MustCompile("Invalid Agent Memory response")},
			{Config: config, Check: func(state *terraform.State) error {
				store := state.RootModule().Resources["rediscloud_agent_memory.identity"]
				key := state.RootModule().Resources["rediscloud_agent_memory_api_key.identity"]
				oldID, _ := firstID.Load().(string)
				if !injected.Load() || oldID == "" || store == nil || key == nil || store.Primary.ID == oldID || key.Primary.Attributes["api_key"] == "" {
					return errors.New("read-identity fault did not recover tracked ownership and key")
				}
				if creates.Load() != 2 || deletes.Load() != 1 {
					return errors.New("tainted replacement did not delete before recreating")
				}
				ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
				defer cancel()
				_, err := admin.Client.AgentMemory.Get(ctx, oldID)
				var notFound *agentmemoryapi.NotFound
				if !errors.As(err, &notFound) {
					return errors.New("original tracked fixture still exists or absence could not be verified")
				}
				return nil
			}},
			{Config: config, PlanOnly: true},
		},
		CheckDestroy: func(_ *terraform.State) error {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			stores, err := admin.Client.AgentMemory.List(ctx)
			if err != nil {
				return errors.New(client.SafeAPIError(err))
			}
			for _, store := range stores {
				if store.Name == prefix {
					return errors.New("read-identity fixture remains; reconcile its run prefix")
				}
			}
			return nil
		},
	})
}
