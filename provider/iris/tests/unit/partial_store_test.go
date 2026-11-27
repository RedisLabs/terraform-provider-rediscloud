package unit_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/require"
)

func TestIrisStoreCreationRejectsImplicitlyDisabledGroups(t *testing.T) {
	f, server := newFixture(t)
	resource.UnitTest(t, resource.TestCase{Steps: []resource.TestStep{
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/partial_store.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy"), "partial": config.BoolVariable(true)}, PlanOnly: true, ExpectError: regexp.MustCompile("Missing store setting")},
	}})
	f.mu.Lock()
	defer f.mu.Unlock()
	require.Zero(t, f.creates)
}

func TestIrisStoreReplacementRequiresCreationSettings(t *testing.T) {
	f, server := newFixture(t)
	resource.UnitTest(t, resource.TestCase{Steps: []resource.TestStep{
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/partial_store.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy")}},
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/partial_store.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy"), "partial": config.BoolVariable(true), "database_id": config.StringVariable("db-2")}, PlanOnly: true, ExpectError: regexp.MustCompile("Missing store setting")},
	}})
	f.mu.Lock()
	defer f.mu.Unlock()
	require.Equal(t, 1, f.creates)
	require.Equal(t, 1, f.deletes, "only final test cleanup may destroy the original store")
}

func TestIrisStorePartialSettingsPreserveOmittedValues(t *testing.T) {
	for _, tc := range []struct {
		edit, attribute, value string
	}{
		{"summary_threshold", "summarization_event_threshold", "200"},
		{"summary_retain", "summarization_retain_count", "0"},
		{"summary_trigger", "summarization_trigger_strategy", "EVENT_COUNT"},
		{"semantic_enabled", "semantic_exclusions_enabled", "false"},
		{"semantic_prompt", "semantic_exclusions_prompt", "new prompt"},
		{"built_in_enabled", "built_in_detectors_enabled", "false"},
		{"custom_enabled", "custom_detectors_enabled", "false"},
		{"detector_action", "built_in_detector.0.action", "REDACT"},
	} {
		t.Run(tc.edit, func(t *testing.T) {
			f, server := newFixture(t)
			const address = "rediscloud_iris_agent_memory.store"
			expected := map[string]string{
				"exclusions_enabled": "true", "summarization_enabled": "true", "summarization_trigger_strategy": "EVENT_COUNT",
				"summarization_event_threshold": "100", "summarization_retain_count": "10",
				"semantic_exclusions_enabled": "true", "semantic_exclusions_prompt": "private data",
				"built_in_detectors_enabled": "true", "custom_detectors_enabled": "true",
			}
			var checks []resource.TestCheckFunc
			if tc.edit == "detector_action" {
				checks = append(checks, resource.TestCheckTypeSetElemNestedAttrs(address, "built_in_detector.*", map[string]string{"id": "email", "action": tc.value}))
			} else {
				expected[tc.attribute] = tc.value
			}
			for name, value := range expected {
				checks = append(checks, resource.TestCheckResourceAttr(address, name, value))
			}
			resource.UnitTest(t, resource.TestCase{Steps: []resource.TestStep{
				{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/partial_store.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy")}},
				{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/partial_store.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy"), "partial": config.BoolVariable(true), "edit": config.StringVariable(tc.edit)}, Check: resource.ComposeTestCheckFunc(checks...)},
				{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/partial_store.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy"), "partial": config.BoolVariable(true), "edit": config.StringVariable(tc.edit)}, PlanOnly: true},
			}})
			f.mu.Lock()
			defer f.mu.Unlock()
			require.Equal(t, 1, f.creates)
			require.Equal(t, 1, f.deletes)
			updates := 1
			if tc.edit == "summary_trigger" {
				updates = 0 // Taking control of the unchanged trigger needs no API write.
			}
			require.Equal(t, updates, f.updates)
		})
	}
}
