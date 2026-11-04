package activeactive

import (
	"context"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestUnitActiveActiveDatabase_UnrelatedRegionUpdateOmitsThroughputWhenUnset(t *testing.T) {
	plan := &ActiveActiveDatabaseModel{
		GlobalSourceIPs: types.SetNull(types.StringType),
		GlobalAlert:     types.SetNull(types.ObjectType{AttrTypes: getAlertAttrTypes()}),
		OverrideRegion: overrideRegionSet(t, OverrideRegionModel{
			Name:                             types.StringValue("us-east-1"),
			OverrideGlobalPassword:           types.StringValue("secret-password"),
			OverrideGlobalAlert:              types.SetNull(types.ObjectType{AttrTypes: getAlertAttrTypes()}),
			OverrideGlobalSourceIPs:          types.SetNull(types.StringType),
			RemoteBackup:                     types.ListNull(types.ObjectType{AttrTypes: getRemoteBackupAttrTypes()}),
			LocalReadOperationsPerSecond:     types.Int64Null(),
			LocalWriteOperationsPerSecond:    types.Int64Null(),
			OverrideGlobalEnablePasswordless: types.BoolNull(),
			OverrideGlobalDataPersistence:    types.StringNull(),
			EnableDefaultUser:                types.BoolNull(),
		}),
	}

	regions, diags := (&activeActiveDatabaseResource{}).buildRegionsFromPlan(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("build regions: %v", diags)
	}
	if len(regions) != 1 {
		t.Fatalf("got %d regions, want 1", len(regions))
	}
	if regions[0].LocalThroughputMeasurement != nil {
		t.Fatal("unrelated update included throughput when the attributes were unset")
	}
	if redis.StringValue(regions[0].Password) != "secret-password" {
		t.Fatalf("password %q, want the unrelated override to still be sent", redis.StringValue(regions[0].Password))
	}
}

func TestUnitActiveActiveDatabase_RegionUpdateSendsThroughputWhenBothAttributesAreSet(t *testing.T) {
	plan := &ActiveActiveDatabaseModel{
		GlobalSourceIPs: types.SetNull(types.StringType),
		GlobalAlert:     types.SetNull(types.ObjectType{AttrTypes: getAlertAttrTypes()}),
		OverrideRegion: overrideRegionSet(t, OverrideRegionModel{
			Name:                             types.StringValue("us-east-1"),
			OverrideGlobalAlert:              types.SetNull(types.ObjectType{AttrTypes: getAlertAttrTypes()}),
			OverrideGlobalSourceIPs:          types.SetNull(types.StringType),
			RemoteBackup:                     types.ListNull(types.ObjectType{AttrTypes: getRemoteBackupAttrTypes()}),
			LocalReadOperationsPerSecond:     types.Int64Value(4000),
			LocalWriteOperationsPerSecond:    types.Int64Value(2500),
			OverrideGlobalEnablePasswordless: types.BoolNull(),
			OverrideGlobalDataPersistence:    types.StringNull(),
			EnableDefaultUser:                types.BoolNull(),
			OverrideGlobalPassword:           types.StringNull(),
		}),
	}

	regions, diags := (&activeActiveDatabaseResource{}).buildRegionsFromPlan(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("build regions: %v", diags)
	}
	got := regions[0].LocalThroughputMeasurement
	if got == nil {
		t.Fatal("update omitted configured throughput")
	}
	if redis.IntValue(got.ReadOperationsPerSecond) != 4000 || redis.IntValue(got.WriteOperationsPerSecond) != 2500 {
		t.Fatalf("got read %d write %d, want 4000 and 2500", redis.IntValue(got.ReadOperationsPerSecond), redis.IntValue(got.WriteOperationsPerSecond))
	}
}

func TestUnitActiveActiveDatabase_RefreshLeavesThroughputNullWhenUnset(t *testing.T) {
	state := &ActiveActiveDatabaseModel{
		GlobalEnableDefaultUser: types.BoolValue(true),
		OverrideRegion: overrideRegionSet(t, OverrideRegionModel{
			Name:                             types.StringValue("us-east-1"),
			OverrideGlobalAlert:              types.SetNull(types.ObjectType{AttrTypes: getAlertAttrTypes()}),
			OverrideGlobalSourceIPs:          types.SetNull(types.StringType),
			RemoteBackup:                     types.ListNull(types.ObjectType{AttrTypes: getRemoteBackupAttrTypes()}),
			LocalReadOperationsPerSecond:     types.Int64Null(),
			LocalWriteOperationsPerSecond:    types.Int64Null(),
			OverrideGlobalEnablePasswordless: types.BoolNull(),
			OverrideGlobalDataPersistence:    types.StringNull(),
			EnableDefaultUser:                types.BoolNull(),
			OverrideGlobalPassword:           types.StringNull(),
		}),
	}
	db := &databases.ActiveActiveDatabase{
		CrdbDatabases: []*databases.CrdbDatabase{{
			Region:                   redis.String("us-east-1"),
			ReadOperationsPerSecond:  redis.Int(2000),
			WriteOperationsPerSecond: redis.Int(1500),
		}},
	}

	got, diags := (&activeActiveDatabaseResource{}).buildOverrideRegionFromAPI(context.Background(), db, state)
	if diags.HasError() {
		t.Fatalf("refresh: %v", diags)
	}
	var regions []OverrideRegionModel
	if diags := got.ElementsAs(context.Background(), &regions, false); diags.HasError() {
		t.Fatalf("decode refresh: %v", diags)
	}
	if len(regions) != 1 {
		t.Fatalf("got %d regions, want 1", len(regions))
	}
	if !regions[0].LocalReadOperationsPerSecond.IsNull() || !regions[0].LocalWriteOperationsPerSecond.IsNull() {
		t.Fatalf("refresh stored API throughput read=%s write=%s", regions[0].LocalReadOperationsPerSecond, regions[0].LocalWriteOperationsPerSecond)
	}
}

func overrideRegionSet(t *testing.T, region OverrideRegionModel) types.Set {
	t.Helper()
	var response resource.SchemaResponse
	(&activeActiveDatabaseResource{}).Schema(context.Background(), resource.SchemaRequest{}, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("schema: %v", response.Diagnostics)
	}
	block, ok := response.Schema.Blocks["override_region"].(schema.SetNestedBlock)
	if !ok {
		t.Fatal("override_region is not a set nested block")
	}
	objectType, ok := block.NestedObject.Type().(types.ObjectType)
	if !ok {
		t.Fatalf("override_region element type %T, want an object", block.NestedObject.Type())
	}
	set, diags := types.SetValueFrom(context.Background(), objectType, []OverrideRegionModel{region})
	if diags.HasError() {
		t.Fatalf("override region set: %v", diags)
	}
	return set
}
