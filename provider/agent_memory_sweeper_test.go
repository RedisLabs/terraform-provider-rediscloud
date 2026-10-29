package provider_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	rediscloudapi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	api "github.com/RedisLabs/rediscloud-go-api/service/agentmemory"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sweepFixturePrefix = "tf-sweep-current"
const sweepFixtureSubscription = sweepFixturePrefix + "-fixture-am"

type agentMemorySweepFixture struct {
	t                *testing.T
	mu               sync.Mutex
	server           *httptest.Server
	stores           map[string]api.StoreSummary
	events           []string
	deployment       string
	subscriptionName string
	failList         bool
	failDelete       bool
	failGet          bool
	deletePending    bool
	pendingReads     int
	getCalls         int
}

func newAgentMemorySweepFixture(t *testing.T) *agentMemorySweepFixture {
	t.Helper()
	old := time.Now().Add(-48 * time.Hour)
	f := &agentMemorySweepFixture{
		t:                t,
		deployment:       subscriptions.SubscriptionDeploymentTypeSingleRegion,
		subscriptionName: sweepFixtureSubscription,
		stores: map[string]api.StoreSummary{
			"store-owned": {
				StoreID: redis.String("store-owned"), SubscriptionID: redis.String("101"),
				Name: redis.String(sweepFixtureSubscription + "-memory"), CreatedAt: &old,
			},
		},
	}
	f.server = httptest.NewServer(http.HandlerFunc(f.serveHTTP))
	t.Cleanup(f.server.Close)
	return f
}

func (f *agentMemorySweepFixture) serveHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	var body any
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/subscriptions":
		body = map[string]any{"subscriptions": []map[string]any{
			{"id": 101, "name": f.subscriptionName, "status": "active", "deploymentType": f.deployment},
			{"id": 202, "name": "customer-prod", "status": "active", "deploymentType": f.deployment},
		}}
	case r.Method == http.MethodGet && r.URL.Path == "/memory-stores":
		if f.failList {
			http.Error(w, "fixture list failure", http.StatusInternalServerError)
			return
		}
		stores := make([]api.StoreSummary, 0, len(f.stores))
		for _, store := range f.stores {
			stores = append(stores, store)
		}
		body = api.ListStoresResponse{Data: stores}
	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/memory-stores/"):
		if f.failDelete {
			http.Error(w, "fixture deletion failure", http.StatusInternalServerError)
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/memory-stores/")
		if _, ok := f.stores[id]; !ok {
			http.NotFound(w, r)
			return
		}
		if f.pendingReads > 0 {
			f.deletePending = true
		} else {
			delete(f.stores, id)
			f.events = append(f.events, "store")
		}
		body = map[string]any{"taskId": "delete-task"}
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/memory-stores/"):
		f.getCalls++
		if f.failGet {
			http.Error(w, "fixture deletion polling failure", http.StatusForbidden)
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/memory-stores/")
		if f.deletePending {
			if f.pendingReads == 0 {
				delete(f.stores, id)
				f.events = append(f.events, "store")
				f.deletePending = false
			} else {
				f.pendingReads--
			}
		}
		store, exists := f.stores[id]
		if !exists {
			http.NotFound(w, r)
			return
		}
		body = api.Store{StoreID: store.StoreID, Status: redis.String(api.StatusReady)}
	case r.Method == http.MethodGet && r.URL.Path == "/subscriptions/101/databases":
		dbs := []map[string]any{}
		if r.URL.Query().Get("offset") == "0" {
			dbs = append(dbs, map[string]any{"databaseId": 201, "status": "active", "activatedOn": time.Now().Add(-48 * time.Hour)})
		}
		body = map[string]any{"subscription": []map[string]any{{"subscriptionId": 101, "databases": dbs}}}
	case r.Method == http.MethodDelete && r.URL.Path == "/subscriptions/101/databases/201":
		if len(f.stores) != 0 {
			http.Error(w, "store must be removed before database", http.StatusConflict)
			return
		}
		f.events = append(f.events, "database")
		body = map[string]any{"taskId": "delete-task"}
	case r.Method == http.MethodDelete && r.URL.Path == "/subscriptions/101":
		f.events = append(f.events, "subscription")
		body = map[string]any{"taskId": "delete-task"}
	case r.Method == http.MethodGet && r.URL.Path == "/tasks/delete-task":
		body = map[string]any{"status": "processing-completed"}
	default:
		f.t.Errorf("unexpected sweeper request: %s %s", r.Method, r.URL.Path)
		http.Error(w, "unexpected request", http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		f.t.Errorf("encode sweeper fixture response: %v", err)
	}
}

func (f *agentMemorySweepFixture) run(t *testing.T, sweeper, age string) ([]byte, error) {
	t.Helper()
	binary, err := os.Executable()
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	// The subprocess exercises sweep mode in TestMain against the local fake API.
	//lintignore:XR008
	cmd := exec.CommandContext(ctx, binary, "-test.run=^$", "-sweep=ALL", "-sweep-run="+sweeper)
	// Exercise the real TestMain registry against only the local fake API.
	for _, value := range os.Environ() {
		if !strings.HasPrefix(value, "REDISCLOUD_") && !strings.HasPrefix(value, "TEST_RESOURCE_PREFIX=") && !strings.HasPrefix(value, "SWEEP_AGE_THRESHOLD=") {
			cmd.Env = append(cmd.Env, value)
		}
	}
	cmd.Env = append(cmd.Env,
		"REDISCLOUD_URL="+f.server.URL, "REDISCLOUD_ACCESS_KEY=fake-access", "REDISCLOUD_SECRET_KEY=fake-secret",
		"TEST_RESOURCE_PREFIX="+sweepFixturePrefix, "SWEEP_AGE_THRESHOLD="+age,
	)
	return cmd.CombinedOutput()
}

func TestAgentMemorySharedSweeperDependencyOrder(t *testing.T) {
	for _, tc := range []struct{ name, deployment string }{
		{"rediscloud_subscription", subscriptions.SubscriptionDeploymentTypeSingleRegion},
		{"rediscloud_active_active_subscription", subscriptions.SubscriptionDeploymentTypeActiveActive},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newAgentMemorySweepFixture(t)
			f.mu.Lock()
			f.deployment = tc.deployment
			f.mu.Unlock()
			out, err := f.run(t, tc.name, "0s")
			require.NoError(t, err, "%s", out)
			f.mu.Lock()
			defer f.mu.Unlock()
			assert.Equal(t, []string{"store", "database", "subscription"}, f.events)
		})
	}
}

func TestAgentMemorySharedSweeperFailureBlocksParents(t *testing.T) {
	for _, step := range []string{"list", "delete", "poll"} {
		t.Run(step, func(t *testing.T) {
			f := newAgentMemorySweepFixture(t)
			f.mu.Lock()
			f.failList, f.failDelete = step == "list", step == "delete"
			if step == "poll" {
				f.failGet, f.pendingReads = true, 1
			}
			f.mu.Unlock()
			out, err := f.run(t, "rediscloud_subscription", "0s")
			require.Error(t, err, "%s", out)
			switch step {
			case "list":
				assert.Contains(t, string(out), "fixture list failure")
			case "delete":
				assert.Contains(t, string(out), "fixture deletion failure")
			case "poll":
				assert.Contains(t, string(out), "fixture deletion polling failure")
			}
			f.mu.Lock()
			defer f.mu.Unlock()
			assert.Empty(t, f.events)
			assert.Contains(t, f.stores, "store-owned")
		})
	}
}

func TestAgentMemorySharedSweeperWaitsForDisappearance(t *testing.T) {
	f := newAgentMemorySweepFixture(t)
	f.mu.Lock()
	f.pendingReads = 1
	f.mu.Unlock()
	out, err := f.run(t, "rediscloud_subscription", "0s")
	require.NoError(t, err, "%s", out)
	f.mu.Lock()
	defer f.mu.Unlock()
	assert.Equal(t, 2, f.getCalls)
	assert.Equal(t, []string{"store", "database", "subscription"}, f.events)
	assert.Empty(t, f.stores)
}

func TestAgentMemorySweeperDeletionWaitHonorsContext(t *testing.T) {
	f := newAgentMemorySweepFixture(t)
	c, err := rediscloudapi.NewClient(rediscloudapi.BaseURL(f.server.URL), rediscloudapi.Auth("fake-access", "fake-secret"))
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	err = waitForSweptAgentMemoryDeletion(ctx, c, "store-owned")
	require.Error(t, err)
	require.Error(t, ctx.Err())
	f.mu.Lock()
	defer f.mu.Unlock()
	assert.Contains(t, f.stores, "store-owned")
	assert.Empty(t, f.events)
}

func TestAgentMemorySharedSweeperSkipsAPIWithoutFixtures(t *testing.T) {
	f := newAgentMemorySweepFixture(t)
	f.mu.Lock()
	f.subscriptionName = sweepFixturePrefix + "-database"
	f.stores = map[string]api.StoreSummary{}
	f.failList = true
	f.mu.Unlock()
	out, err := f.run(t, "rediscloud_subscription", "0s")
	require.NoError(t, err, "%s", out)
	f.mu.Lock()
	defer f.mu.Unlock()
	assert.Equal(t, []string{"database", "subscription"}, f.events)
}

func TestAgentMemorySharedSweeperPreservesOtherFixtures(t *testing.T) {
	f := newAgentMemorySweepFixture(t)
	old, recent := time.Now().Add(-48*time.Hour), time.Now().Add(-time.Hour)
	f.mu.Lock()
	f.stores["store-recent"] = api.StoreSummary{StoreID: redis.String("store-recent"), SubscriptionID: redis.String("101"), Name: redis.String(sweepFixtureSubscription + "-recent"), CreatedAt: &recent}
	f.stores["store-foreign"] = api.StoreSummary{StoreID: redis.String("store-foreign"), SubscriptionID: redis.String("202"), Name: redis.String(sweepFixtureSubscription + "-foreign"), CreatedAt: &old}
	f.stores["store-other-name"] = api.StoreSummary{StoreID: redis.String("store-other-name"), SubscriptionID: redis.String("101"), Name: redis.String("tf-sweep-other-fixture-am-memory"), CreatedAt: &old}
	f.mu.Unlock()
	out, err := f.run(t, "rediscloud_agent_memory", "24h")
	require.NoError(t, err, "%s", out)
	f.mu.Lock()
	defer f.mu.Unlock()
	assert.Equal(t, []string{"store"}, f.events)
	assert.NotContains(t, f.stores, "store-owned")
	assert.Contains(t, f.stores, "store-recent")
	assert.Contains(t, f.stores, "store-foreign")
	assert.Contains(t, f.stores, "store-other-name")
}

func TestSweepAgentMemoryFixtureIsolation(t *testing.T) {
	for _, tc := range []struct {
		name, prefix string
		want         bool
	}{
		{"tf-test-abcdef-am", "tf-test-abcdef", true},
		{"tf-test-ghijkl-am", "tf-test-abcdef", false},
		{"tf-test-abcdef-am", "tf-test", true},
		{"tf-testing-abcdef-am", "tf-test", false},
		{"customer-prod", "tf-test", false},
		{"tf-test-abcdef", "tf-test", false},
	} {
		t.Run(tc.name+"/"+tc.prefix, func(t *testing.T) {
			if got := agentMemorySweepOwnsSubscription(tc.name, tc.prefix); got != tc.want {
				t.Fatalf("subscription ownership=%t, want %t", got, tc.want)
			}
		})
	}
}

func TestSweepAgentMemoryOwnership(t *testing.T) {
	cutoff := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	old, recent := cutoff.Add(-time.Hour), cutoff.Add(time.Hour)
	for _, tc := range []struct {
		name, subscription, storeName, id string
		created                           *time.Time
		want                              bool
	}{
		{"owned and old", "123", "tf-test-abcdef-am-memory", "store-1", &old, true},
		{"boundary age", "123", "tf-test-abcdef-am-memory", "store-1", &cutoff, true},
		{"other subscription", "456", "tf-test-abcdef-am-memory", "store-1", &old, false},
		{"other fixture", "123", "tf-test-other-am-memory", "store-1", &old, false},
		{"prefix collision", "123", "tf-test-abcdef-amother", "store-1", &old, false},
		{"missing subscription", "", "tf-test-abcdef-am-memory", "store-1", &old, false},
		{"missing ID", "123", "tf-test-abcdef-am-memory", "", &old, false},
		{"missing timestamp", "123", "tf-test-abcdef-am-memory", "store-1", nil, false},
		{"recent store", "123", "tf-test-abcdef-am-memory", "store-1", &recent, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := api.StoreSummary{StoreID: redis.String(tc.id), SubscriptionID: redis.String(tc.subscription), Name: redis.String(tc.storeName), CreatedAt: tc.created}
			if got := agentMemorySweepOwnsStore(store, map[string]string{"123": "tf-test-abcdef-am"}, cutoff); got != tc.want {
				t.Fatalf("sweep ownership=%t, want %t", got, tc.want)
			}
		})
	}
}
