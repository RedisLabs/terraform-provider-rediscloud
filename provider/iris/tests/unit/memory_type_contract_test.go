package unit_test

import (
	"context"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

func seedMemoryTypeParent(f *fixture) {
	f.objects["/iris/v1/agent-memory/stores/"+storeID] = map[string]any{"storeId": storeID, "name": "existing", "state": "READY"}
}

func TestIrisMemoryTypeReadFailureAfterCreateDoesNotRetireName(t *testing.T) {
	f, server := newFixture(t)
	seedMemoryTypeParent(f)
	f.typeReadFailures = 1
	resource.UnitTest(t, resource.TestCase{Steps: []resource.TestStep{
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/memory_type.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy")}, Check: resource.TestCheckResourceAttr("rediscloud_iris_agent_memory_type.type", "revision", "1")},
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/memory_type.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy")}, PlanOnly: true},
	}})
	f.mu.Lock()
	defer f.mu.Unlock()
	require.Zero(t, f.typeReadFailures, "exercise the failed follow-up GET")
	require.Equal(t, 1, f.creates)
	require.Equal(t, 1, f.deletes, "only final cleanup may retire the name")
}
func TestIrisMemoryTypeRejectsUnsafeReplacement(t *testing.T) {
	for _, field := range []string{"description", "field_type", "name"} {
		t.Run(field, func(t *testing.T) {
			f, server := newFixture(t)
			seedMemoryTypeParent(f)
			changed := config.StringVariable("changed")
			if field == "field_type" {
				changed = config.StringVariable("BOOLEAN")
			}
			if field == "name" {
				changed = config.StringVariable("PROFILE")
			}
			resource.UnitTest(t, resource.TestCase{Steps: []resource.TestStep{
				{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/memory_type.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy")}},
				{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/memory_type.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy"), field: changed}, PlanOnly: true, ExpectError: regexp.MustCompile("Immutable memory type")},
			}})
			f.mu.Lock()
			defer f.mu.Unlock()
			require.Equal(t, 1, f.creates)
			require.Equal(t, 1, f.deletes)
			require.Equal(t, 1, f.updates)
		})
	}
}
func TestIrisFixtureRejectsRetiredTypeNames(t *testing.T) {
	f, server := newFixture(t)
	seedMemoryTypeParent(f)
	api, err := client.NewIrisServiceClient(client.IrisServiceConfig{URL: server.URL + "/memory-proxy", Token: "memory-token"})
	require.NoError(t, err)
	path := "/iris/v1/agent-memory/stores/" + storeID + "/memory-types"
	var created struct {
		Revision string `json:"revision"`
	}
	body := map[string]any{"name": "profile", "description": "Profile"}
	require.NoError(t, api.Do(context.Background(), http.MethodPost, path, nil, body, &created))
	require.NoError(t, api.Do(context.Background(), http.MethodDelete, path+"/profile", map[string][]string{"expectedRevision": {created.Revision}}, nil, nil))
	body["name"] = "PROFILE"
	err = api.Do(context.Background(), http.MethodPost, path, nil, body, nil)
	var failure *client.IrisHTTPError
	require.ErrorAs(t, err, &failure)
	require.Equal(t, 409, failure.StatusCode)
}

func TestIrisMemoryTypeAllowsDistinctReplacementName(t *testing.T) {
	f, server := newFixture(t)
	seedMemoryTypeParent(f)
	resource.UnitTest(t, resource.TestCase{Steps: []resource.TestStep{
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/memory_type.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy")}},
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/memory_type.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy"), "name": config.StringVariable("new_profile"), "description": config.StringVariable("New schema"), "field_type": config.StringVariable("BOOLEAN")}, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("rediscloud_iris_agent_memory_type.type", plancheck.ResourceActionDestroyBeforeCreate)}}, Check: resource.TestCheckResourceAttr("rediscloud_iris_agent_memory_type.type", "id", storeID+"/new_profile")},
	}})
	f.mu.Lock()
	defer f.mu.Unlock()
	require.Equal(t, 2, f.creates)
	require.Equal(t, 2, f.deletes)
}

func TestIrisMemoryTypeRejectsAddingExtractionStrategy(t *testing.T) {
	f, server := newFixture(t)
	seedMemoryTypeParent(f)
	resource.UnitTest(t, resource.TestCase{Steps: []resource.TestStep{
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/memory_type.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy"), "with_strategy": config.BoolVariable(false)}},
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/memory_type.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy")}, PlanOnly: true, ExpectError: regexp.MustCompile("Missing extraction strategy")},
	}})
	f.mu.Lock()
	defer f.mu.Unlock()
	require.Equal(t, 1, f.creates)
	require.Equal(t, 1, f.deletes)
	require.Zero(t, f.updates)
}

func TestIrisMemoryTypeRequiresPromptWhenExtractionIsConfigured(t *testing.T) {
	_, server := newFixture(t)
	resource.UnitTest(t, resource.TestCase{Steps: []resource.TestStep{{
		ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/memory_type.tf"),
		ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy"), "with_strategy": config.BoolVariable(false), "enabled_without_prompt": config.BoolVariable(true)},
		PlanOnly:        true, ExpectError: regexp.MustCompile("extraction_prompt"),
	}}})
}

func TestIrisExistingExtractionCanBeDisabledWithoutPrompt(t *testing.T) {
	f, server := newFixture(t)
	seedMemoryTypeParent(f)
	const address = "rediscloud_iris_agent_memory_type.type"
	resource.UnitTest(t, resource.TestCase{Steps: []resource.TestStep{
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/memory_type.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy")}},
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/memory_type.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy"), "with_strategy": config.BoolVariable(false), "enabled_without_prompt": config.BoolVariable(true)}, Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr(address, "extraction_enabled", "false"), resource.TestCheckResourceAttr(address, "extraction_prompt", "Extract profiles"))},
	}})
	f.mu.Lock()
	defer f.mu.Unlock()
	require.Equal(t, 1, f.updates)
	require.Equal(t, 1, f.deletes)
}

func TestIrisSiblingUpdatesUseIndependentTypeRevisions(t *testing.T) {
	f, server := newFixture(t)
	seedMemoryTypeParent(f)
	resource.UnitTest(t, resource.TestCase{Steps: []resource.TestStep{
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/sibling_types.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy")}, Check: resource.TestCheckResourceAttr("rediscloud_iris_agent_memory_type.type.0", "revision", "1")},
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/sibling_types.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy"), "prompt": config.StringVariable("Updated")}, Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("rediscloud_iris_agent_memory_type.type.0", "revision", "2"), resource.TestCheckResourceAttr("rediscloud_iris_agent_memory_type.type.1", "revision", "2"), resource.TestCheckResourceAttr("rediscloud_iris_agent_memory_type.type.2", "revision", "2"))},
	}})
	f.mu.Lock()
	defer f.mu.Unlock()
	require.Equal(t, 3, f.creates)
	require.Equal(t, 3, f.deletes)
}
