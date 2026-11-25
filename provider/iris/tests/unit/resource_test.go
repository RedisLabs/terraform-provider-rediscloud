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
	"time"

	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	provider "github.com/RedisLabs/terraform-provider-rediscloud/provider"
)

const storeID = "st_11111111111111111111111111111111"
const keyID = "33333333333333333333333333333333"

type fixture struct {
	mu                        sync.Mutex
	objects                   map[string]map[string]any
	retiredTypes              map[string]bool
	revisions                 int
	creates, deletes, updates int
	keysCreated               int
	typeReadFailures          int
	t                         *testing.T
}

func newFixture(t *testing.T) (*fixture, *httptest.Server) {
	f := &fixture{objects: map[string]map[string]any{}, retiredTypes: map[string]bool{}, t: t}
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
	isType := strings.Contains(p, "/memory-types/")
	isKey := strings.HasPrefix(p, "/v1/api-keys/")
	if isKey {
		if expires, ok := f.objects[p]["expiresAt"].(float64); ok && expires > 0 && expires <= float64(time.Now().Unix()) {
			delete(f.objects, p) // Redis expires the record before any read or write.
		}
	}
	if r.Method == http.MethodGet {
		if isType && f.typeReadFailures > 0 {
			f.typeReadFailures--
			http.Error(w, "temporary read failure", http.StatusBadGateway)
			return
		}
		current, ok := f.objects[p]
		if !ok {
			w.WriteHeader(404)
			return
		}
		out := clone(current)
		delete(out, "token")
		if grants, ok := out["grants"].([]any); ok {
			for _, rawGrant := range grants {
				grant := rawGrant.(map[string]any)
				actions := grant["actions"].([]any)
				for i, j := 0, len(actions)-1; i < j; i, j = i+1, j-1 {
					actions[i], actions[j] = actions[j], actions[i]
				}
			}
			for i, j := 0, len(grants)-1; i < j; i, j = i+1, j-1 {
				grants[i], grants[j] = grants[j], grants[i]
			}
		}
		if p == "/iris/v1/agent-memory/stores/"+storeID {
			out["state"] = "READY"
			delete(out, "customMemoryTypes")
			out["database"] = map[string]any{"databaseId": "db-1", "name": "API-added database name"}
			if long, ok := out["longTermMemory"].(map[string]any); ok {
				for _, purpose := range []string{"llm", "embedding", "classification"} {
					if model, ok := long[purpose].(map[string]any); ok {
						delete(model, "credentials")
					}
				}
			}
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
		if strings.Contains(p, "/memory-types/") {
			if current["revision"] != r.URL.Query().Get("expectedRevision") {
				http.Error(w, "revision conflict", http.StatusPreconditionFailed)
				return
			}
			if strategy, ok := current["extractionStrategy"].(map[string]any); ok && strategy["enabled"] != false {
				http.Error(w, "memory type in use", http.StatusConflict)
				return
			}
			f.retiredTypes[memoryTypeRetirementKey(p)] = true
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
	if strings.HasPrefix(p, "/v1/api-keys") {
		if expires, ok := body["expiresAt"].(float64); ok && expires != 0 && expires <= float64(time.Now().Unix()) {
			http.Error(w, "expiresAt must be in the future or zero", http.StatusBadRequest)
			return
		}
	}
	if r.Method == http.MethodPost {
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
		case strings.HasSuffix(p, "/memory-types"):
			if f.objects[strings.TrimSuffix(p, "/memory-types")] == nil {
				http.NotFound(w, r)
				return
			}
			if strategy, ok := out["extractionStrategy"].(map[string]any); ok {
				if strategy["prompt"] == nil || strategy["prompt"] == "" {
					http.Error(w, "prompt required on creation", http.StatusBadRequest)
					return
				}
				if strategy["enabled"] == nil {
					strategy["enabled"] = true
				}
			}
			p += "/" + body["name"].(string)
			if f.retiredTypes[memoryTypeRetirementKey(p)] {
				http.Error(w, "memory type name retired", http.StatusConflict)
				return
			}
		case p == "/v1/api-keys":
			out["tenantId"] = "tenant-1"
			if grants := out["grants"].([]any); len(grants) > 0 {
				if tenant, ok := grants[0].(map[string]any)["tenant"].(string); ok && strings.TrimSpace(tenant) != "" {
					out["tenantId"] = strings.TrimSpace(tenant)
				}
				// The provider owes Iris one tenant per key, stamped on every grant.
				for _, grant := range grants {
					assert.Equal(f.t, grants[0].(map[string]any)["tenant"], grant.(map[string]any)["tenant"], "grants of one key must carry the same tenant")
				}
			}
			if !stampGrantTenants(out) {
				http.Error(w, "agent key grants have conflicting tenants", http.StatusBadRequest)
				return
			}
			f.keysCreated++
			id, token := keyID, "create-only-secret"
			if f.keysCreated > 1 {
				id = fmt.Sprintf("%032d", f.keysCreated)
				token += fmt.Sprintf("-%d", f.keysCreated)
			}
			out["keyId"], out["token"], out["createdAt"] = id, token, float64(122+f.keysCreated)
			p += "/" + id
			grants := out["grants"].([]any)
			assert.NotEmpty(f.t, grants)
			for _, rawGrant := range grants {
				grant := rawGrant.(map[string]any)
				assert.NotContains(f.t, grant, "constraints")
				assert.Equal(f.t, "mem-store", grant["resourceType"])
			}
		default:
			http.Error(w, "unhandled create", 400)
			return
		}
		if f.objects[p] != nil {
			http.Error(w, "already exists", http.StatusConflict)
			return
		}
		if strings.Contains(p, "/memory-types/") {
			out["revision"] = "1"
		}
		f.creates++
		f.objects[p] = out
		w.WriteHeader(201)
		if strings.HasPrefix(p, "/v1/api-keys/") {
			write(w, map[string]any{"keyId": out["keyId"], "token": out["token"], "createdAt": out["createdAt"]})
		} else {
			write(w, out)
		}
		return
	}
	current, ok := f.objects[p]
	if !ok {
		w.WriteHeader(404)
		return
	}
	if strings.Contains(p, "/memory-types/") {
		if current["revision"] != body["expectedRevision"] {
			http.Error(w, "revision conflict", http.StatusPreconditionFailed)
			return
		}
		if body["extractionStrategy"] != nil && current["extractionStrategy"] == nil {
			http.Error(w, "memory type has no extraction strategy to update", http.StatusBadRequest)
			return
		}
	}
	if isKey && body["name"] == "renamed" {
		assert.NotContains(f.t, body, "grants")
	}
	if isKey && body["grants"] != nil {
		if !stampGrantTenants(map[string]any{"tenantId": current["tenantId"], "grants": body["grants"]}) {
			http.Error(w, "agent key grants differ from record tenant", http.StatusBadRequest)
			return
		}
	}
	if p == "/iris/v1/agent-memory/stores/"+storeID && body["name"] == "updated" {
		assert.NotContains(f.t, body, "longTermMemoryExclusions")
	}
	previousRevision := current["revision"]
	f.updates++
	delete(body, "expectedRevision")
	merge(current, body)
	f.revisions++
	current["revision"] = fmt.Sprint(f.revisions)
	if isType {
		revision, err := strconv.Atoi(previousRevision.(string))
		require.NoError(f.t, err)
		current["revision"] = strconv.Itoa(revision + 1)
	}
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
func factories(t *testing.T) map[string]func() (tfprotov5.ProviderServer, error) {
	t.Helper()
	// Local mux tests must not inherit live endpoints or account credentials.
	for _, name := range []string{
		rediscloudApi.RedisCloudUrlEnvVar, rediscloudApi.AccessKeyEnvVar, rediscloudApi.SecretKeyEnvVar,
		"REDISCLOUD_IRIS_AGENT_MEMORY_URL", "REDISCLOUD_IRIS_AGENT_MEMORY_TOKEN",
		"REDISCLOUD_IRIS_IDENTITY_URL", "REDISCLOUD_IRIS_IDENTITY_TOKEN",
	} {
		t.Setenv(name, "")
	}
	return map[string]func() (tfprotov5.ProviderServer, error){"rediscloud": func() (tfprotov5.ProviderServer, error) {
		f, err := provider.MuxProviderServerCreator(provider.NewSdkProvider("test")(), provider.NewFrameworkProvider("test")())
		if err != nil {
			return nil, err
		}
		return f(), nil
	}}
}
func TestIrisResourcesLifecycle(t *testing.T) {
	// The mux fixture must ignore a developer's incomplete Cloud environment.
	t.Setenv(rediscloudApi.AccessKeyEnvVar, "outside-test-key")
	t.Setenv(rediscloudApi.SecretKeyEnvVar, "")
	f, server := newFixture(t)
	const key = "rediscloud_iris_identity_api_key.key"
	const store = "rediscloud_iris_agent_memory.store"
	steps := []resource.TestStep{
		{ConfigFile: config.StaticFile("testdata/lifecycle.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy"), "identity_url": config.StringVariable(server.URL + "/identity-proxy"), "name": config.StringVariable("initial")}, Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr(store, "id", storeID), resource.TestCheckResourceAttr(store, "state", "READY"), resource.TestCheckNoResourceAttr(store, "llm_api_key"), resource.TestCheckNoResourceAttr(store, "embedding_api_key"), resource.TestCheckResourceAttr(key, "token", "create-only-secret"))},
		{ConfigFile: config.StaticFile("testdata/lifecycle.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy"), "identity_url": config.StringVariable(server.URL + "/identity-proxy"), "name": config.StringVariable("initial"), "model_key_suffix": config.StringVariable("-unversioned")}, PlanOnly: true},
		{ConfigFile: config.StaticFile("testdata/lifecycle.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy"), "identity_url": config.StringVariable(server.URL + "/identity-proxy"), "name": config.StringVariable("initial"), "key_name": config.StringVariable("renamed")}},
		{ConfigFile: config.StaticFile("testdata/lifecycle.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy"), "identity_url": config.StringVariable(server.URL + "/identity-proxy"), "name": config.StringVariable("updated")}, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectKnownValue(key, tfjsonpath.New("token"), knownvalue.StringExact("create-only-secret")), plancheck.ExpectKnownValue(key, tfjsonpath.New("tenant"), knownvalue.StringExact("tenant-1")), plancheck.ExpectKnownValue(store, tfjsonpath.New("created_at"), knownvalue.StringExact("2026-11-19T12:00:00Z"))}}, Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr(store, "name", "updated"), resource.TestCheckResourceAttr(store, "llm_model", "gpt-4.1-mini"), resource.TestCheckNoResourceAttr(store, "llm_api_key"), resource.TestCheckResourceAttr(key, "token", "create-only-secret"), resource.TestCheckResourceAttr(key, "expires_at", "0"))},
	}
	steps = append(steps,
		resource.TestStep{ConfigFile: config.StaticFile("testdata/lifecycle.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy"), "identity_url": config.StringVariable(server.URL + "/identity-proxy"), "name": config.StringVariable("updated")}, PlanOnly: true},
		resource.TestStep{ConfigFile: config.StaticFile("testdata/lifecycle.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy"), "identity_url": config.StringVariable(server.URL + "/identity-proxy"), "name": config.StringVariable("updated"), "embedding_model": config.StringVariable("text-embedding-3-large")}, PlanOnly: true, ExpectNonEmptyPlan: true, ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectResourceAction(store, plancheck.ResourceActionDestroyBeforeCreate)}}},
	)
	for _, entry := range []struct {
		address, id string
		ignore      []string
	}{
		{store, storeID, []string{"custom_memory_type", "llm_api_key_version", "embedding_api_key_version"}},
		{"rediscloud_iris_agent_memory_type.profile", storeID + "/profile", nil},
	} {
		steps = append(steps, resource.TestStep{ConfigFile: config.StaticFile("testdata/lifecycle.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy"), "identity_url": config.StringVariable(server.URL + "/identity-proxy"), "name": config.StringVariable("updated")}, ResourceName: entry.address, ImportState: true, ImportStateId: entry.id, ImportStateVerify: true, ImportStateVerifyIgnore: entry.ignore})
	}
	for i := range steps {
		steps[i].ProtoV5ProviderFactories = factories(t)
	}
	resource.UnitTest(t, resource.TestCase{Steps: steps})
	f.mu.Lock()
	defer f.mu.Unlock()
	require.Empty(t, f.objects)
	require.Equal(t, 3, f.creates)
	require.Equal(t, 3, f.deletes)
	require.Equal(t, 5, f.updates)
}

func TestAPIKeyRejectsJSONAndConstraints(t *testing.T) {
	for _, file := range []string{"testdata/key_invalid_json.tf", "testdata/key_invalid_constraints.tf"} {
		t.Run(file, func(t *testing.T) {
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: factories(t), Steps: []resource.TestStep{{ConfigFile: config.StaticFile(file), PlanOnly: true, ExpectError: regexp.MustCompile("Unsupported argument")}}})
		})
	}
}

func merge(current, patch map[string]any) {
	for key, value := range patch {
		if key == "summarization" || key == "longTermMemoryExclusions" {
			group := value.(map[string]any)
			old, _ := current[key].(map[string]any)
			if group["enabled"] == nil {
				group["enabled"] = false
			}
			if key == "summarization" {
				if group["triggerStrategy"] == nil {
					group["triggerStrategy"] = "EVENT_COUNT"
				}
				if group["eventCount"] == nil {
					group["eventCount"] = old["eventCount"]
				}
			} else {
				for _, name := range []string{"semantic", "builtInDetectors", "customDetectors"} {
					child, ok := group[name].(map[string]any)
					stored, _ := old[name].(map[string]any)
					if !ok {
						group[name] = old[name]
						continue
					}
					if child["enabled"] == nil {
						child["enabled"] = false
					}
					if name == "semantic" {
						if child["prompt"] == nil || child["prompt"] == "" {
							child["prompt"] = stored["prompt"]
						}
					} else if child["detectors"] == nil {
						child["detectors"] = stored["detectors"]
					}
				}
			}
			current[key] = group
			continue
		}
		if nested, ok := value.(map[string]any); ok {
			if old, ok := current[key].(map[string]any); ok {
				merge(old, nested)
				continue
			}
		}
		current[key] = value
	}
}

func stampGrantTenants(record map[string]any) bool {
	if grants, ok := record["grants"].([]any); ok {
		for _, raw := range grants {
			grant := raw.(map[string]any)
			for _, field := range []string{"product", "resourceType"} {
				grant[field] = strings.ToLower(strings.TrimSpace(grant[field].(string)))
			}
			if grant["tenant"] == nil || strings.TrimSpace(grant["tenant"].(string)) == "" {
				grant["tenant"] = record["tenantId"]
			} else {
				grant["tenant"] = strings.TrimSpace(grant["tenant"].(string))
			}
			if grant["tenant"] != record["tenantId"] {
				return false
			}
		}
	}
	return true
}

func memoryTypeRetirementKey(path string) string {
	index := strings.LastIndex(path, "/")
	return path[:index+1] + strings.ToLower(path[index+1:])
}
