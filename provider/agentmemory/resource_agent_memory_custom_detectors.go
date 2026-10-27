package agentmemory

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func validateCustomDetectorNamesPlan(ctx context.Context, plan AgentMemoryResourceModel, resp *resource.ModifyPlanResponse) {
	if plan.LongTermMemoryExclusions == nil || plan.LongTermMemoryExclusions.CustomDetectors == nil {
		return
	}
	list := plan.LongTermMemoryExclusions.CustomDetectors.Detectors
	if list.IsUnknown() {
		return
	}
	detectors := customDetectorModels(ctx, list, &resp.Diagnostics)
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

func validateCustomDetectorMatchersConfig(ctx context.Context, config AgentMemoryResourceModel, resp *resource.ModifyPlanResponse) {
	if config.LongTermMemoryExclusions == nil || config.LongTermMemoryExclusions.CustomDetectors == nil {
		return
	}
	list := config.LongTermMemoryExclusions.CustomDetectors.Detectors
	if list.IsUnknown() {
		return
	}
	for i, detector := range customDetectorModels(ctx, list, &resp.Diagnostics) {
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
