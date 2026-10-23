package agentmemory

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func validateCustomDetectorNamesPlan(plan AgentMemoryResourceModel, resp *resource.ModifyPlanResponse) {
	if plan.LongTermMemoryExclusions == nil || plan.LongTermMemoryExclusions.CustomDetectors == nil {
		return
	}
	detectors := plan.LongTermMemoryExclusions.CustomDetectors.Detectors
	names := make(map[string]int, len(detectors))
	// Disabled detectors remain stored, so their names must also be unique.
	for i, detector := range detectors {
		name, known := knownString(detector.Name)
		if !known {
			continue
		}
		if previous, exists := names[name]; exists {
			resp.Diagnostics.AddAttributeError(
				path.Root("long_term_memory_exclusions").AtName("custom_detectors").AtName("detectors").AtListIndex(i).AtName("name"),
				"Duplicate Agent Memory custom detector name",
				fmt.Sprintf("Custom detector names must be unique within the store. %q is already configured at long_term_memory_exclusions.custom_detectors.detectors.%d.", name, previous),
			)
			continue
		}
		names[name] = i
	}
}

func validateCustomDetectorMatchersConfig(config AgentMemoryResourceModel, resp *resource.ModifyPlanResponse) {
	if config.LongTermMemoryExclusions == nil || config.LongTermMemoryExclusions.CustomDetectors == nil {
		return
	}
	for i, detector := range config.LongTermMemoryExclusions.CustomDetectors.Detectors {
		// Presence is checked by the schema.
		if detector.Matcher == nil {
			continue
		}
		// Defer unknown kinds: only regex matchers require this block.
		kind, known := knownString(detector.Matcher.Kind)
		if known && kind == "regex" && detector.Matcher.Regex == nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("long_term_memory_exclusions").AtName("custom_detectors").AtName("detectors").AtListIndex(i).AtName("matcher").AtName("regex"),
				"Missing custom detector regex",
				"custom detector matcher.regex must be configured when matcher.kind is regex.",
			)
		}
	}
}
