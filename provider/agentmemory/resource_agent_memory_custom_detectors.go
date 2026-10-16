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
