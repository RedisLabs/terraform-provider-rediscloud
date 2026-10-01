package agentmemory_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentmemoryapi "github.com/RedisLabs/rediscloud-go-api/service/agentmemory"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAgentMemoryResource_mockedCreateReadDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
			var request agentmemoryapi.CreateStore
			require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
			assert.Equal(t, "store", request.Name)
			assert.Equal(t, 123, request.DatabaseID)
			require.NotNil(t, request.ShortMemory)
			assert.Equal(t, 86400, request.ShortMemory.TTLSeconds)
			require.NotNil(t, request.LongTermMemory)
			assert.Equal(t, 31536000, request.LongTermMemory.TTLSeconds)
			require.NotNil(t, request.ExtractionCadence)
			assert.Equal(t, 300, request.ExtractionCadence.ActiveIntervalSeconds)

			_, _ = w.Write([]byte(`{"taskId":"create-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/create-task":
			_, _ = w.Write([]byte(`{"taskId":"create-task","status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1":
			_, _ = w.Write([]byte(`{
				"storeId":"store-1",
				"name":"store",
				"databaseId":"123",
				"shortMemory":{"ttlSeconds":86400},
				"longTermMemory":{"ttlSeconds":31536000},
				"extractionCadence":{"activeIntervalSeconds":300},
				"extractionStrategy":"INSTRUCT",
				"endpoint":"https://aws-us-east-1.memory.redis.io",
				"endpoints":[
					{"url":"https://gcp-us-east4.memory.redis.io","provider":"GCP","region":"us-east4","egressIps":[],"isAccessible":false},
					{"url":"https://aws-us-east-1.memory.redis.io","provider":"AWS","region":"us-east-1","egressIps":["10.0.0.1"],"isAccessible":true}
				],
				"status":"READY",
				"createdAt":"2026-09-22T12:00:00Z"
			}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-1":
			_, _ = w.Write([]byte(`{"taskId":"delete-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/delete-task":
			_, _ = w.Write([]byte(`{"taskId":"delete-task","status":"processing-completed"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig(server.URL) + `
resource "rediscloud_agent_memory" "example" {
  name                       = "store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "name", "store"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "database_id", "123"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "short_term_ttl_seconds", "86400"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "long_term_ttl_seconds", "31536000"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "extraction_cadence_seconds", "300"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "extraction_strategy", "INSTRUCT"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "endpoint", "https://aws-us-east-1.memory.redis.io"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "endpoints.#", "2"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "endpoints.0.url", "https://aws-us-east-1.memory.redis.io"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "endpoints.0.provider", "AWS"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "endpoints.0.region", "us-east-1"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "endpoints.0.egress_ips.#", "1"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "endpoints.0.egress_ips.0", "10.0.0.1"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "endpoints.0.is_accessible", "true"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "endpoints.1.url", "https://gcp-us-east4.memory.redis.io"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "endpoints.1.provider", "GCP"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "endpoints.1.region", "us-east4"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "endpoints.1.egress_ips.#", "0"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "endpoints.1.is_accessible", "false"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "status", "READY"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "created_at", "2026-09-22T12:00:00Z"),
				),
			},
		},
	})
}

func TestAgentMemoryResource_mockedCreateWithAPIDefaults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
			var request agentmemoryapi.CreateStore
			require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
			assert.Equal(t, "store", request.Name)
			assert.Equal(t, 123, request.DatabaseID)
			assert.Nil(t, request.ShortMemory)
			assert.Nil(t, request.LongTermMemory)
			assert.Nil(t, request.ExtractionCadence)

			_, _ = w.Write([]byte(`{"taskId":"create-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/create-task":
			_, _ = w.Write([]byte(`{"taskId":"create-task","status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1":
			_, _ = w.Write([]byte(`{
				"storeId":"store-1",
				"name":"store",
				"databaseId":"123",
				"shortMemory":{"ttlSeconds":86400},
				"longTermMemory":{"ttlSeconds":31536000},
				"extractionCadence":{"activeIntervalSeconds":300},
				"extractionStrategy":"INSTRUCT",
				"summarization":{"enabled":true,"triggerStrategy":"event_count","eventCount":{"threshold":20,"retainCount":10}},
				"longTermMemoryExclusions":{"enabled":false,"semantic":{"enabled":false},"builtInDetectors":{"enabled":false},"customDetectors":{"enabled":false}},
				"customMemoryTypes":[{"name":"api_default","description":"Returned by the API","fields":[{"name":"field1","description":"field","type":"str"}]}],
				"endpoints":[{"url":"https://aws-us-east-1.memory.redis.io","provider":"AWS","region":"us-east-1","egressIps":["10.0.0.1"],"isAccessible":true}],
				"status":"READY",
				"createdAt":"2026-09-22T12:00:00Z"
			}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-1":
			_, _ = w.Write([]byte(`{"taskId":"delete-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/delete-task":
			_, _ = w.Write([]byte(`{"taskId":"delete-task","status":"processing-completed"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig(server.URL) + `
resource "rediscloud_agent_memory" "example" {
  name        = "store"
  database_id = 123
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "name", "store"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "short_term_ttl_seconds", "86400"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "long_term_ttl_seconds", "31536000"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "extraction_cadence_seconds", "300"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "extraction_strategy", "INSTRUCT"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "endpoint", "https://aws-us-east-1.memory.redis.io"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "endpoints.#", "1"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "endpoints.0.egress_ips.0", "10.0.0.1"),
				),
			},
		},
	})
}

func TestAgentMemoryResource_mockedCreateWithAdvancedConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
			var request agentmemoryapi.CreateStore
			require.NoError(t, json.NewDecoder(r.Body).Decode(&request))

			require.NotNil(t, request.Summarization)
			assert.True(t, request.Summarization.Enabled)
			assert.Equal(t, agentmemoryapi.SummarizationTriggerStrategyEventCount, request.Summarization.TriggerStrategy)
			require.NotNil(t, request.Summarization.EventCount)
			assert.Equal(t, 21, request.Summarization.EventCount.Threshold)
			assert.Equal(t, 11, request.Summarization.EventCount.RetainCount)

			require.NotNil(t, request.LongTermMemoryExclusions)
			assert.True(t, request.LongTermMemoryExclusions.Enabled)
			require.NotNil(t, request.LongTermMemoryExclusions.Semantic)
			assert.True(t, request.LongTermMemoryExclusions.Semantic.Enabled)
			assert.Equal(t, "test", request.LongTermMemoryExclusions.Semantic.Prompt)
			require.NotNil(t, request.LongTermMemoryExclusions.BuiltInDetectors)
			assert.True(t, request.LongTermMemoryExclusions.BuiltInDetectors.Enabled)
			assert.ElementsMatch(t, []agentmemoryapi.DetectorSelection{
				{ID: "credit-card", Enabled: true, Action: "redact"},
				{ID: "email", Enabled: true, Action: "redact"},
			}, request.LongTermMemoryExclusions.BuiltInDetectors.Detectors)
			require.NotNil(t, request.LongTermMemoryExclusions.CustomDetectors)
			assert.True(t, request.LongTermMemoryExclusions.CustomDetectors.Enabled)
			require.Len(t, request.LongTermMemoryExclusions.CustomDetectors.Detectors, 1)
			assert.Equal(t, agentmemoryapi.CustomDetector{
				Name:    "detector",
				Enabled: true,
				Action:  "redact",
				Matcher: &agentmemoryapi.MatcherSpec{
					Kind:  "regex",
					Regex: &agentmemoryapi.RegexSpec{Pattern: "ACCT-[9]"},
				},
			}, request.LongTermMemoryExclusions.CustomDetectors.Detectors[0])

			require.Len(t, request.CustomMemoryTypes, 1)
			customType := request.CustomMemoryTypes[0]
			assert.Equal(t, "custom_name", customType.Name)
			assert.Equal(t, "test", customType.Description)
			assert.Equal(t, []agentmemoryapi.CustomField{
				{Name: "field1", Description: "capture the field", Type: "str"},
			}, customType.Fields)
			require.NotNil(t, customType.ExtractionStrategy)
			assert.Equal(t, "test", customType.ExtractionStrategy.Prompt)
			require.NotNil(t, customType.ExtractionStrategy.Enabled)
			assert.True(t, *customType.ExtractionStrategy.Enabled)

			_, _ = w.Write([]byte(`{"taskId":"create-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/create-task":
			_, _ = w.Write([]byte(`{"taskId":"create-task","status":"processing-completed","response":{"resource":{"storeId":"store-advanced"}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-advanced":
			writeAdvancedStoreResponse(t, w, "store")
		case r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-advanced":
			_, _ = w.Write([]byte(`{"taskId":"delete-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/delete-task":
			_, _ = w.Write([]byte(`{"taskId":"delete-task","status":"processing-completed"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig(server.URL) + advancedAgentMemoryConfig("store"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-advanced"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "summarization.enabled", "true"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "summarization.event_count.threshold", "21"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "summarization.event_count.retain_count", "11"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "long_term_memory_exclusions.enabled", "true"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "long_term_memory_exclusions.semantic.prompt", "test"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "custom_memory_types.#", "1"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "custom_memory_types.0.name", "custom_name"),
				),
			},
			{
				ResourceName:      "rediscloud_agent_memory.example",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAgentMemoryResource_mockedCreateWithModelConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
			var request agentmemoryapi.CreateStore
			require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
			require.NotNil(t, request.LLM)
			assert.Equal(t, "openai", request.LLM.Provider)
			assert.Equal(t, "gpt-4o-mini", request.LLM.Model)
			require.NotNil(t, request.LLM.Credentials)
			assert.Equal(t, "apiKey", request.LLM.Credentials.Type)
			assert.Equal(t, "llm-secret", request.LLM.Credentials.APIKey)

			require.NotNil(t, request.LongTermMemory)
			assert.Equal(t, 31536000, request.LongTermMemory.TTLSeconds)
			require.NotNil(t, request.LongTermMemory.Embedding)
			assert.Equal(t, "openai", request.LongTermMemory.Embedding.Provider)
			assert.Equal(t, "text-embedding-3-small", request.LongTermMemory.Embedding.Model)
			require.NotNil(t, request.LongTermMemory.Embedding.Credentials)
			assert.Equal(t, "apiKey", request.LongTermMemory.Embedding.Credentials.Type)
			assert.Equal(t, "embedding-secret", request.LongTermMemory.Embedding.Credentials.APIKey)

			_, _ = w.Write([]byte(`{"taskId":"create-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/create-task":
			_, _ = w.Write([]byte(`{"taskId":"create-task","status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1":
			writeStoreResponseWithModelConfig(t, w, "store", "gpt-4o-mini", "text-embedding-3-small")
		case r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-1":
			_, _ = w.Write([]byte(`{"taskId":"delete-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/delete-task":
			_, _ = w.Write([]byte(`{"taskId":"delete-task","status":"processing-completed"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig(server.URL) + agentMemoryModelConfig("store", "gpt-4o-mini", "text-embedding-3-small", "llm-secret", "embedding-secret"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "llm.provider", "openai"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "llm.model", "gpt-4o-mini"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "llm.credentials.type", "apiKey"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "llm.credentials.api_key", "llm-secret"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "embedding.provider", "openai"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "embedding.model", "text-embedding-3-small"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "embedding.credentials.type", "apiKey"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "embedding.credentials.api_key", "embedding-secret"),
				),
			},
		},
	})
}

func TestAgentMemoryResource_mockedUpdateAdvancedConfig(t *testing.T) {
	threshold := 21
	retainCount := 11
	semanticPrompt := "test"
	emailAction := "redact"
	patches := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
			_, _ = w.Write([]byte(`{"taskId":"create-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/create-task":
			_, _ = w.Write([]byte(`{"taskId":"create-task","status":"processing-completed","response":{"resource":{"storeId":"store-advanced"}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-advanced":
			writeAdvancedStoreResponseWithValues(t, w, "store", threshold, retainCount, semanticPrompt, emailAction)
		case r.Method == http.MethodPatch && r.URL.Path == "/memory-stores/store-advanced":
			var request agentmemoryapi.UpdateStore
			require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
			assert.Nil(t, request.Name)
			assert.Nil(t, request.ShortMemory)
			assert.Nil(t, request.LongTermMemory)
			assert.Nil(t, request.ExtractionCadence)
			assert.Empty(t, request.AddCustomMemoryTypes)
			assert.Empty(t, request.UpdateCustomMemoryTypeStrategies)

			require.NotNil(t, request.Summarization)
			require.NotNil(t, request.Summarization.EventCount)
			assert.Equal(t, 30, request.Summarization.EventCount.Threshold)
			assert.Equal(t, 12, request.Summarization.EventCount.RetainCount)

			require.NotNil(t, request.LongTermMemoryExclusions)
			require.NotNil(t, request.LongTermMemoryExclusions.Semantic)
			assert.Equal(t, "updated", request.LongTermMemoryExclusions.Semantic.Prompt)
			require.NotNil(t, request.LongTermMemoryExclusions.BuiltInDetectors)
			require.Len(t, request.LongTermMemoryExclusions.BuiltInDetectors.Detectors, 2)
			assert.Equal(t, "drop", request.LongTermMemoryExclusions.BuiltInDetectors.Detectors[1].Action)

			patches++
			threshold = 30
			retainCount = 12
			semanticPrompt = "updated"
			emailAction = "drop"
			_, _ = w.Write([]byte(`{"taskId":"update-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/update-task":
			_, _ = w.Write([]byte(`{"taskId":"update-task","status":"processing-completed"}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-advanced":
			_, _ = w.Write([]byte(`{"taskId":"delete-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/delete-task":
			_, _ = w.Write([]byte(`{"taskId":"delete-task","status":"processing-completed"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig(server.URL) + advancedAgentMemoryConfig("store"),
			},
			{
				Config: testProviderConfig(server.URL) + advancedAgentMemoryConfigWithValues("store", 30, 12, "updated", "drop"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-advanced"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "summarization.event_count.threshold", "30"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "summarization.event_count.retain_count", "12"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "long_term_memory_exclusions.semantic.prompt", "updated"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "long_term_memory_exclusions.built_in_detectors.detectors.1.action", "drop"),
				),
			},
		},
	})

	assert.Equal(t, 1, patches)
}

func TestAgentMemoryResource_mockedUpdateCustomMemoryTypeStrategy(t *testing.T) {
	customPrompt := "test"
	customEnabled := true
	creates := 0
	patches := 0
	deletes := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
			creates++
			_, _ = w.Write([]byte(`{"taskId":"create-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/create-task":
			_, _ = w.Write([]byte(`{"taskId":"create-task","status":"processing-completed","response":{"resource":{"storeId":"store-advanced"}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-advanced":
			writeAdvancedStoreResponseWithCustomMemoryTypes(t, w, "store", 21, 11, "test", "redact", advancedCustomMemoryTypesResponse(customPrompt, customEnabled))
		case r.Method == http.MethodPatch && r.URL.Path == "/memory-stores/store-advanced":
			var request agentmemoryapi.UpdateStore
			require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
			assert.Empty(t, request.AddCustomMemoryTypes)
			require.Len(t, request.UpdateCustomMemoryTypeStrategies, 1)
			assert.Equal(t, "custom_name", request.UpdateCustomMemoryTypeStrategies[0].TypeName)
			assert.Equal(t, "updated extraction prompt", request.UpdateCustomMemoryTypeStrategies[0].Prompt)
			require.NotNil(t, request.UpdateCustomMemoryTypeStrategies[0].Enabled)
			assert.False(t, *request.UpdateCustomMemoryTypeStrategies[0].Enabled)

			patches++
			customPrompt = "updated extraction prompt"
			customEnabled = false
			_, _ = w.Write([]byte(`{"taskId":"update-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/update-task":
			_, _ = w.Write([]byte(`{"taskId":"update-task","status":"processing-completed"}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-advanced":
			deletes++
			_, _ = w.Write([]byte(`{"taskId":"delete-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/delete-task":
			_, _ = w.Write([]byte(`{"taskId":"delete-task","status":"processing-completed"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig(server.URL) + advancedAgentMemoryConfig("store"),
			},
			{
				Config: testProviderConfig(server.URL) + advancedAgentMemoryConfigWithCustomMemoryStrategy("store", 21, 11, "test", "redact", "updated extraction prompt", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-advanced"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "custom_memory_types.0.extraction_strategy.prompt", "updated extraction prompt"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "custom_memory_types.0.extraction_strategy.enabled", "false"),
				),
			},
		},
	})

	assert.Equal(t, 1, creates)
	assert.Equal(t, 1, patches)
	assert.Equal(t, 1, deletes)
}

func TestAgentMemoryResource_mockedAddCustomMemoryType(t *testing.T) {
	customMemoryTypes := advancedCustomMemoryTypesResponse("test", true)
	creates := 0
	patches := 0
	deletes := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
			creates++
			_, _ = w.Write([]byte(`{"taskId":"create-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/create-task":
			_, _ = w.Write([]byte(`{"taskId":"create-task","status":"processing-completed","response":{"resource":{"storeId":"store-advanced"}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-advanced":
			writeAdvancedStoreResponseWithCustomMemoryTypes(t, w, "store", 21, 11, "test", "redact", customMemoryTypes)
		case r.Method == http.MethodPatch && r.URL.Path == "/memory-stores/store-advanced":
			var request agentmemoryapi.UpdateStore
			require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
			assert.Empty(t, request.UpdateCustomMemoryTypeStrategies)
			require.Len(t, request.AddCustomMemoryTypes, 1)
			assert.Equal(t, "support_case", request.AddCustomMemoryTypes[0].Name)
			assert.Equal(t, "Persistent support case details", request.AddCustomMemoryTypes[0].Description)
			assert.Equal(t, []agentmemoryapi.CustomField{
				{Name: "case_priority", Description: "case priority", Type: "int"},
			}, request.AddCustomMemoryTypes[0].Fields)

			patches++
			customMemoryTypes = advancedCustomMemoryTypesResponseWithAdditional()
			_, _ = w.Write([]byte(`{"taskId":"update-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/update-task":
			_, _ = w.Write([]byte(`{"taskId":"update-task","status":"processing-completed"}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-advanced":
			deletes++
			_, _ = w.Write([]byte(`{"taskId":"delete-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/delete-task":
			_, _ = w.Write([]byte(`{"taskId":"delete-task","status":"processing-completed"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig(server.URL) + advancedAgentMemoryConfig("store"),
			},
			{
				Config: testProviderConfig(server.URL) + advancedAgentMemoryConfigWithAdditionalCustomMemoryType("store"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-advanced"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "custom_memory_types.#", "2"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "custom_memory_types.1.name", "support_case"),
				),
			},
		},
	})

	assert.Equal(t, 1, creates)
	assert.Equal(t, 1, patches)
	assert.Equal(t, 1, deletes)
}

func TestAgentMemoryResource_rejectsCustomMemoryTypeRedefinition(t *testing.T) {
	patches := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
			_, _ = w.Write([]byte(`{"taskId":"create-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/create-task":
			_, _ = w.Write([]byte(`{"taskId":"create-task","status":"processing-completed","response":{"resource":{"storeId":"store-advanced"}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-advanced":
			writeAdvancedStoreResponse(t, w, "store")
		case r.Method == http.MethodPatch && r.URL.Path == "/memory-stores/store-advanced":
			patches++
			_, _ = w.Write([]byte(`{"taskId":"update-task"}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-advanced":
			_, _ = w.Write([]byte(`{"taskId":"delete-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/delete-task":
			_, _ = w.Write([]byte(`{"taskId":"delete-task","status":"processing-completed"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig(server.URL) + advancedAgentMemoryConfig("store"),
			},
			{
				Config:      testProviderConfig(server.URL) + advancedAgentMemoryConfigWithChangedCustomMemoryTypeField("store"),
				ExpectError: regexp.MustCompile(`Unsupported Agent Memory custom memory type redefinition`),
			},
		},
	})

	assert.Equal(t, 0, patches)
}

func TestAgentMemoryResource_rejectsCustomMemoryTypeRemoval(t *testing.T) {
	patches := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
			_, _ = w.Write([]byte(`{"taskId":"create-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/create-task":
			_, _ = w.Write([]byte(`{"taskId":"create-task","status":"processing-completed","response":{"resource":{"storeId":"store-advanced"}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-advanced":
			writeAdvancedStoreResponse(t, w, "store")
		case r.Method == http.MethodPatch && r.URL.Path == "/memory-stores/store-advanced":
			patches++
			_, _ = w.Write([]byte(`{"taskId":"update-task"}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-advanced":
			_, _ = w.Write([]byte(`{"taskId":"delete-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/delete-task":
			_, _ = w.Write([]byte(`{"taskId":"delete-task","status":"processing-completed"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig(server.URL) + advancedAgentMemoryConfig("store"),
			},
			{
				Config:      testProviderConfig(server.URL) + advancedAgentMemoryConfigWithoutCustomMemoryTypes("store"),
				ExpectError: regexp.MustCompile(`Unsupported Agent Memory custom memory type removal`),
			},
		},
	})

	assert.Equal(t, 0, patches)
}

func TestAgentMemoryResource_mockedUpdateDisablesAdvancedConfigWithStaleAPIChildren(t *testing.T) {
	disabled := false
	patches := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
			_, _ = w.Write([]byte(`{"taskId":"create-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/create-task":
			_, _ = w.Write([]byte(`{"taskId":"create-task","status":"processing-completed","response":{"resource":{"storeId":"store-advanced"}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-advanced":
			if disabled {
				writeAdvancedStoreResponseWithDisabledGroups(t, w, "store-disabled")
				return
			}
			writeAdvancedStoreResponse(t, w, "store")
		case r.Method == http.MethodPatch && r.URL.Path == "/memory-stores/store-advanced":
			var request agentmemoryapi.UpdateStore
			require.NoError(t, json.NewDecoder(r.Body).Decode(&request))

			require.NotNil(t, request.Summarization)
			assert.False(t, request.Summarization.Enabled)
			assert.Empty(t, request.Summarization.TriggerStrategy)
			assert.Nil(t, request.Summarization.EventCount)

			require.NotNil(t, request.LongTermMemoryExclusions)
			require.NotNil(t, request.LongTermMemoryExclusions.Semantic)
			assert.False(t, request.LongTermMemoryExclusions.Semantic.Enabled)
			assert.Empty(t, request.LongTermMemoryExclusions.Semantic.Prompt)
			require.NotNil(t, request.LongTermMemoryExclusions.BuiltInDetectors)
			assert.False(t, request.LongTermMemoryExclusions.BuiltInDetectors.Enabled)
			assert.Empty(t, request.LongTermMemoryExclusions.BuiltInDetectors.Detectors)
			require.NotNil(t, request.LongTermMemoryExclusions.CustomDetectors)
			assert.False(t, request.LongTermMemoryExclusions.CustomDetectors.Enabled)
			assert.Empty(t, request.LongTermMemoryExclusions.CustomDetectors.Detectors)

			patches++
			disabled = true
			_, _ = w.Write([]byte(`{"taskId":"update-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/update-task":
			_, _ = w.Write([]byte(`{"taskId":"update-task","status":"processing-completed"}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-advanced":
			_, _ = w.Write([]byte(`{"taskId":"delete-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/delete-task":
			_, _ = w.Write([]byte(`{"taskId":"delete-task","status":"processing-completed"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig(server.URL) + advancedAgentMemoryConfig("store"),
			},
			{
				Config: testProviderConfig(server.URL) + disabledAdvancedAgentMemoryConfig("store-disabled"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "name", "store-disabled"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "summarization.enabled", "false"),
					resource.TestCheckNoResourceAttr("rediscloud_agent_memory.example", "summarization.trigger_strategy"),
					resource.TestCheckNoResourceAttr("rediscloud_agent_memory.example", "summarization.event_count.threshold"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "long_term_memory_exclusions.semantic.enabled", "false"),
					resource.TestCheckNoResourceAttr("rediscloud_agent_memory.example", "long_term_memory_exclusions.semantic.prompt"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "long_term_memory_exclusions.built_in_detectors.enabled", "false"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "long_term_memory_exclusions.built_in_detectors.detectors.#", "0"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "long_term_memory_exclusions.custom_detectors.enabled", "false"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "long_term_memory_exclusions.custom_detectors.detectors.#", "0"),
				),
			},
		},
	})

	assert.Equal(t, 1, patches)
}

func TestAgentMemoryResource_mockedUpdate(t *testing.T) {
	name := "store"
	shortTTL := 86400
	longTTL := 31536000
	cadence := 300

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
			var request agentmemoryapi.CreateStore
			require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
			assert.Equal(t, name, request.Name)
			assert.Equal(t, 123, request.DatabaseID)

			_, _ = w.Write([]byte(`{"taskId":"create-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/create-task":
			_, _ = w.Write([]byte(`{"taskId":"create-task","status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1":
			writeStoreResponse(t, w, name, shortTTL, longTTL, cadence)
		case r.Method == http.MethodPatch && r.URL.Path == "/memory-stores/store-1":
			var request agentmemoryapi.UpdateStore
			require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
			require.NotNil(t, request.Name)
			assert.Equal(t, "store-updated", *request.Name)
			require.NotNil(t, request.ShortMemory)
			assert.Equal(t, 172800, request.ShortMemory.TTLSeconds)
			require.NotNil(t, request.LongTermMemory)
			assert.Equal(t, 604800, request.LongTermMemory.TTLSeconds)
			require.NotNil(t, request.ExtractionCadence)
			assert.Equal(t, 600, request.ExtractionCadence.ActiveIntervalSeconds)

			name = "store-updated"
			shortTTL = 172800
			longTTL = 604800
			cadence = 600
			_, _ = w.Write([]byte(`{"taskId":"update-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/update-task":
			_, _ = w.Write([]byte(`{"taskId":"update-task","status":"processing-completed"}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-1":
			_, _ = w.Write([]byte(`{"taskId":"delete-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/delete-task":
			_, _ = w.Write([]byte(`{"taskId":"delete-task","status":"processing-completed"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig(server.URL) + `
resource "rediscloud_agent_memory" "example" {
  name                       = "store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "name", "store"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "short_term_ttl_seconds", "86400"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "long_term_ttl_seconds", "31536000"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "extraction_cadence_seconds", "300"),
				),
			},
			{
				Config: testProviderConfig(server.URL) + `
resource "rediscloud_agent_memory" "example" {
  name                       = "store-updated"
  database_id                = 123
  short_term_ttl_seconds     = 172800
  long_term_ttl_seconds      = 604800
  extraction_cadence_seconds = 600
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "name", "store-updated"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "short_term_ttl_seconds", "172800"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "long_term_ttl_seconds", "604800"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "extraction_cadence_seconds", "600"),
				),
			},
		},
	})
}

func TestAgentMemoryResource_mockedUpdateModelConfig(t *testing.T) {
	llmModel := "gpt-4o-mini"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
			_, _ = w.Write([]byte(`{"taskId":"create-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/create-task":
			_, _ = w.Write([]byte(`{"taskId":"create-task","status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1":
			writeStoreResponseWithModelConfig(t, w, "store", llmModel, "text-embedding-3-small")
		case r.Method == http.MethodPatch && r.URL.Path == "/memory-stores/store-1":
			var request agentmemoryapi.UpdateStore
			require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
			require.NotNil(t, request.LLM)
			assert.Equal(t, "openai", request.LLM.Provider)
			assert.Equal(t, "gpt-4.1-mini", request.LLM.Model)
			require.NotNil(t, request.LLM.Credentials)
			assert.Equal(t, "llm-secret-2", request.LLM.Credentials.APIKey)

			require.NotNil(t, request.LongTermMemory)
			assert.Equal(t, 31536000, request.LongTermMemory.TTLSeconds)
			require.NotNil(t, request.LongTermMemory.Embedding)
			assert.Equal(t, "openai", request.LongTermMemory.Embedding.Provider)
			assert.Equal(t, "text-embedding-3-small", request.LongTermMemory.Embedding.Model)
			require.NotNil(t, request.LongTermMemory.Embedding.Credentials)
			assert.Equal(t, "embedding-secret-2", request.LongTermMemory.Embedding.Credentials.APIKey)

			llmModel = "gpt-4.1-mini"
			_, _ = w.Write([]byte(`{"taskId":"update-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/update-task":
			_, _ = w.Write([]byte(`{"taskId":"update-task","status":"processing-completed"}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-1":
			_, _ = w.Write([]byte(`{"taskId":"delete-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/delete-task":
			_, _ = w.Write([]byte(`{"taskId":"delete-task","status":"processing-completed"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig(server.URL) + agentMemoryModelConfig("store", "gpt-4o-mini", "text-embedding-3-small", "llm-secret", "embedding-secret"),
			},
			{
				Config: testProviderConfig(server.URL) + agentMemoryModelConfig("store", "gpt-4.1-mini", "text-embedding-3-small", "llm-secret-2", "embedding-secret-2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "llm.model", "gpt-4.1-mini"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "llm.credentials.api_key", "llm-secret-2"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "embedding.model", "text-embedding-3-small"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "embedding.credentials.api_key", "embedding-secret-2"),
				),
			},
		},
	})
}

func TestAgentMemoryResource_updateDoesNotReplaceDependentAPIKey(t *testing.T) {
	name := "store"
	apiKeyExists := false
	apiKeyCreates := 0
	apiKeyDeletes := 0
	storePatches := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
			_, _ = w.Write([]byte(`{"taskId":"create-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/create-task":
			_, _ = w.Write([]byte(`{"taskId":"create-task","status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1":
			writeStoreResponse(t, w, name, 86400, 604800, 300)
		case r.Method == http.MethodPatch && r.URL.Path == "/memory-stores/store-1":
			var request agentmemoryapi.UpdateStore
			require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
			require.NotNil(t, request.Name)
			assert.Equal(t, "store-updated", *request.Name)
			assert.Nil(t, request.ShortMemory)
			assert.Nil(t, request.LongTermMemory)
			assert.Nil(t, request.ExtractionCadence)

			storePatches++
			name = "store-updated"
			_, _ = w.Write([]byte(`{"taskId":"update-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/update-task":
			_, _ = w.Write([]byte(`{"taskId":"update-task","status":"processing-completed"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1/api-keys":
			if !apiKeyExists {
				_, _ = w.Write([]byte(`{"apiKeys":[]}`))
				return
			}
			_, _ = w.Write([]byte(`{"apiKeys":[{"apiKeyId":"key-1","keyName":"terraform","obfuscatedToken":"mem1...RQ==","createdAt":1790078409}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/memory-stores/store-1/api-keys":
			apiKeyCreates++
			apiKeyExists = true
			_, _ = w.Write([]byte(`{"apiKey":"secret-token"}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-1/api-keys/key-1":
			apiKeyDeletes++
			apiKeyExists = false
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-1":
			_, _ = w.Write([]byte(`{"taskId":"delete-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/delete-task":
			_, _ = w.Write([]byte(`{"taskId":"delete-task","status":"processing-completed"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig(server.URL) + `
resource "rediscloud_agent_memory" "example" {
  name                       = "store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 604800
  extraction_cadence_seconds = 300
}

resource "rediscloud_agent_memory_api_key" "example" {
  store_id = rediscloud_agent_memory.example.id
  name     = "terraform"
}
`,
			},
			{
				Config: testProviderConfig(server.URL) + `
resource "rediscloud_agent_memory" "example" {
  name                       = "store-updated"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 604800
  extraction_cadence_seconds = 300
}

resource "rediscloud_agent_memory_api_key" "example" {
  store_id = rediscloud_agent_memory.example.id
  name     = "terraform"
}
`,
			},
		},
	})

	assert.Equal(t, 1, storePatches)
	assert.Equal(t, 1, apiKeyCreates)
	assert.Equal(t, 1, apiKeyDeletes)
}

func TestAgentMemoryResource_mockedImport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
			_, _ = w.Write([]byte(`{"taskId":"create-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/create-task":
			_, _ = w.Write([]byte(`{"taskId":"create-task","status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1":
			writeStoreResponse(t, w, "store", 86400, 31536000, 300)
		case r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-1":
			_, _ = w.Write([]byte(`{"taskId":"delete-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/delete-task":
			_, _ = w.Write([]byte(`{"taskId":"delete-task","status":"processing-completed"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig(server.URL) + `
resource "rediscloud_agent_memory" "example" {
  name                       = "store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
}
`,
			},
			{
				ResourceName:      "rediscloud_agent_memory.example",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAgentMemoryResource_mockedImportWithModelConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1":
			writeStoreResponseWithModelConfig(t, w, "store", "gpt-4o-mini", "text-embedding-3-small")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig(server.URL) + `
resource "rediscloud_agent_memory" "example" {
  name                       = "store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300

  llm {
    provider = "openai"
    model    = "gpt-4o-mini"
  }

  embedding {
    provider = "openai"
    model    = "text-embedding-3-small"
  }
}
`,
				ResourceName:  "rediscloud_agent_memory.example",
				ImportState:   true,
				ImportStateId: "store-1",
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					attrs := states[0].Attributes
					expected := map[string]string{
						"llm.provider":       "openai",
						"llm.model":          "gpt-4o-mini",
						"embedding.provider": "openai",
						"embedding.model":    "text-embedding-3-small",
					}
					for key, want := range expected {
						if got := attrs[key]; got != want {
							return fmt.Errorf("expected imported %s to be %q, got %q", key, want, got)
						}
					}
					if _, ok := attrs["llm.credentials.api_key"]; ok {
						return fmt.Errorf("imported llm credentials api_key should not be set because the Agent Memory API does not return it")
					}
					if _, ok := attrs["embedding.credentials.api_key"]; ok {
						return fmt.Errorf("imported embedding credentials api_key should not be set because the Agent Memory API does not return it")
					}
					return nil
				},
			},
		},
	})
}

func TestAgentMemoryResource_mockedImportWithAdvancedConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-advanced":
			writeAdvancedStoreResponse(t, w, "store")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig(server.URL) + `
resource "rediscloud_agent_memory" "example" {
  name                       = "store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 604800
  extraction_cadence_seconds = 300
}
`,
				ResourceName:  "rediscloud_agent_memory.example",
				ImportState:   true,
				ImportStateId: "store-advanced",
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					attrs := states[0].Attributes
					expected := map[string]string{
						"summarization.enabled":                                      "true",
						"summarization.trigger_strategy":                             "event_count",
						"summarization.event_count.threshold":                        "21",
						"summarization.event_count.retain_count":                     "11",
						"long_term_memory_exclusions.enabled":                        "true",
						"long_term_memory_exclusions.semantic.enabled":               "true",
						"long_term_memory_exclusions.semantic.prompt":                "test",
						"long_term_memory_exclusions.built_in_detectors.enabled":     "true",
						"long_term_memory_exclusions.built_in_detectors.detectors.#": "2",
						"long_term_memory_exclusions.custom_detectors.enabled":       "true",
						"long_term_memory_exclusions.custom_detectors.detectors.#":   "1",
						"custom_memory_types.#":                                      "1",
						"custom_memory_types.0.name":                                 "custom_name",
						"custom_memory_types.0.description":                          "test",
						"custom_memory_types.0.fields.#":                             "1",
						"custom_memory_types.0.fields.0.name":                        "field1",
						"custom_memory_types.0.extraction_strategy.enabled":          "true",
						"custom_memory_types.0.extraction_strategy.prompt":           "test",
					}
					for key, want := range expected {
						if got := attrs[key]; got != want {
							return fmt.Errorf("expected imported %s to be %q, got %q", key, want, got)
						}
					}
					return nil
				},
			},
		},
	})
}

func TestAgentMemoryResource_rejectsInvalidExtractionCadence(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig("http://127.0.0.1:1") + `
resource "rediscloud_agent_memory" "example" {
  name                       = "store"
  database_id                = 123
  extraction_cadence_seconds = 30
}
`,
				ExpectError: regexp.MustCompile(`60.*600|600.*60`),
			},
		},
	})
}

func TestAgentMemoryResource_rejectsInvalidTTLs(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig("http://127.0.0.1:1") + `
resource "rediscloud_agent_memory" "example" {
  name                   = "store"
  database_id            = 123
  short_term_ttl_seconds = 0
  long_term_ttl_seconds  = 0
}
`,
				ExpectError: regexp.MustCompile(`1.*31536000|31536000.*1`),
			},
			{
				Config: testProviderConfig("http://127.0.0.1:1") + `
resource "rediscloud_agent_memory" "example" {
  name                   = "store"
  database_id            = 123
  short_term_ttl_seconds = 31536001
  long_term_ttl_seconds  = 31536001
}
`,
				ExpectError: regexp.MustCompile(`1.*31536000|31536000.*1`),
			},
		},
	})
}

func TestAgentMemoryResource_rejectsIncompleteModelConfig(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig("https://example.invalid") + `
resource "rediscloud_agent_memory" "example" {
  name                  = "store"
  database_id           = 123
  long_term_ttl_seconds = 31536000

  llm {
    provider = "openai"
    model    = "gpt-4o-mini"

    credentials {
      type    = "apiKey"
      api_key = "llm-secret"
    }
  }
}
`,
				ExpectError: regexp.MustCompile("embedding must be configured when llm is configured"),
			},
		},
	})
}

func TestAgentMemoryResource_rejectsModelConfigWithoutLongTermTTL(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig("https://example.invalid") + `
resource "rediscloud_agent_memory" "example" {
  name        = "store"
  database_id = 123

  llm {
    provider = "openai"
    model    = "gpt-4o-mini"

    credentials {
      type    = "apiKey"
      api_key = "llm-secret"
    }
  }

  embedding {
    provider = "openai"
    model    = "text-embedding-3-small"

    credentials {
      type    = "apiKey"
      api_key = "embedding-secret"
    }
  }
}
`,
				ExpectError: regexp.MustCompile("long_term_ttl_seconds must be configured when embedding is configured"),
			},
		},
	})
}

func TestAgentMemoryResource_rejectsAddingModelConfigAfterCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
			_, _ = w.Write([]byte(`{"taskId":"create-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/create-task":
			_, _ = w.Write([]byte(`{"taskId":"create-task","status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1":
			writeStoreResponse(t, w, "store", 86400, 31536000, 300)
		case r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-1":
			_, _ = w.Write([]byte(`{"taskId":"delete-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/delete-task":
			_, _ = w.Write([]byte(`{"taskId":"delete-task","status":"processing-completed"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig(server.URL) + `
resource "rediscloud_agent_memory" "example" {
  name                       = "store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
}
`,
			},
			{
				Config:      testProviderConfig(server.URL) + agentMemoryModelConfig("store", "gpt-4o-mini", "text-embedding-3-small", "llm-secret", "embedding-secret"),
				ExpectError: regexp.MustCompile("Unsupported Agent Memory model configuration change"),
			},
		},
	})
}

func TestAgentMemoryResource_rejectsEmbeddingModelChange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
			_, _ = w.Write([]byte(`{"taskId":"create-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/create-task":
			_, _ = w.Write([]byte(`{"taskId":"create-task","status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1":
			writeStoreResponseWithModelConfig(t, w, "store", "gpt-4o-mini", "text-embedding-3-small")
		case r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-1":
			_, _ = w.Write([]byte(`{"taskId":"delete-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/delete-task":
			_, _ = w.Write([]byte(`{"taskId":"delete-task","status":"processing-completed"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig(server.URL) + agentMemoryModelConfig("store", "gpt-4o-mini", "text-embedding-3-small", "llm-secret", "embedding-secret"),
			},
			{
				Config:      testProviderConfig(server.URL) + agentMemoryModelConfig("store", "gpt-4o-mini", "text-embedding-3-large", "llm-secret", "embedding-secret"),
				ExpectError: regexp.MustCompile("embedding.model as immutable after creation"),
			},
		},
	})
}

func TestAgentMemoryResource_rejectsModelConfigRemoval(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
			_, _ = w.Write([]byte(`{"taskId":"create-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/create-task":
			_, _ = w.Write([]byte(`{"taskId":"create-task","status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1":
			writeStoreResponseWithModelConfig(t, w, "store", "gpt-4o-mini", "text-embedding-3-small")
		case r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-1":
			_, _ = w.Write([]byte(`{"taskId":"delete-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/delete-task":
			_, _ = w.Write([]byte(`{"taskId":"delete-task","status":"processing-completed"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig(server.URL) + agentMemoryModelConfig("store", "gpt-4o-mini", "text-embedding-3-small", "llm-secret", "embedding-secret"),
			},
			{
				Config: testProviderConfig(server.URL) + `
resource "rediscloud_agent_memory" "example" {
  name                       = "store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
}
`,
				ExpectError: regexp.MustCompile("Unsupported Agent Memory model configuration removal"),
			},
		},
	})
}

func TestAgentMemoryResource_rejectsInvalidNameAndDatabaseID(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig("http://127.0.0.1:1") + `
resource "rediscloud_agent_memory" "example" {
  name        = "   "
  database_id = 0
}
`,
				ExpectError: regexp.MustCompile(`non-whitespace|at least 1`),
			},
		},
	})
}

func TestAgentMemoryResource_rejectsInvalidSummarizationThreshold(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig("http://127.0.0.1:1") + `
resource "rediscloud_agent_memory" "example" {
  name        = "store"
  database_id = 123

  summarization {
    enabled          = true
    trigger_strategy = "event_count"

    event_count {
      threshold    = 10
      retain_count = 10
    }
  }
}
`,
				ExpectError: regexp.MustCompile(`threshold must be greater`),
			},
		},
	})
}

func TestAgentMemoryResource_rejectsInvalidDetectorConfiguration(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig("http://127.0.0.1:1") + `
resource "rediscloud_agent_memory" "example" {
  name        = "store"
  database_id = 123

  long_term_memory_exclusions {
    enabled = true

    built_in_detectors {
      enabled = true

      detectors {
        id      = "email"
        enabled = true
        action  = "mask"
      }
    }
  }
}
`,
				ExpectError: regexp.MustCompile(`redact|drop`),
			},
			{
				Config: testProviderConfig("http://127.0.0.1:1") + `
resource "rediscloud_agent_memory" "example" {
  name        = "store"
  database_id = 123

  long_term_memory_exclusions {
    enabled = true

    custom_detectors {
      enabled = true

      detectors {
        name    = "tenant_id"
        enabled = true
        action  = "redact"
      }
    }
  }
}
`,
				ExpectError: regexp.MustCompile(`Missing Configuration for Required Attribute|matcher\.kind|matcher\.regex\.pattern`),
			},
		},
	})
}

func TestAgentMemoryResource_rejectsDuplicateCustomMemoryTypeNames(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      testProviderConfig("http://127.0.0.1:1") + agentMemoryConfigWithDuplicateCustomMemoryTypeNames(),
				ExpectError: regexp.MustCompile(`Duplicate Agent Memory custom memory type`),
			},
		},
	})
}

func TestAgentMemoryResource_deleteSucceedsWhenStoreAlreadyMissing(t *testing.T) {
	storeExists := true

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
			storeExists = true
			_, _ = w.Write([]byte(`{"taskId":"create-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/create-task":
			_, _ = w.Write([]byte(`{"taskId":"create-task","status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1":
			if !storeExists {
				http.NotFound(w, r)
				return
			}
			writeStoreResponse(t, w, "store", 86400, 31536000, 300)
		case r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-1":
			if !storeExists {
				http.NotFound(w, r)
				return
			}
			storeExists = false
			_, _ = w.Write([]byte(`{"taskId":"delete-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/delete-task":
			_, _ = w.Write([]byte(`{"taskId":"delete-task","status":"processing-completed"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig(server.URL) + `
resource "rediscloud_agent_memory" "example" {
  name                       = "store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
}
`,
			},
			{
				PreConfig: func() {
					storeExists = false
				},
				Config: testProviderConfig(server.URL) + `
resource "rediscloud_agent_memory" "example" {
  name                       = "store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
}
`,
				Destroy: true,
			},
		},
	})
}

func TestAgentMemoryResource_removesStateWhenStoreNotFound(t *testing.T) {
	storeExists := true

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
			storeExists = true
			_, _ = w.Write([]byte(`{"taskId":"create-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/create-task":
			_, _ = w.Write([]byte(`{"taskId":"create-task","status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1":
			if !storeExists {
				http.NotFound(w, r)
				return
			}
			writeStoreResponse(t, w, "store", 86400, 31536000, 300)
		case r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-1":
			storeExists = false
			_, _ = w.Write([]byte(`{"taskId":"delete-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/delete-task":
			_, _ = w.Write([]byte(`{"taskId":"delete-task","status":"processing-completed"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig(server.URL) + `
resource "rediscloud_agent_memory" "example" {
  name                       = "store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
}
`,
			},
			{
				PreConfig: func() {
					storeExists = false
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAgentMemoryAPIKeyResource_mockedCreateReadDelete(t *testing.T) {
	apiKeyCreated := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1/api-keys":
			if !apiKeyCreated {
				_, _ = w.Write([]byte(`{"apiKeys":[]}`))
				return
			}
			_, _ = w.Write([]byte(`{"apiKeys":[{"apiKeyId":"key-1","keyName":"terraform","obfuscatedToken":"mem1...RQ==","createdAt":1790078409}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/memory-stores/store-1/api-keys":
			var request agentmemoryapi.CreateAPIKeyRequest
			require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
			assert.Equal(t, "terraform", request.KeyName)
			apiKeyCreated = true

			_, _ = w.Write([]byte(`{"apiKey":"secret-token"}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-1/api-keys/key-1":
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig(server.URL) + `
resource "rediscloud_agent_memory_api_key" "example" {
  store_id = "store-1"
  name     = "terraform"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("rediscloud_agent_memory_api_key.example", "id", "key-1"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory_api_key.example", "store_id", "store-1"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory_api_key.example", "name", "terraform"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory_api_key.example", "api_key", "secret-token"),
					resource.TestCheckResourceAttr("rediscloud_agent_memory_api_key.example", "obfuscated_token", "mem1...RQ=="),
					resource.TestCheckResourceAttr("rediscloud_agent_memory_api_key.example", "created_at", "1790078409"),
				),
			},
		},
	})
}

func TestAgentMemoryAPIKeyResource_mockedImport(t *testing.T) {
	apiKeyCreated := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1/api-keys":
			if !apiKeyCreated {
				_, _ = w.Write([]byte(`{"apiKeys":[]}`))
				return
			}
			_, _ = w.Write([]byte(`{"apiKeys":[{"apiKeyId":"key-1","keyName":"terraform","obfuscatedToken":"mem1...RQ==","createdAt":1790078409}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/memory-stores/store-1/api-keys":
			apiKeyCreated = true
			_, _ = w.Write([]byte(`{"apiKey":"secret-token"}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-1/api-keys/key-1":
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig(server.URL) + `
resource "rediscloud_agent_memory_api_key" "example" {
  store_id = "store-1"
  name     = "terraform"
}
`,
			},
			{
				ResourceName:            "rediscloud_agent_memory_api_key.example",
				ImportState:             true,
				ImportStateId:           "store-1/key-1",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"api_key"},
			},
		},
	})
}

func TestAgentMemoryAPIKeyResource_rejectsInvalidImportID(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig("http://127.0.0.1:1") + `
resource "rediscloud_agent_memory_api_key" "example" {
  store_id = "store-1"
  name     = "terraform"
}
`,
				ResourceName:  "rediscloud_agent_memory_api_key.example",
				ImportState:   true,
				ImportStateId: "store-only",
				ExpectError:   regexp.MustCompile("Invalid import ID"),
			},
		},
	})
}

func TestAgentMemoryAPIKeyResource_rejectsInvalidStoreIDAndName(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig("http://127.0.0.1:1") + `
resource "rediscloud_agent_memory_api_key" "example" {
  store_id = "store/1"
  name     = "invalid name"
}
`,
				ExpectError: regexp.MustCompile(`letters, numbers, and hyphens|must not contain whitespace`),
			},
		},
	})
}

func TestAgentMemoryAPIKeyResource_deleteSucceedsWhenKeyAlreadyMissing(t *testing.T) {
	apiKeyExists := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1/api-keys":
			if !apiKeyExists {
				_, _ = w.Write([]byte(`{"apiKeys":[]}`))
				return
			}
			_, _ = w.Write([]byte(`{"apiKeys":[{"apiKeyId":"key-1","keyName":"terraform","obfuscatedToken":"mem1...RQ==","createdAt":1790078409}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/memory-stores/store-1/api-keys":
			apiKeyExists = true
			_, _ = w.Write([]byte(`{"apiKey":"secret-token"}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-1/api-keys/key-1":
			if !apiKeyExists {
				http.NotFound(w, r)
				return
			}
			apiKeyExists = false
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig(server.URL) + `
resource "rediscloud_agent_memory_api_key" "example" {
  store_id = "store-1"
  name     = "terraform"
}
`,
			},
			{
				PreConfig: func() {
					apiKeyExists = false
				},
				Config: testProviderConfig(server.URL) + `
resource "rediscloud_agent_memory_api_key" "example" {
  store_id = "store-1"
  name     = "terraform"
}
`,
				Destroy: true,
			},
		},
	})
}

func TestAgentMemoryAPIKeyResource_removesStateWhenKeyNotFound(t *testing.T) {
	apiKeyExists := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1/api-keys":
			if !apiKeyExists {
				_, _ = w.Write([]byte(`{"apiKeys":[]}`))
				return
			}
			_, _ = w.Write([]byte(`{"apiKeys":[{"apiKeyId":"key-1","keyName":"terraform","obfuscatedToken":"mem1...RQ==","createdAt":1790078409}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/memory-stores/store-1/api-keys":
			apiKeyExists = true
			_, _ = w.Write([]byte(`{"apiKey":"secret-token"}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-1/api-keys/key-1":
			apiKeyExists = false
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig(server.URL) + `
resource "rediscloud_agent_memory_api_key" "example" {
  store_id = "store-1"
  name     = "terraform"
}
`,
			},
			{
				PreConfig: func() {
					apiKeyExists = false
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testProviderConfig(url string) string {
	return `
provider "rediscloud" {
  url        = "` + url + `"
  api_key    = "test-api-key"
  secret_key = "test-secret-key"
}
`
}

func writeStoreResponse(t *testing.T, w http.ResponseWriter, name string, shortTTL, longTTL, cadence int) {
	t.Helper()
	require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
		"storeId":            "store-1",
		"name":               name,
		"databaseId":         "123",
		"shortMemory":        map[string]any{"ttlSeconds": shortTTL},
		"longTermMemory":     map[string]any{"ttlSeconds": longTTL},
		"extractionCadence":  map[string]any{"activeIntervalSeconds": cadence},
		"extractionStrategy": "INSTRUCT",
		"endpoint":           "https://aws-us-east-1.memory.redis.io",
		"endpoints": []map[string]any{
			{
				"url":          "https://aws-us-east-1.memory.redis.io",
				"provider":     "AWS",
				"region":       "us-east-1",
				"egressIps":    []string{"10.0.0.1"},
				"isAccessible": true,
			},
		},
		"status":    "READY",
		"createdAt": "2026-09-22T12:00:00Z",
	}))
}

func writeStoreResponseWithModelConfig(t *testing.T, w http.ResponseWriter, name, llmModel, embeddingModel string) {
	t.Helper()
	require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
		"storeId":     "store-1",
		"name":        name,
		"databaseId":  "123",
		"shortMemory": map[string]any{"ttlSeconds": 86400},
		"longTermMemory": map[string]any{
			"ttlSeconds": 31536000,
			"embedding": map[string]any{
				"provider": "openai",
				"model":    embeddingModel,
			},
		},
		"llm": map[string]any{
			"provider": "openai",
			"model":    llmModel,
		},
		"extractionCadence":  map[string]any{"activeIntervalSeconds": 300},
		"extractionStrategy": "INSTRUCT",
		"endpoint":           "https://aws-us-east-1.memory.redis.io",
		"endpoints": []map[string]any{
			{
				"url":          "https://aws-us-east-1.memory.redis.io",
				"provider":     "AWS",
				"region":       "us-east-1",
				"egressIps":    []string{"10.0.0.1"},
				"isAccessible": true,
			},
		},
		"status":    "READY",
		"createdAt": "2026-09-22T12:00:00Z",
	}))
}

func advancedCustomMemoryTypesResponse(prompt string, enabled bool) []map[string]any {
	return []map[string]any{
		{
			"name":        "custom_name",
			"description": "test",
			"fields": []map[string]any{
				{"name": "field1", "description": "capture the field", "type": "str"},
			},
			"extractionStrategy": map[string]any{
				"enabled": enabled,
				"prompt":  prompt,
			},
		},
	}
}

func agentMemoryModelConfig(name, llmModel, embeddingModel, llmAPIKey, embeddingAPIKey string) string {
	return fmt.Sprintf(`
resource "rediscloud_agent_memory" "example" {
  name                       = %[1]q
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300

  llm {
    provider = "openai"
    model    = %[2]q

    credentials {
      type    = "apiKey"
      api_key = %[4]q
    }
  }

  embedding {
    provider = "openai"
    model    = %[3]q

    credentials {
      type    = "apiKey"
      api_key = %[5]q
    }
  }
}
`, name, llmModel, embeddingModel, llmAPIKey, embeddingAPIKey)
}

func advancedCustomMemoryTypesResponseWithAdditional() []map[string]any {
	memoryTypes := advancedCustomMemoryTypesResponse("test", true)
	memoryTypes = append(memoryTypes, map[string]any{
		"name":        "support_case",
		"description": "Persistent support case details",
		"fields": []map[string]any{
			{"name": "case_priority", "description": "case priority", "type": "int"},
		},
		"extractionStrategy": map[string]any{
			"enabled": true,
			"prompt":  "Extract durable support case details.",
		},
	})
	return memoryTypes
}

func writeAdvancedStoreResponse(t *testing.T, w http.ResponseWriter, name string) {
	writeAdvancedStoreResponseWithValues(t, w, name, 21, 11, "test", "redact")
}

func writeAdvancedStoreResponseWithValues(t *testing.T, w http.ResponseWriter, name string, threshold, retainCount int, semanticPrompt, emailAction string) {
	t.Helper()
	writeAdvancedStoreResponseWithCustomMemoryTypes(t, w, name, threshold, retainCount, semanticPrompt, emailAction, advancedCustomMemoryTypesResponse("test", true))
}

func writeAdvancedStoreResponseWithCustomMemoryTypes(t *testing.T, w http.ResponseWriter, name string, threshold, retainCount int, semanticPrompt, emailAction string, customMemoryTypes []map[string]any) {
	t.Helper()
	require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
		"storeId":            "store-advanced",
		"name":               name,
		"databaseId":         "123",
		"shortMemory":        map[string]any{"ttlSeconds": 86400},
		"longTermMemory":     map[string]any{"ttlSeconds": 604800},
		"extractionCadence":  map[string]any{"activeIntervalSeconds": 300},
		"extractionStrategy": "INSTRUCT",
		"summarization": map[string]any{
			"enabled":         true,
			"triggerStrategy": "event_count",
			"eventCount": map[string]any{
				"threshold":   threshold,
				"retainCount": retainCount,
			},
		},
		"longTermMemoryExclusions": map[string]any{
			"enabled": true,
			"semantic": map[string]any{
				"enabled": true,
				"prompt":  semanticPrompt,
			},
			"builtInDetectors": map[string]any{
				"enabled": true,
				"detectors": []map[string]any{
					{"id": "credit-card", "enabled": true, "action": "redact"},
					{"id": "email", "enabled": true, "action": emailAction},
				},
			},
			"customDetectors": map[string]any{
				"enabled": true,
				"detectors": []map[string]any{
					{
						"name":    "detector",
						"enabled": true,
						"action":  "redact",
						"matcher": map[string]any{
							"kind":  "regex",
							"regex": map[string]any{"pattern": "ACCT-[9]"},
						},
					},
				},
			},
		},
		"customMemoryTypes": customMemoryTypes,
		"endpoint":          "https://aws-us-east-1.memory.redis.io",
		"endpoints": []map[string]any{
			{
				"url":          "https://aws-us-east-1.memory.redis.io",
				"provider":     "AWS",
				"region":       "us-east-1",
				"egressIps":    []string{"10.0.0.1"},
				"isAccessible": true,
			},
		},
		"status":    "READY",
		"createdAt": "2026-09-22T12:00:00Z",
	}))
}

func writeAdvancedStoreResponseWithDisabledGroups(t *testing.T, w http.ResponseWriter, name string) {
	t.Helper()
	enabled := true
	require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
		"storeId":            "store-advanced",
		"name":               name,
		"databaseId":         "123",
		"shortMemory":        map[string]any{"ttlSeconds": 86400},
		"longTermMemory":     map[string]any{"ttlSeconds": 604800},
		"extractionCadence":  map[string]any{"activeIntervalSeconds": 300},
		"extractionStrategy": "INSTRUCT",
		"summarization": map[string]any{
			"enabled":         false,
			"triggerStrategy": "event_count",
			"eventCount": map[string]any{
				"threshold":   21,
				"retainCount": 11,
			},
		},
		"longTermMemoryExclusions": map[string]any{
			"enabled": true,
			"semantic": map[string]any{
				"enabled": false,
				"prompt":  "stale semantic prompt",
			},
			"builtInDetectors": map[string]any{
				"enabled": false,
				"detectors": []map[string]any{
					{"id": "credit-card", "enabled": true, "action": "redact"},
					{"id": "email", "enabled": true, "action": "redact"},
				},
			},
			"customDetectors": map[string]any{
				"enabled": false,
				"detectors": []map[string]any{
					{
						"name":    "detector",
						"enabled": true,
						"action":  "redact",
						"matcher": map[string]any{
							"kind":  "regex",
							"regex": map[string]any{"pattern": "ACCT-[9]"},
						},
					},
				},
			},
		},
		"customMemoryTypes": []map[string]any{
			{
				"name":        "custom_name",
				"description": "test",
				"fields": []map[string]any{
					{"name": "field1", "description": "capture the field", "type": "str"},
				},
				"extractionStrategy": map[string]any{
					"enabled": enabled,
					"prompt":  "test",
				},
			},
		},
		"endpoint": "https://aws-us-east-1.memory.redis.io",
		"endpoints": []map[string]any{
			{
				"url":          "https://aws-us-east-1.memory.redis.io",
				"provider":     "AWS",
				"region":       "us-east-1",
				"egressIps":    []string{"10.0.0.1"},
				"isAccessible": true,
			},
		},
		"status":    "READY",
		"createdAt": "2026-09-22T12:00:00Z",
	}))
}

func advancedAgentMemoryConfig(name string) string {
	return advancedAgentMemoryConfigWithValues(name, 21, 11, "test", "redact")
}

func advancedAgentMemoryConfigWithValues(name string, threshold, retainCount int, semanticPrompt, emailAction string) string {
	return advancedAgentMemoryConfigWithCustomMemoryStrategy(name, threshold, retainCount, semanticPrompt, emailAction, "test", true)
}

func advancedAgentMemoryConfigWithCustomMemoryStrategy(name string, threshold, retainCount int, semanticPrompt, emailAction, customMemoryPrompt string, customMemoryEnabled bool) string {
	return `
resource "rediscloud_agent_memory" "example" {
  name                       = "` + name + `"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 604800
  extraction_cadence_seconds = 300

  summarization {
    enabled          = true
    trigger_strategy = "event_count"

    event_count {
      threshold    = ` + fmt.Sprint(threshold) + `
      retain_count = ` + fmt.Sprint(retainCount) + `
    }
  }

  custom_memory_types {
    name        = "custom_name"
    description = "test"

    fields {
      name        = "field1"
      type        = "str"
      description = "capture the field"
    }

    extraction_strategy {
      enabled = ` + fmt.Sprint(customMemoryEnabled) + `
      prompt  = "` + customMemoryPrompt + `"
    }
  }

  long_term_memory_exclusions {
    enabled = true

    semantic {
      enabled = true
      prompt  = "` + semanticPrompt + `"
    }

    built_in_detectors {
      enabled = true

      detectors {
        id      = "credit-card"
        enabled = true
        action  = "redact"
      }

      detectors {
        id      = "email"
        enabled = true
        action  = "` + emailAction + `"
      }
    }

    custom_detectors {
      enabled = true

      detectors {
        name    = "detector"
        enabled = true
        action  = "redact"

        matcher {
          kind = "regex"

          regex {
            pattern = "ACCT-[9]"
          }
        }
      }
    }
  }
}
`
}

func disabledAdvancedAgentMemoryConfig(name string) string {
	return `
resource "rediscloud_agent_memory" "example" {
  name                       = "` + name + `"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 604800
  extraction_cadence_seconds = 300

  summarization {
    enabled = false
  }

  custom_memory_types {
    name        = "custom_name"
    description = "test"

    fields {
      name        = "field1"
      type        = "str"
      description = "capture the field"
    }

    extraction_strategy {
      enabled = true
      prompt  = "test"
    }
  }

  long_term_memory_exclusions {
    enabled = true

    semantic {
      enabled = false
    }

    built_in_detectors {
      enabled = false
    }

    custom_detectors {
      enabled = false
    }
  }
}
`
}

func advancedAgentMemoryConfigWithAdditionalCustomMemoryType(name string) string {
	return `
resource "rediscloud_agent_memory" "example" {
  name                       = "` + name + `"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 604800
  extraction_cadence_seconds = 300

  summarization {
    enabled          = true
    trigger_strategy = "event_count"

    event_count {
      threshold    = 21
      retain_count = 11
    }
  }

  custom_memory_types {
    name        = "custom_name"
    description = "test"

    fields {
      name        = "field1"
      type        = "str"
      description = "capture the field"
    }

    extraction_strategy {
      enabled = true
      prompt  = "test"
    }
  }

  custom_memory_types {
    name        = "support_case"
    description = "Persistent support case details"

    fields {
      name        = "case_priority"
      type        = "int"
      description = "case priority"
    }

    extraction_strategy {
      enabled = true
      prompt  = "Extract durable support case details."
    }
  }

  long_term_memory_exclusions {
    enabled = true

    semantic {
      enabled = true
      prompt  = "test"
    }

    built_in_detectors {
      enabled = true

      detectors {
        id      = "credit-card"
        enabled = true
        action  = "redact"
      }

      detectors {
        id      = "email"
        enabled = true
        action  = "redact"
      }
    }

    custom_detectors {
      enabled = true

      detectors {
        name    = "detector"
        enabled = true
        action  = "redact"

        matcher {
          kind = "regex"

          regex {
            pattern = "ACCT-[9]"
          }
        }
      }
    }
  }
}
`
}

func advancedAgentMemoryConfigWithChangedCustomMemoryTypeField(name string) string {
	return `
resource "rediscloud_agent_memory" "example" {
  name                       = "` + name + `"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 604800
  extraction_cadence_seconds = 300

  custom_memory_types {
    name        = "custom_name"
    description = "test"

    fields {
      name        = "field1"
      type        = "int"
      description = "capture the field"
    }

    extraction_strategy {
      enabled = true
      prompt  = "test"
    }
  }
}
`
}

func agentMemoryConfigWithDuplicateCustomMemoryTypeNames() string {
	return `
resource "rediscloud_agent_memory" "example" {
  name                       = "store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 604800
  extraction_cadence_seconds = 300

  custom_memory_types {
    name        = "duplicate_name"
    description = "first type"

    fields {
      name        = "first_field"
      type        = "str"
      description = "first field"
    }

    extraction_strategy {
      enabled = true
      prompt  = "Extract the first type."
    }
  }

  custom_memory_types {
    name        = "duplicate_name"
    description = "second type"

    fields {
      name        = "second_field"
      type        = "str"
      description = "second field"
    }

    extraction_strategy {
      enabled = true
      prompt  = "Extract the second type."
    }
  }
}
`
}

func advancedAgentMemoryConfigWithoutCustomMemoryTypes(name string) string {
	return `
resource "rediscloud_agent_memory" "example" {
  name                       = "` + name + `"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 604800
  extraction_cadence_seconds = 300
}
`
}
