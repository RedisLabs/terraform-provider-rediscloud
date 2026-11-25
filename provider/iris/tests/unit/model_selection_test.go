package unit_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/require"
)

func TestIrisStoreModelSelectionCredentials(t *testing.T) {
	for _, authenticatedLLM := range []bool{false, true} {
		name := "no_keys"
		if authenticatedLLM {
			name = "mixed_providers"
		}
		t.Run(name, func(t *testing.T) {
			f, server := newFixture(t)
			variables := config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy")}
			if authenticatedLLM {
				variables["model_providers"] = config.MapVariable(map[string]config.Variable{"llm": config.StringVariable("openai"), "embedding": config.StringVariable("redis_local"), "classification": config.StringVariable("redis_local")})
				variables["models"] = config.MapVariable(map[string]config.Variable{"llm": config.StringVariable("gpt-4o-mini"), "embedding": config.StringVariable("redis-embed-v1"), "classification": config.StringVariable("redis-llm-v1")})
				variables["keys"] = config.MapVariable(map[string]config.Variable{"llm": config.StringVariable("fake-llm-key")})
				variables["versions"] = config.MapVariable(map[string]config.Variable{"llm": config.IntegerVariable(1)})
			}
			resource.UnitTest(t, resource.TestCase{Steps: []resource.TestStep{
				{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/models.tf"), ConfigVariables: variables, Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("rediscloud_iris_agent_memory.store", "embedding_provider", "redis_local"),
					resource.TestCheckResourceAttr("rediscloud_iris_agent_memory.store", "classification_model", "redis-llm-v1"),
					func(s *terraform.State) error {
						f.mu.Lock()
						defer f.mu.Unlock()
						models := f.objects["/iris/v1/agent-memory/stores/"+storeID]["longTermMemory"].(map[string]any)
						for _, purpose := range []string{"llm", "embedding", "classification"} {
							model := models[purpose].(map[string]any)
							if purpose == "llm" && authenticatedLLM {
								require.Equal(t, map[string]any{"type": "API_KEY", "apiKey": "fake-llm-key"}, model["credentials"])
							} else {
								require.NotContains(t, model, "credentials")
								require.NoError(t, resource.TestCheckNoResourceAttr("rediscloud_iris_agent_memory.store", purpose+"_api_key_version")(s))
							}
							require.NoError(t, resource.TestCheckNoResourceAttr("rediscloud_iris_agent_memory.store", purpose+"_api_key")(s))
						}
						return nil
					},
				)},
				{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/models.tf"), ConfigVariables: variables, PlanOnly: true},
			}})
		})
	}
}

func TestIrisStoreModelOnlyUpdatePreservesCredentials(t *testing.T) {
	f, server := newFixture(t)
	initial := config.Variables{
		"memory_url":      config.StringVariable(server.URL + "/memory-proxy"),
		"model_providers": config.MapVariable(map[string]config.Variable{"llm": config.StringVariable("openai"), "embedding": config.StringVariable("openai"), "classification": config.StringVariable("openai")}),
		"models":          config.MapVariable(map[string]config.Variable{"llm": config.StringVariable("gpt-4o-mini"), "embedding": config.StringVariable("text-embedding-3-small"), "classification": config.StringVariable("gpt-4o-mini")}),
		"keys":            config.MapVariable(map[string]config.Variable{"llm": config.StringVariable("fake-llm-key"), "embedding": config.StringVariable("fake-embedding-key"), "classification": config.StringVariable("fake-classification-key")}),
		"versions":        config.MapVariable(map[string]config.Variable{"llm": config.IntegerVariable(1), "embedding": config.IntegerVariable(1), "classification": config.IntegerVariable(1)}),
	}
	partial := config.Variables{
		"memory_url":      config.StringVariable(server.URL + "/memory-proxy"),
		"model_providers": config.MapVariable(map[string]config.Variable{}),
		"models":          config.MapVariable(map[string]config.Variable{"llm": config.StringVariable("gpt-4.1-mini"), "classification": config.StringVariable("gpt-4.1-mini")}),
	}
	resource.UnitTest(t, resource.TestCase{Steps: []resource.TestStep{
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/models.tf"), ConfigVariables: initial},
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/models.tf"), ConfigVariables: partial, Check: resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("rediscloud_iris_agent_memory.store", "llm_model", "gpt-4.1-mini"),
			resource.TestCheckResourceAttr("rediscloud_iris_agent_memory.store", "classification_model", "gpt-4.1-mini"),
			resource.TestCheckResourceAttr("rediscloud_iris_agent_memory.store", "embedding_model", "text-embedding-3-small"),
			func(s *terraform.State) error {
				f.mu.Lock()
				defer f.mu.Unlock()
				models := f.objects["/iris/v1/agent-memory/stores/"+storeID]["longTermMemory"].(map[string]any)
				for _, purpose := range []string{"llm", "embedding", "classification"} {
					model := models[purpose].(map[string]any)
					require.Equal(t, "openai", model["provider"])
					require.Equal(t, map[string]any{"type": "API_KEY", "apiKey": "fake-" + purpose + "-key"}, model["credentials"])
					require.NoError(t, resource.TestCheckNoResourceAttr("rediscloud_iris_agent_memory.store", purpose+"_api_key")(s))
				}
				return nil
			},
		)},
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/models.tf"), ConfigVariables: partial, PlanOnly: true},
	}})
	f.mu.Lock()
	defer f.mu.Unlock()
	require.Equal(t, 1, f.creates)
	require.Equal(t, 1, f.updates)
	require.Equal(t, 1, f.deletes)
}

func TestIrisStoreModelKeysRequireMatchingVersions(t *testing.T) {
	for _, purpose := range []string{"llm", "embedding", "classification"} {
		for _, missing := range []string{"keys", "versions"} {
			t.Run(purpose+"/missing_"+missing, func(t *testing.T) {
				f, server := newFixture(t)
				variables := config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy")}
				if missing == "keys" {
					variables["versions"] = config.MapVariable(map[string]config.Variable{purpose: config.IntegerVariable(1)})
				} else {
					variables["keys"] = config.MapVariable(map[string]config.Variable{purpose: config.StringVariable("fake-key")})
				}
				resource.UnitTest(t, resource.TestCase{Steps: []resource.TestStep{
					{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/models.tf"), ConfigVariables: variables, PlanOnly: true, ExpectError: regexp.MustCompile("Invalid Attribute Combination")},
				}})
				f.mu.Lock()
				defer f.mu.Unlock()
				require.Zero(t, f.creates)
			})
		}
	}
}

func TestIrisStoreModelSelectionsRequireCompleteMetadata(t *testing.T) {
	for _, purpose := range []string{"llm", "embedding", "classification"} {
		for _, missing := range []string{"provider", "model"} {
			t.Run(purpose+"/missing_"+missing, func(t *testing.T) {
				f, server := newFixture(t)
				providers, models := map[string]config.Variable{}, map[string]config.Variable{}
				for _, name := range []string{"llm", "embedding", "classification"} {
					if name != purpose || missing != "provider" {
						providers[name] = config.StringVariable("redis_local")
					}
					if name != purpose || missing != "model" {
						models[name] = config.StringVariable("redis-model")
					}
				}
				resource.UnitTest(t, resource.TestCase{Steps: []resource.TestStep{
					{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/models.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy"), "model_providers": config.MapVariable(providers), "models": config.MapVariable(models)}, PlanOnly: true, ExpectError: regexp.MustCompile("Configure " + purpose + "_" + missing)},
				}})
				f.mu.Lock()
				defer f.mu.Unlock()
				require.Zero(t, f.creates)
			})
		}
	}
	for _, purpose := range []string{"llm", "embedding"} {
		t.Run("only_"+purpose, func(t *testing.T) {
			f, server := newFixture(t)
			resource.UnitTest(t, resource.TestCase{Steps: []resource.TestStep{
				{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/models.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy"), "model_providers": config.MapVariable(map[string]config.Variable{purpose: config.StringVariable("redis_local")}), "models": config.MapVariable(map[string]config.Variable{purpose: config.StringVariable("redis-model")})}, PlanOnly: true, ExpectError: regexp.MustCompile("Missing store setting")},
			}})
			f.mu.Lock()
			defer f.mu.Unlock()
			require.Zero(t, f.creates)
		})
	}
}
