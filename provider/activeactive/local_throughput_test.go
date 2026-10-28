package activeactive

import (
	"context"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestUnrelatedRegionUpdateOmitsThroughputWhenUnset(t *testing.T) {
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

func TestRegionUpdateSendsThroughputWhenBothAttributesAreSet(t *testing.T) {
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

func TestRefreshLeavesThroughputNullWhenUnset(t *testing.T) {
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
	objectType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"name":                                types.StringType,
		"override_global_alert":               types.SetType{ElemType: types.ObjectType{AttrTypes: getAlertAttrTypes()}},
		"override_global_password":            types.StringType,
		"override_global_enable_passwordless": types.BoolType,
		"override_global_source_ips":          types.SetType{ElemType: types.StringType},
		"override_global_data_persistence":    types.StringType,
		"enable_default_user":                 types.BoolType,
		"local_read_operations_per_second":    types.Int64Type,
		"local_write_operations_per_second":   types.Int64Type,
		"remote_backup":                       types.ListType{ElemType: types.ObjectType{AttrTypes: getRemoteBackupAttrTypes()}},
	}}
	set, diags := types.SetValueFrom(context.Background(), objectType, []OverrideRegionModel{region})
	if diags.HasError() {
		t.Fatalf("override region set: %v", diags)
	}
	return set
}
