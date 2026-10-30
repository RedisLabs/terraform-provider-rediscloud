package agentmemory

import (
	"context"
	"reflect"
	"slices"
	"strings"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	agentmemoryapi "github.com/RedisLabs/rediscloud-go-api/service/agentmemory"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func hasAgentMemoryUpdates(update agentmemoryapi.UpdateStore) bool {
	return update.Name != nil || update.ShortMemory != nil || update.LongTermMemory != nil ||
		update.LLM != nil || update.Summarization != nil || update.LongTermMemoryExclusions != nil ||
		update.ExtractionCadence != nil || len(update.AddCustomMemoryTypes) > 0 ||
		len(update.UpdateCustomMemoryTypeStrategies) > 0
}

func exclusionsConfigEqual(ctx context.Context, plan, state ExclusionsListValue, diags *diag.Diagnostics) bool {
	planned := exclusionsFromModel(ctx, exclusionsBlock(ctx, plan, diags), diags)
	existing := exclusionsFromModel(ctx, exclusionsBlock(ctx, state, diags), diags)
	// Detector order belongs to Terraform state, not to the remote configuration.
	for _, config := range []*agentmemoryapi.LongTermMemoryExclusions{planned, existing} {
		if config == nil {
			continue
		}
		if config.BuiltInDetectors != nil {
			slices.SortStableFunc(config.BuiltInDetectors.Detectors, func(a, b agentmemoryapi.DetectorSelection) int {
				return strings.Compare(redis.StringValue(a.ID), redis.StringValue(b.ID))
			})
		}
		if config.CustomDetectors != nil {
			slices.SortStableFunc(config.CustomDetectors.Detectors, func(a, b agentmemoryapi.CustomDetector) int {
				return strings.Compare(redis.StringValue(a.Name), redis.StringValue(b.Name))
			})
		}
	}
	return reflect.DeepEqual(planned, existing)
}
