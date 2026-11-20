package unit_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	provider "github.com/RedisLabs/terraform-provider-rediscloud/provider"
)

const storeID = "st_11111111111111111111111111111111"
const extractorID = "ex_22222222222222222222222222222222"
const keyID = "33333333333333333333333333333333"
const grantID = "44444444444444444444444444444444"

type fixture struct {
	mu                        sync.Mutex
	objects                   map[string]map[string]any
	revisions                 int
	creates, deletes, updates int
	t                         *testing.T
}

func newFixture(t *testing.T) (*fixture, *httptest.Server) {
	f := &fixture{objects: map[string]map[string]any{}, t: t}
	server := httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(server.Close)
	return f, server
}
func (f *fixture) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p := r.URL.Path
	if strings.HasPrefix(p, "/memory-proxy/") {
		assert.Equal(f.t, "Bearer memory-token", r.Header.Get("Authorization"))
		p = strings.TrimPrefix(p, "/memory-proxy")
	} else if strings.HasPrefix(p, "/identity-proxy/") {
		assert.Equal(f.t, "Bearer identity-token", r.Header.Get("Authorization"))
		p = strings.TrimPrefix(p, "/identity-proxy")
	} else {
		http.Error(w, "unknown service", 404)
		return
	}
	if r.Method == http.MethodGet {
		current, ok := f.objects[p]
		if !ok {
			w.WriteHeader(404)
			return
		}
		out := clone(current)
		if out["destinationNamespaceId"] == "" {
			delete(out, "destinationNamespaceId")
		}
		delete(out, "token")
		if p == "/iris/v1/agent-memory/stores/"+storeID {
			out["state"] = "READY"
			out["database"] = map[string]any{"databaseId": "db-1", "name": "API-added database name"}
		}
		write(w, out)
		return
	}
	if r.Method == http.MethodDelete {
		current, ok := f.objects[p]
		if !ok {
			w.WriteHeader(404)
			return
		}
		if strings.Contains(p, "/extractors/") || strings.Contains(p, "/memory-types/") {
			assert.Equal(f.t, current["revision"], r.URL.Query().Get("expectedRevision"))
		}
		if p == "/iris/v1/agent-memory/stores/"+storeID {
			assert.Equal(f.t, "false", r.URL.Query().Get("flush"))
		}
		delete(f.objects, p)
		f.deletes++
		w.WriteHeader(204)
		return
	}
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request", 400)
		return
	}
	if r.Method == http.MethodPost {
		f.creates++
		out := clone(body)
		out["createdAt"] = "2026-11-19T12:00:00Z"
		out["updatedAt"] = "2026-11-19T12:00:00Z"
		f.revisions++
		out["revision"] = fmt.Sprint(f.revisions)
		switch {
		case p == "/iris/v1/agent-memory/stores":
			out["storeId"] = storeID
			out["state"] = "PROVISIONING"
			out["extractionStrategy"] = "INSTRUCT"
			p += "/" + storeID
		case strings.HasSuffix(p, "/extractors"):
			out["extractorId"] = extractorID
			p += "/" + extractorID
			assert.Equal(f.t, false, body["enabled"])
		case strings.HasSuffix(p, "/memory-types"):
			p += "/" + body["name"].(string)
		case p == "/v1/api-keys":
			out["keyId"] = keyID
			out["token"] = "create-only-secret"
			out["tenantId"] = "tenant-1"
			out["createdAt"] = float64(123)
			p += "/" + keyID
			assert.Equal(f.t, float64(0), body["expiresAt"])
			grants := body["grants"].([]any)
			assert.Equal(f.t, map[string]any{"schema": "test", "nested": map[string]any{"enabled": false, "count": float64(0)}}, grants[0].(map[string]any)["constraints"])
		case strings.HasSuffix(p, "/identity-providers"):
			out["tenant"] = "tenant-1"
			p += "/" + body["name"].(string)
		case p == "/v1/subject-grants":
			out["id"] = grantID
			out["products"] = map[string]any{"memory": map[string]any{"grants": map[string]any{"store": map[string]any{storeID: map[string]any{"actions": []any{"read"}, "constraints": map[string]any{"schema": "nested"}}}}}}
			p += "/" + grantID
		default:
			http.Error(w, "unhandled create", 400)
			return
		}
		if f.objects[p] != nil {
			http.Error(w, "already exists", http.StatusConflict)
			return
		}
		f.objects[p] = out
		w.WriteHeader(201)
		write(w, out)
		return
	}
	current, ok := f.objects[p]
	if !ok {
		w.WriteHeader(404)
		return
	}
	if strings.Contains(p, "/extractors/") || strings.Contains(p, "/memory-types/") || strings.Contains(p, "/identity-providers/") {
		assert.Equal(f.t, current["revision"], body["expectedRevision"])
	}
	f.updates++
	delete(body, "expectedRevision")
	for k, v := range body {
		current[k] = v
	}
	f.revisions++
	current["revision"] = fmt.Sprint(f.revisions)
	write(w, current)
}
func clone(value map[string]any) map[string]any {
	b, _ := json.Marshal(value)
	var out map[string]any
	_ = json.Unmarshal(b, &out)
	return out
}
func write(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
func factories() map[string]func() (tfprotov5.ProviderServer, error) {
	return map[string]func() (tfprotov5.ProviderServer, error){"rediscloud": func() (tfprotov5.ProviderServer, error) {
		f, err := provider.MuxProviderServerCreator(provider.NewSdkProvider("test")(), provider.NewFrameworkProvider("test")())
		if err != nil {
			return nil, err
		}
		return f(), nil
	}}
}
func TestIrisResourcesLifecycle(t *testing.T) {
	t.Setenv("REDISCLOUD_ACCESS_KEY", "")
	t.Setenv("REDISCLOUD_SECRET_KEY", "")
	f, server := newFixture(t)
	const key = "rediscloud_iris_identity_api_key.key"
	const store = "rediscloud_iris_agent_memory.store"
	steps := []resource.TestStep{
		{ConfigFile: config.StaticFile("testdata/lifecycle.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy"), "identity_url": config.StringVariable(server.URL + "/identity-proxy"), "name": config.StringVariable("initial")}, Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr(store, "id", storeID), resource.TestCheckResourceAttr(store, "state", "READY"), resource.TestCheckResourceAttr(key, "token", "create-only-secret"))},
		{ConfigFile: config.StaticFile("testdata/lifecycle.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy"), "identity_url": config.StringVariable(server.URL + "/identity-proxy"), "name": config.StringVariable("updated")}, Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr(store, "name", "updated"), resource.TestCheckResourceAttr(key, "token", "create-only-secret"), resource.TestCheckResourceAttr(key, "expires_at", "0"))},
	}
	for _, entry := range []struct {
		address, id string
		ignore      []string
	}{
		{store, storeID, []string{"database_json", "long_term_memory_json", "flush_on_delete"}},
		{"rediscloud_iris_agent_memory_type.profile", storeID + "/profile", nil},
		{"rediscloud_iris_agent_memory_extractor.extractor", storeID + "/" + extractorID, nil},
		{key, keyID, []string{"token"}},
		{"rediscloud_iris_identity_provider.oidc", "tenant-1/Corporate", nil},
		{"rediscloud_iris_identity_subject_grant_set.subject", grantID, nil},
	} {
		steps = append(steps, resource.TestStep{ConfigFile: config.StaticFile("testdata/lifecycle.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy"), "identity_url": config.StringVariable(server.URL + "/identity-proxy"), "name": config.StringVariable("updated")}, ResourceName: entry.address, ImportState: true, ImportStateId: entry.id, ImportStateVerify: true, ImportStateVerifyIgnore: entry.ignore})
	}
	for i := range steps {
		steps[i].ProtoV5ProviderFactories = factories()
	}
	resource.UnitTest(t, resource.TestCase{Steps: steps})
	f.mu.Lock()
	defer f.mu.Unlock()
	require.Empty(t, f.objects)
	require.Equal(t, 6, f.creates)
	require.Equal(t, 6, f.deletes)
	require.Equal(t, 5, f.updates)
}
