package activeactive

import (
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildCreateLocalThroughputsUsesConfiguredRegionAndDefaultsTheRest(t *testing.T) {
	overrides := map[string]OverrideRegionModel{
		"us-east-1": {
			Name:                          types.StringValue("us-east-1"),
			LocalReadOperationsPerSecond:  types.Int64Value(2000),
			LocalWriteOperationsPerSecond: types.Int64Value(1500),
		},
		"us-west-2": {
			Name: types.StringValue("us-west-2"),
		},
	}

	got := buildCreateLocalThroughputs([]string{"us-east-1", "us-west-2", "eu-west-1"}, overrides)
	if len(got) != 3 {
		t.Fatalf("got %d regions, want 3", len(got))
	}

	assertThroughput(t, got[0], "us-east-1", 2000, 1500)
	assertThroughput(t, got[1], "us-west-2", 1000, 1000)
	assertThroughput(t, got[2], "eu-west-1", 1000, 1000)
}

func TestBuildCreateLocalThroughputsIgnoresAPartialOverride(t *testing.T) {
	overrides := map[string]OverrideRegionModel{
		"us-east-1": {
			Name:                         types.StringValue("us-east-1"),
			LocalReadOperationsPerSecond: types.Int64Value(2000),
		},
	}

	got := buildCreateLocalThroughputs([]string{"us-east-1"}, overrides)
	assertThroughput(t, got[0], "us-east-1", 1000, 1000)
}

func TestSetLocalThroughputOmitsUnsetAttributes(t *testing.T) {
	props := &databases.LocalRegionProperties{Region: redis.String("us-east-1")}
	setLocalThroughput(props, OverrideRegionModel{Name: types.StringValue("us-east-1")})
	if props.LocalThroughputMeasurement != nil {
		t.Fatal("update included throughput when the attributes were unset")
	}
}

func TestSetLocalThroughputSendsBothValuesWhenSet(t *testing.T) {
	props := &databases.LocalRegionProperties{Region: redis.String("us-east-1")}
	setLocalThroughput(props, OverrideRegionModel{
		Name:                          types.StringValue("us-east-1"),
		LocalReadOperationsPerSecond:  types.Int64Value(4000),
		LocalWriteOperationsPerSecond: types.Int64Value(2500),
	})
	if props.LocalThroughputMeasurement == nil {
		t.Fatal("update omitted configured throughput")
	}
	assertThroughput(t, props.LocalThroughputMeasurement, "us-east-1", 4000, 2500)
}

func TestLocalThroughputStateValuesStayNullWhenUnset(t *testing.T) {
	read, write := localThroughputStateValues(OverrideRegionModel{}, redis.Int(2000), redis.Int(1500))
	if !read.IsNull() || !write.IsNull() {
		t.Fatalf("unset attributes stored API values read=%s write=%s", read, write)
	}
}

func TestLocalThroughputStateValuesCopyTheAPIWhenConfigured(t *testing.T) {
	read, write := localThroughputStateValues(OverrideRegionModel{
		LocalReadOperationsPerSecond:  types.Int64Value(1),
		LocalWriteOperationsPerSecond: types.Int64Value(1),
	}, redis.Int(3000), redis.Int(2500))
	if read.ValueInt64() != 3000 || write.ValueInt64() != 2500 {
		t.Fatalf("got read=%s write=%s, want 3000 and 2500", read, write)
	}
}

func assertThroughput(t *testing.T, got *databases.LocalThroughput, region string, read, write int) {
	t.Helper()
	if redis.StringValue(got.Region) != region {
		t.Errorf("region %q, want %q", redis.StringValue(got.Region), region)
	}
	if redis.IntValue(got.ReadOperationsPerSecond) != read {
		t.Errorf("%s read %d, want %d", region, redis.IntValue(got.ReadOperationsPerSecond), read)
	}
	if redis.IntValue(got.WriteOperationsPerSecond) != write {
		t.Errorf("%s write %d, want %d", region, redis.IntValue(got.WriteOperationsPerSecond), write)
	}
}
