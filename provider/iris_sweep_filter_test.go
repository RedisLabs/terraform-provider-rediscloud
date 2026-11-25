package provider_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/require"
)

func TestIrisSweepFiltersOwnershipAndPrefix(t *testing.T) {
	deleted := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if irisSweepCloudFixture(t, w, r) {
			return
		}
		if r.Method == http.MethodDelete {
			require.Equal(t, "false", r.URL.Query().Get("flush"))
			deleted = append(deleted, r.URL.Path)
			w.WriteHeader(204)
			return
		}
		require.Equal(t, "/iris/v1/agent-memory/stores", r.URL.Path)
		recent := irisSweepStore(testResourcePrefix+"-iris-acc-recent", "recent", "111")
		recent["createdAt"] = time.Now().UTC().Format(time.RFC3339)
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"items": []any{
			irisSweepStore(testResourcePrefix+"-iris-acc-test", "owned", "111"),
			irisSweepStore(testResourcePrefix+"-iris-acc-second", "second-db", "222"),
			irisSweepStore(testResourcePrefix+"-iris-acc-user-db", "not-owned", "333"),
			irisSweepStore("existing-store", "existing", "111"), recent,
		}}))
	}))
	defer server.Close()
	irisSweepEnv(t, server.URL)
	require.NoError(t, testSweepIrisAgentMemory(t.Name()))
	// Both test-named databases are ours; the user's database, the unprefixed
	// store and the recent store are not.
	require.Equal(t, []string{"/iris/v1/agent-memory/stores/owned", "/iris/v1/agent-memory/stores/second-db"}, deleted)
}

func irisSweepStore(name, id, database string) map[string]any {
	return map[string]any{"name": name, "storeId": id, "createdAt": "2020-01-01T00:00:00Z", "database": map[string]string{"databaseId": database}}
}

// irisSweepCloudFixture serves one old test-named Essentials subscription whose
// databases 111 and 222 are test-named, next to a user subscription it must ignore.
func irisSweepCloudFixture(t *testing.T, w http.ResponseWriter, r *http.Request) bool {
	t.Helper()
	switch r.URL.Path {
	case "/fixed/subscriptions":
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"subscriptions": []any{
			map[string]any{"id": 123, "name": testResourcePrefix + "-iris-acc-old", "creationDate": "2020-01-01T00:00:00Z"},
			map[string]any{"id": 456, "name": "user-subscription", "creationDate": "2020-01-01T00:00:00Z"},
		}}))
	case "/fixed/subscriptions/123/databases":
		databases := []any{}
		if r.URL.Query().Get("offset") == "0" {
			databases = []any{
				map[string]any{"databaseId": 111, "name": testResourcePrefix + "-iris-acc-one"},
				map[string]any{"databaseId": 222, "name": testResourcePrefix + "-iris-acc-two"},
			}
		}
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"subscription": map[string]any{"databases": databases}}))
	default:
		return false
	}
	return true
}

func irisSweepEnv(t *testing.T, serverURL string) {
	t.Helper()
	t.Setenv("REDISCLOUD_URL", serverURL)
	t.Setenv("REDISCLOUD_ACCESS_KEY", "test-key")
	t.Setenv("REDISCLOUD_SECRET_KEY", "test-secret")
	t.Setenv("REDISCLOUD_IRIS_AGENT_MEMORY_URL", serverURL)
	t.Setenv("REDISCLOUD_IRIS_AGENT_MEMORY_TOKEN", "test-token")
	t.Setenv("REDISCLOUD_IRIS_IDENTITY_URL", serverURL)
	t.Setenv("REDISCLOUD_IRIS_IDENTITY_TOKEN", "test-token")
	t.Setenv("SWEEP_AGE_THRESHOLD", "2h")
}

// irisSweepServer serves the Cloud API, the store listing and an Identity
// handler on one host. Owned stores are owned-store (db 111) and other-owned
// (db 222); foreign-store sits in a database this run did not create.
func irisSweepServer(t *testing.T, identity http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if irisSweepCloudFixture(t, w, r) {
			return
		}
		if r.URL.Path == "/iris/v1/agent-memory/stores" {
			require.Equal(t, http.MethodGet, r.Method)
			prefix := testResourcePrefix + "-iris-acc-"
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"items": []any{
				irisSweepStore(prefix+"one", "owned-store", "111"),
				irisSweepStore(prefix+"two", "other-owned", "222"),
				irisSweepStore(prefix+"foreign", "foreign-store", "333"),
			}}))
			return
		}
		identity(w, r)
	}))
	t.Cleanup(server.Close)
	irisSweepEnv(t, server.URL)
	return server
}

func init() {
	if marker := os.Getenv("IRIS_SWEEP_REGRESSION_MARKER"); marker != "" {
		resource.AddTestSweepers("iris_regression_failure", &resource.Sweeper{Name: "iris_regression_failure", F: func(string) error { return errors.New("simulated Iris failure") }})
		resource.AddTestSweepers("iris_regression_infra", &resource.Sweeper{Name: "iris_regression_infra", Dependencies: []string{"iris_regression_failure"}, F: func(string) error { return os.WriteFile(marker, []byte("cleanup ran"), 0600) }})
	}
}

func TestSweeperFailureDoesNotSuppressInfrastructureCleanup(t *testing.T) {
	for _, allow := range []bool{false, true} {
		marker := filepath.Join(t.TempDir(), "cleanup")
		ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
		defer cancel()
		// Exercise TestMain in this test binary with fake sweepers, not a Terraform subprocess.
		//lintignore:XR008
		cmd := exec.CommandContext(ctx, os.Args[0], "-sweep=unit", "-sweep-run=iris_regression_infra")
		if allow {
			cmd.Args = append(cmd.Args, "-sweep-allow-failures")
		}
		cmd.Env = append(os.Environ(), "IRIS_SWEEP_REGRESSION_MARKER="+marker)
		output, err := cmd.CombinedOutput()
		require.Error(t, err, "the Iris failure must remain visible: %s", output)
		_, err = os.Stat(marker)
		if allow {
			require.NoError(t, err, "paid cleanup must run despite dependency failure: %s", output)
		} else {
			require.True(t, os.IsNotExist(err))
		}
	}
}

func TestIrisSweepRequiresOptInForZeroAge(t *testing.T) {
	t.Setenv("SWEEP_AGE_THRESHOLD", "0s")
	t.Setenv("IRIS_SWEEP_ALLOW_ACTIVE", "")
	t.Setenv("REDISCLOUD_IRIS_AGENT_MEMORY_URL", "https://must-not-be-called.example.com")
	t.Setenv("REDISCLOUD_IRIS_IDENTITY_URL", "https://must-not-be-called.example.com")
	require.ErrorContains(t, testSweepIrisAgentMemory(t.Name()), "IRIS_SWEEP_ALLOW_ACTIVE")
	require.ErrorContains(t, testSweepIrisIdentityAPIKeys(t.Name()), "IRIS_SWEEP_ALLOW_ACTIVE")
	t.Setenv("IRIS_SWEEP_ALLOW_ACTIVE", "1")
	age, err := irisSweepAge()
	require.NoError(t, err)
	require.Zero(t, age)
}

func irisSweepKey(name, id string, created int64, grants ...map[string]string) map[string]any {
	list := []any{}
	for _, grant := range grants {
		list = append(list, grant)
	}
	return map[string]any{"name": name, "keyId": id, "createdAt": created, "grants": list}
}

func irisSweepGrant(resourceID string) map[string]string {
	return map[string]string{"product": "memory", "resourceType": "mem-store", "resourceId": resourceID}
}

func TestIrisKeySweepDeletesOnlyKeysOfOwnedStores(t *testing.T) {
	var deleted []string
	var mu sync.Mutex
	old := time.Now().Add(-3 * time.Hour).Unix()
	recent := time.Now().Add(-time.Hour).Unix()
	prefix := testResourcePrefix + "-iris-acc-"
	otherProduct := map[string]string{"product": "other", "resourceType": "mem-store", "resourceId": "owned-store"}
	otherType := map[string]string{"product": "memory", "resourceType": "other", "resourceId": "owned-store"}
	irisSweepServer(t, func(w http.ResponseWriter, req *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if req.Method == http.MethodDelete {
			deleted = append(deleted, req.URL.Path)
			if len(deleted) == 2 {
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.WriteHeader(http.StatusNoContent)
			}
			return
		}
		require.Equal(t, "/v1/api-keys", req.URL.Path)
		require.Empty(t, deleted, "finish listing before deleting paginated records")
		require.Equal(t, "memory", req.URL.Query().Get("product"))
		require.Equal(t, "mem-store", req.URL.Query().Get("resourceType"))
		require.Equal(t, "100", req.URL.Query().Get("maxResults"))
		if req.URL.Query().Get("nextToken") == "" {
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"apiKeys": []any{
				irisSweepKey(prefix+"owned", "owned", old, irisSweepGrant("owned-store")),
				irisSweepKey("existing-key", "existing", old, irisSweepGrant("owned-store")),
				irisSweepKey(prefix+"recent", "recent", recent, irisSweepGrant("owned-store")),
				irisSweepKey(prefix+"missing-date", "missing-date", 0, irisSweepGrant("owned-store")),
				irisSweepKey(prefix+"foreign", "foreign", old, irisSweepGrant("foreign-store")),
				irisSweepKey(prefix+"mixed", "mixed", old, irisSweepGrant("owned-store"), irisSweepGrant("foreign-store")),
				irisSweepKey(prefix+"no-grants", "no-grants", old),
			}, "nextToken": "page-2"}))
		} else {
			require.Equal(t, "page-2", req.URL.Query().Get("nextToken"))
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"apiKeys": []any{
				irisSweepKey(prefix+"other-product", "other-product", old, otherProduct),
				irisSweepKey(prefix+"other-type", "other-type", old, otherType),
				irisSweepKey(prefix+"two-owned", "two-owned", old, irisSweepGrant("owned-store"), irisSweepGrant("other-owned")),
			}}))
		}
	})
	require.NoError(t, testSweepIrisIdentityAPIKeys(t.Name()))
	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, []string{"/v1/api-keys/owned", "/v1/api-keys/two-owned"}, deleted)
}

func TestIrisKeySweepNeedsAgentMemoryToProveOwnership(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("the key sweeper must not call any service when ownership cannot be proven")
	}))
	defer server.Close()
	irisSweepEnv(t, server.URL)
	t.Setenv("REDISCLOUD_IRIS_AGENT_MEMORY_URL", "")
	require.NoError(t, testSweepIrisIdentityAPIKeys(t.Name()))
}

func TestIrisKeySweepRefusesMalformedListings(t *testing.T) {
	for _, failure := range []string{"missing ID", "repeated token"} {
		t.Run(failure, func(t *testing.T) {
			irisSweepServer(t, func(w http.ResponseWriter, req *http.Request) {
				require.Equal(t, http.MethodGet, req.Method, "unsafe listings must prevent deletion")
				if failure == "missing ID" {
					require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"apiKeys": []any{irisSweepKey(testResourcePrefix+"-iris-acc-owned", "", time.Now().Add(-3*time.Hour).Unix(), irisSweepGrant("owned-store"))}}))
				} else {
					require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"apiKeys": []any{}, "nextToken": "repeated"}))
				}
			})
			require.Error(t, testSweepIrisIdentityAPIKeys(t.Name()))
		})
	}
}
