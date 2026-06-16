package activeactive

import (
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFlattenBackupPlan_AbsentBackupIsEmptyList(t *testing.T) {
	cases := []struct {
		name   string
		backup *databases.Backup
	}{
		{name: "nil backup", backup: nil},
		{name: "disabled backup", backup: &databases.Backup{Enabled: redis.Bool(false)}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			list, diags := flattenBackupPlan(tc.backup, "")
			if diags.HasError() {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if list.IsNull() {
				t.Fatalf("expected empty list, got null — would cause 'planned set element does not correlate' on block removal")
			}
			if len(list.Elements()) != 0 {
				t.Fatalf("expected empty list, got %d elements", len(list.Elements()))
			}
		})
	}
}

func TestFlattenBackupPlan_TimeUTC(t *testing.T) {
	cases := []struct {
		name      string
		timeUTC   *string
		wantNull  bool
		wantValue string
	}{
		{name: "nil pointer is null", timeUTC: nil, wantNull: true},
		{name: "empty string is null", timeUTC: redis.String(""), wantNull: true},
		{name: "set value is preserved", timeUTC: redis.String("12:00"), wantValue: "12:00"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			backup := &databases.Backup{
				Enabled:     redis.Bool(true),
				Interval:    redis.String("every-24-hours"),
				TimeUTC:     tc.timeUTC,
				Destination: redis.String("s3://example"),
			}

			list, diags := flattenBackupPlan(backup, "aws-s3")
			if diags.HasError() {
				t.Fatalf("unexpected diags: %v", diags)
			}

			elems := list.Elements()
			if len(elems) != 1 {
				t.Fatalf("expected 1 element, got %d", len(elems))
			}

			obj := elems[0].(types.Object)
			timeUTC := obj.Attributes()["time_utc"].(types.String)

			if tc.wantNull {
				if !timeUTC.IsNull() {
					t.Fatalf("expected time_utc to be null, got %q", timeUTC.ValueString())
				}
				return
			}
			if timeUTC.IsNull() {
				t.Fatalf("expected time_utc %q, got null", tc.wantValue)
			}
			if got := timeUTC.ValueString(); got != tc.wantValue {
				t.Fatalf("expected time_utc %q, got %q", tc.wantValue, got)
			}
		})
	}
}
