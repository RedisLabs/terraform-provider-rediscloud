package unit_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestIrisSetElementUpdates(t *testing.T) {
	for _, kind := range []string{"key", "store"} {
		t.Run(kind, func(t *testing.T) {
			_, server := newFixture(t)
			resource.UnitTest(t, resource.TestCase{Steps: []resource.TestStep{
				{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/set_updates.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy"), "identity_url": config.StringVariable(server.URL + "/identity-proxy"), "kind": config.StringVariable(kind), "items": config.SetVariable(config.StringVariable("first"))}},
				{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/set_updates.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy"), "identity_url": config.StringVariable(server.URL + "/identity-proxy"), "kind": config.StringVariable(kind), "items": config.SetVariable(config.StringVariable("first"), config.StringVariable("second"))}},
				{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/set_updates.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy"), "identity_url": config.StringVariable(server.URL + "/identity-proxy"), "kind": config.StringVariable(kind), "items": config.SetVariable(config.StringVariable("first"), config.StringVariable("second"))}, PlanOnly: true},
				{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/set_updates.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy"), "identity_url": config.StringVariable(server.URL + "/identity-proxy"), "kind": config.StringVariable(kind), "items": config.SetVariable(config.StringVariable("second"))}},
			}})
		})
	}
}

func TestIrisDetectorActionCanBeRemoved(t *testing.T) {
	f, server := newFixture(t)
	check := func(action any) resource.TestCheckFunc {
		return func(_ *terraform.State) error {
			f.mu.Lock()
			defer f.mu.Unlock()
			groups := f.objects["/iris/v1/agent-memory/stores/"+storeID]["longTermMemoryExclusions"].(map[string]any)
			for _, name := range []string{"builtInDetectors", "customDetectors"} {
				detector := groups[name].(map[string]any)["detectors"].([]any)[0].(map[string]any)
				require.Equal(t, action, detector["action"])
			}
			return nil
		}
	}
	resource.UnitTest(t, resource.TestCase{Steps: []resource.TestStep{
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/set_updates.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy"), "identity_url": config.StringVariable(server.URL + "/identity-proxy"), "kind": config.StringVariable("store"), "items": config.SetVariable(config.StringVariable("first"))}, Check: check("DROP")},
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/set_updates.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy"), "identity_url": config.StringVariable(server.URL + "/identity-proxy"), "kind": config.StringVariable("store"), "items": config.SetVariable(config.StringVariable("first")), "detector_action": config.StringVariable("")}, Check: check(nil)},
	}})
}
