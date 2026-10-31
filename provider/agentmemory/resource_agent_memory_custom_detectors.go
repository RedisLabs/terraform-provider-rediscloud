package agentmemory

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func validateCustomDetectorNamesPlan(ctx context.Context, plan AgentMemoryResourceModel, resp *resource.ModifyPlanResponse) {
	if plan.LongTermMemoryExclusions.IsUnknown() {
		return
	}
	exclusions := exclusionsBlock(ctx, plan.LongTermMemoryExclusions, &resp.Diagnostics)
	if exclusions == nil || exclusions.CustomDetectors.IsUnknown() {
		return
	}
	custom := customDetectorsBlock(ctx, exclusions.CustomDetectors, &resp.Diagnostics)
	if custom == nil {
		return
	}
	list := custom.Detectors
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
				path.Root("long_term_memory_exclusions").AtListIndex(0).AtName("custom_detectors").AtListIndex(0).AtName("detectors").AtListIndex(i).AtName("name"),
				"Duplicate Agent Memory custom detector name",
				fmt.Sprintf("Custom detector names must be unique within the store. %q is already configured at long_term_memory_exclusions[0].custom_detectors[0].detectors[%d].", name, previous),
			)
			continue
		}
		names[name] = i
	}
}

func validateCustomDetectorMatchersConfig(ctx context.Context, config AgentMemoryResourceModel, resp *resource.ModifyPlanResponse) {
	if config.LongTermMemoryExclusions.IsUnknown() {
		return
	}
	exclusions := exclusionsBlock(ctx, config.LongTermMemoryExclusions, &resp.Diagnostics)
	if exclusions == nil || exclusions.CustomDetectors.IsUnknown() {
		return
	}
	custom := customDetectorsBlock(ctx, exclusions.CustomDetectors, &resp.Diagnostics)
	if custom == nil {
		return
	}
	list := custom.Detectors
	if list.IsUnknown() {
		return
	}
	for i, detector := range customDetectorModels(ctx, list, &resp.Diagnostics) {
		// Presence is checked by the schema.
		if detector.Matcher.IsUnknown() {
			continue
		}
		matcher := matcherSpecBlock(ctx, detector.Matcher, &resp.Diagnostics)
		if matcher == nil {
			continue
		}
		// Defer unknown kinds: only regex matchers require this block.
		kind, known := knownString(matcher.Kind)
		if known && kind == "regex" && !matcher.Regex.IsUnknown() && len(matcher.Regex.Elements()) == 0 {
			resp.Diagnostics.AddAttributeError(
				path.Root("long_term_memory_exclusions").AtListIndex(0).AtName("custom_detectors").AtListIndex(0).AtName("detectors").AtListIndex(i).AtName("matcher").AtListIndex(0).AtName("regex"),
				"Missing custom detector regex",
				"custom detector matcher.regex must be configured when matcher.kind is regex.",
			)
		}
	}
}
