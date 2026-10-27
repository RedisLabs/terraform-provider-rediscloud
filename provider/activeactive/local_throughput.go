package activeactive

import (
	"context"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// defaultLocalOperationsPerSecond is the Cloud API default for local read and
// write ops on an Active-Active region when the caller does not set them.
const defaultLocalOperationsPerSecond = 1000

// configuredLocalThroughput reports the local read and write ops configured on
// an override region. ok is false when either attribute is unset, which keeps
// the 1000/1000 default and avoids sending a throughput change on update.
func configuredLocalThroughput(region OverrideRegionModel) (read int, write int, ok bool) {
	if region.LocalReadOperationsPerSecond.IsNull() || region.LocalReadOperationsPerSecond.IsUnknown() {
		return 0, 0, false
	}
	if region.LocalWriteOperationsPerSecond.IsNull() || region.LocalWriteOperationsPerSecond.IsUnknown() {
		return 0, 0, false
	}
	return int(region.LocalReadOperationsPerSecond.ValueInt64()), int(region.LocalWriteOperationsPerSecond.ValueInt64()), true
}

func localThroughputMeasurement(regionName string, read, write int) *databases.LocalThroughput {
	return &databases.LocalThroughput{
		Region:                   redis.String(regionName),
		ReadOperationsPerSecond:  redis.Int(read),
		WriteOperationsPerSecond: redis.Int(write),
	}
}

func overrideRegionsByName(ctx context.Context, plan *ActiveActiveDatabaseModel) (map[string]OverrideRegionModel, diag.Diagnostics) {
	result := map[string]OverrideRegionModel{}
	if plan.OverrideRegion.IsNull() || plan.OverrideRegion.IsUnknown() {
		return result, nil
	}

	var regions []OverrideRegionModel
	diags := plan.OverrideRegion.ElementsAs(ctx, &regions, false)
	if diags.HasError() {
		return nil, diags
	}
	for _, region := range regions {
		result[region.Name.ValueString()] = region
	}
	return result, diags
}

// buildCreateLocalThroughputs sets local read and write ops for every
// subscription region. A region whose override configures both values uses
// those values. Every other region stays at the API default of 1000/1000.
func buildCreateLocalThroughputs(regionNames []string, overrides map[string]OverrideRegionModel) []*databases.LocalThroughput {
	throughputs := make([]*databases.LocalThroughput, 0, len(regionNames))
	for _, name := range regionNames {
		read, write := defaultLocalOperationsPerSecond, defaultLocalOperationsPerSecond
		if override, ok := overrides[name]; ok {
			if configuredRead, configuredWrite, configured := configuredLocalThroughput(override); configured {
				read, write = configuredRead, configuredWrite
			}
		}
		throughputs = append(throughputs, localThroughputMeasurement(name, read, write))
	}
	return throughputs
}

// setLocalThroughput adds localThroughputMeasurement to a region update only
// when the user set both ops attributes. An omitted value must not be sent,
// or this resource would overwrite throughput managed elsewhere.
func setLocalThroughput(props *databases.LocalRegionProperties, region OverrideRegionModel) {
	read, write, ok := configuredLocalThroughput(region)
	if !ok {
		return
	}
	props.LocalThroughputMeasurement = localThroughputMeasurement(region.Name.ValueString(), read, write)
}

// localThroughputStateValues returns the ops values to store for an override
// region. When the user did not set the attributes, both stay null so an
// upgrade does not plan a diff. When they are set, the values come from the API.
func localThroughputStateValues(configured OverrideRegionModel, readFromAPI, writeFromAPI *int) (types.Int64, types.Int64) {
	if configured.LocalReadOperationsPerSecond.IsNull() || configured.LocalWriteOperationsPerSecond.IsNull() {
		return types.Int64Null(), types.Int64Null()
	}

	read := types.Int64Null()
	write := types.Int64Null()
	if readFromAPI != nil {
		read = types.Int64Value(int64(*readFromAPI))
	}
	if writeFromAPI != nil {
		write = types.Int64Value(int64(*writeFromAPI))
	}
	return read, write
}
