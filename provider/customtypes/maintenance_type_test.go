package customtypes_test

import (
	"context"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/maintenance"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
)

func maintenanceStringPointers(values ...string) []*string {
	result := make([]*string, len(values))
	for index, value := range values {
		result[index] = redis.String(value)
	}
	return result
}

func TestNewMaintenanceList(t *testing.T) {
	ctx := context.Background()

	t.Run("nil maintenance yields a null list", func(t *testing.T) {
		list, diags := customtypes.NewMaintenanceList(ctx, nil)

		require.False(t, diags.HasError())
		assert.True(t, list.IsNull())
	})

	t.Run("manual mode maps windows and days", func(t *testing.T) {
		apiMaintenance := &maintenance.Maintenance{
			Mode: redis.String("manual"),
			Windows: []*maintenance.Window{
				{
					StartHour:       redis.Int(22),
					DurationInHours: redis.Int(8),
					Days:            maintenanceStringPointers("Monday", "Thursday"),
				},
				{
					StartHour:       redis.Int(12),
					DurationInHours: redis.Int(6),
					Days:            maintenanceStringPointers("Friday", "Saturday", "Sunday"),
				},
			},
		}

		list, diags := customtypes.NewMaintenanceList(ctx, apiMaintenance)

		require.False(t, diags.HasError())
		require.False(t, list.IsNull())

		entries, modelDiags := list.AsModels(ctx)
		require.False(t, modelDiags.HasError())
		require.Len(t, entries, 1)
		assert.Equal(t, "manual", entries[0].Mode.ValueString())

		var windows []customtypes.MaintenanceWindowModel
		require.False(t, entries[0].Window.ElementsAs(ctx, &windows, false).HasError())
		require.Len(t, windows, 2)
		assert.EqualValues(t, 22, windows[0].StartHour.ValueInt64())
		assert.EqualValues(t, 8, windows[0].DurationInHours.ValueInt64())

		var firstWindowDays []string
		require.False(t, windows[0].Days.ElementsAs(ctx, &firstWindowDays, false).HasError())
		assert.Equal(t, []string{"Monday", "Thursday"}, firstWindowDays)

		assert.EqualValues(t, 12, windows[1].StartHour.ValueInt64())
		var secondWindowDays []string
		require.False(t, windows[1].Days.ElementsAs(ctx, &secondWindowDays, false).HasError())
		assert.Equal(t, []string{"Friday", "Saturday", "Sunday"}, secondWindowDays)
	})

	t.Run("automatic mode has an empty known window list", func(t *testing.T) {
		list, diags := customtypes.NewMaintenanceList(ctx, &maintenance.Maintenance{
			Mode: redis.String("automatic"),
		})

		require.False(t, diags.HasError())
		entries, modelDiags := list.AsModels(ctx)
		require.False(t, modelDiags.HasError())
		require.Len(t, entries, 1)
		assert.False(t, entries[0].Window.IsNull())
		assert.Empty(t, entries[0].Window.Elements())
	})
}

func TestMaintenanceListTypeEqual(t *testing.T) {
	assert.True(t, customtypes.NewMaintenanceListType().Equal(customtypes.NewMaintenanceListType()))
	assert.False(t, customtypes.NewMaintenanceListType().Equal(types.ListType{}))
}
