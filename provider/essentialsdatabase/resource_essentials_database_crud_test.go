package essentialsdatabase

import (
	"context"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	fixedDatabases "github.com/RedisLabs/rediscloud-go-api/service/fixed/databases"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisVersionUpgradeRequested(t *testing.T) {
	model := func(requested, actual string) *EssentialsDatabaseResourceModel {
		return &EssentialsDatabaseResourceModel{RedisVersion: types.StringValue(requested), RedisVersionActual: types.StringValue(actual)}
	}
	assert.True(t, redisVersionUpgradeRequested(model("8.2", "8.2"), model("8.4", "")))
	assert.False(t, redisVersionUpgradeRequested(model("8.6", "8.6"), model("8.4", "")), "a satisfied minimum must not upgrade")
	assert.False(t, redisVersionUpgradeRequested(model("8.4", "8.4"), model("8.4", "")))
	assert.False(t, redisVersionUpgradeRequested(model("", "8.4"), model("8.6", "")))
	assert.True(t, redisVersionUpgradeRequested(model("7.4", "7.4"), model("8.0", "")))
}

func TestSourceIPsIntoModel(t *testing.T) {
	ctx := context.Background()
	empty := types.ListValueMust(types.StringType, []attr.Value{})
	allowAll := types.ListValueMust(types.StringType, []attr.Value{types.StringValue(defaultSourceIP)})

	got, diags := sourceIPsIntoModel(ctx, nil, types.ListNull(types.StringType))
	require.False(t, diags.HasError())
	assert.True(t, got.IsNull())

	got, _ = sourceIPsIntoModel(ctx, &fixedDatabases.Security{SourceIPs: redis.StringSlice(defaultSourceIP)}, empty)
	assert.Equal(t, empty, got)

	got, _ = sourceIPsIntoModel(ctx, &fixedDatabases.Security{SourceIPs: redis.StringSlice(defaultSourceIP)}, allowAll)
	assert.Equal(t, allowAll, got)

	got, _ = sourceIPsIntoModel(ctx, &fixedDatabases.Security{SourceIPs: redis.StringSlice("10.0.0.0/8")}, allowAll)
	assert.Equal(t, types.ListValueMust(types.StringType, []attr.Value{types.StringValue("10.0.0.0/8")}), got)
}

func TestTagsIntoModel(t *testing.T) {
	empty := types.MapValueMust(types.StringType, map[string]attr.Value{})
	apiTags := types.MapValueMust(types.StringType, map[string]attr.Value{"env": types.StringValue("qa")})

	assert.Equal(t, apiTags, tagsIntoModel(apiTags, types.MapNull(types.StringType)))
	assert.Equal(t, empty, tagsIntoModel(empty, empty))
	assert.True(t, tagsIntoModel(empty, apiTags).IsNull())
}

func TestReplicaIntoModelKeepsUnsetSyncSourceFields(t *testing.T) {
	ctx := context.Background()
	priorSources, diags := types.SetValueFrom(ctx, syncSourceElemType, []SyncSourceModel{{
		Endpoint:   types.StringValue("redis://source:6379"),
		Encryption: types.BoolNull(),
		ServerCert: types.StringNull(),
	}})
	require.False(t, diags.HasError())
	prior, diags := types.ListValueFrom(ctx, replicaElemType, []ReplicaModel{{SyncSource: priorSources}})
	require.False(t, diags.HasError())

	got, diags := replicaIntoModel(ctx, &fixedDatabases.ReplicaOf{SyncSources: []*fixedDatabases.SyncSource{{
		Endpoint:   redis.String("redis://source:6379"),
		Encryption: redis.Bool(false),
		ServerCert: redis.String(""),
	}}}, prior)
	require.False(t, diags.HasError())
	assert.Equal(t, prior, got)

	got, diags = replicaIntoModel(ctx, nil, prior)
	require.False(t, diags.HasError())
	assert.Empty(t, got.Elements())
}
