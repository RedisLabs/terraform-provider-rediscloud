package unit_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAgentMemoryTimeoutUpdatesStayLocal(t *testing.T) {
	for _, fixture := range []string{"store_timeouts.tf", "store_advanced.tf", "store_disabled_features.tf"} {
		t.Run(fixture, func(t *testing.T) {
			testAgentMemoryTimeoutUpdatesStayLocal(t, fixture)
		})
	}
}

func testAgentMemoryTimeoutUpdatesStayLocal(t *testing.T, fixture string) {
	t.Helper()
	var mu sync.Mutex
	creates, patches, deletes := 0, 0, 0
	var store map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "POST /memory-stores":
			if err := json.NewDecoder(r.Body).Decode(&store); err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			creates++
			_, _ = w.Write([]byte(`{"taskId":"fixture"}`))
		case "PATCH /memory-stores/store-1":
			patches++
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write([]byte(`{"message":"This local change must not send a PATCH"}`))
		case "GET /tasks/fixture":
			_, _ = w.Write([]byte(`{"taskId":"fixture","status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
		case "GET /memory-stores/store-1":
			if deletes > 0 {
				http.NotFound(w, r)
				return
			}
			response := map[string]any{
				"storeId": "store-1", "name": "local-timeouts", "databaseId": "123", "status": "READY",
				"shortMemory":        map[string]any{"ttlSeconds": 86400},
				"longTermMemory":     map[string]any{"ttlSeconds": 604800},
				"extractionCadence":  map[string]any{"activeIntervalSeconds": 300},
				"extractionStrategy": "INSTRUCT", "createdAt": "2026-10-30T12:00:00Z",
				"endpoint":  "https://memory.example.test",
				"endpoints": []map[string]any{{"url": "https://memory.example.test", "provider": "AWS", "region": "us-east-1"}},
			}
			for name, value := range store {
				response[name] = value
			}
			response["databaseId"] = "123"
			_ = json.NewEncoder(w).Encode(response)
		case "DELETE /memory-stores/store-1":
			deletes++
			_, _ = w.Write([]byte(`{"taskId":"fixture"}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	testProviderConfig(t, server.URL)
	const address = "rediscloud_agent_memory.example"
	check := resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttr(address, "id", "store-1"),
		resource.TestCheckResourceAttr(address, "status", "READY"),
		resource.TestCheckResourceAttr(address, "created_at", "2026-10-30T12:00:00Z"),
		resource.TestCheckResourceAttr(address, "endpoint", "https://memory.example.test"),
		resource.TestCheckResourceAttr(address, "endpoints.0.url", "https://memory.example.test"),
		resource.TestCheckResourceAttr(address, "short_term_ttl_seconds", "86400"),
		resource.TestCheckResourceAttr(address, "long_term_ttl_seconds", "604800"),
		resource.TestCheckResourceAttr(address, "extraction_cadence_seconds", "300"),
	)
	steps := make([]resource.TestStep, 0, 4)
	for _, duration := range []string{"5m", "7m", ""} {
		variables := config.Variables{"name": config.StringVariable("local-timeouts")}
		checks := []resource.TestCheckFunc{check}
		if duration != "" {
			variables["operation_timeout"] = config.StringVariable(duration)
			checks = append(checks, resource.TestCheckResourceAttr(address, "timeouts.0.update", duration))
		} else {
			checks = append(checks, resource.TestCheckNoResourceAttr(address, "timeouts.0.update"))
		}
		steps = append(steps, resource.TestStep{ConfigFile: config.StaticFile("testdata/" + fixture), ConfigVariables: variables, Check: resource.ComposeAggregateTestCheckFunc(checks...)})
	}
	steps = append(steps, resource.TestStep{ConfigFile: config.StaticFile("testdata/" + fixture), ConfigVariables: config.Variables{"name": config.StringVariable("local-timeouts")}, PlanOnly: true})
	resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: steps})
	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, 1, creates)
	require.Zero(t, patches, "local timeout changes must not call the API")
	require.Equal(t, 1, deletes)
}
