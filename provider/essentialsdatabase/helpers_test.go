package essentialsdatabase

import (
	"context"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	fixedDatabases "github.com/RedisLabs/rediscloud-go-api/service/fixed/databases"
	"github.com/RedisLabs/rediscloud-go-api/service/latest_backups"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSourceIPsFromAPI(t *testing.T) {
	assert.Empty(t, sourceIPsFromAPI(nil))
	assert.Empty(t, sourceIPsFromAPI(&fixedDatabases.Security{SourceIPs: redis.StringSlice(defaultSourceIP)}))
	assert.Equal(t,
		[]string{"10.0.0.0/8", defaultSourceIP},
		sourceIPsFromAPI(&fixedDatabases.Security{SourceIPs: redis.StringSlice("10.0.0.0/8", defaultSourceIP)}),
	)
}

func TestMatchesAll(t *testing.T) {
	db := &fixedDatabases.FixedDatabase{DatabaseId: redis.Int(42), Name: redis.String("example")}
	byID := func(db *fixedDatabases.FixedDatabase) bool { return redis.IntValue(db.DatabaseId) == 42 }
	byName := func(db *fixedDatabases.FixedDatabase) bool { return redis.StringValue(db.Name) == "other" }

	assert.True(t, matchesAll(db, nil))
	assert.True(t, matchesAll(db, []databaseFilter{byID}))
	assert.False(t, matchesAll(db, []databaseFilter{byID, byName}))
}

func TestReplicaModelsFromAPI(t *testing.T) {
	ctx := context.Background()

	empty, diags := replicaModelsFromAPI(ctx, nil)
	require.False(t, diags.HasError())
	assert.Empty(t, empty)

	replicas, diags := replicaModelsFromAPI(ctx, &fixedDatabases.ReplicaOf{
		SyncSources: []*fixedDatabases.SyncSource{{Endpoint: redis.String("redis://source:6379"), Encryption: redis.Bool(true)}},
	})
	require.False(t, diags.HasError())
	require.Len(t, replicas, 1)

	var syncSources []SyncSourceModel
	require.False(t, replicas[0].SyncSource.ElementsAs(ctx, &syncSources, false).HasError())
	assert.Equal(t, []SyncSourceModel{{
		Endpoint:   types.StringValue("redis://source:6379"),
		Encryption: types.BoolValue(true),
		ServerCert: types.StringValue(""),
	}}, syncSources)
}

func TestModulesAndAlertsFromAPI(t *testing.T) {
	ctx := context.Background()

	alerts, diags := AlertsFromAPI(ctx, nil)
	require.False(t, diags.HasError())
	assert.Empty(t, alerts.Elements())

	alerts, diags = AlertsFromAPI(ctx, &[]*databases.Alert{{Name: redis.String("throughput-higher-than"), Value: redis.Int(80)}})
	require.False(t, diags.HasError())
	var alertModels []AlertModel
	require.False(t, alerts.ElementsAs(ctx, &alertModels, false).HasError())
	assert.Equal(t, []AlertModel{{Name: types.StringValue("throughput-higher-than"), Value: types.Int64Value(80)}}, alertModels)

	modules, diags := ModulesFromAPI(ctx, &[]*databases.Module{{Name: redis.String("RedisJSON")}})
	require.False(t, diags.HasError())
	var moduleModels []ModuleModel
	require.False(t, modules.ElementsAs(ctx, &moduleModels, false).HasError())
	assert.Equal(t, []ModuleModel{{Name: types.StringValue("RedisJSON")}}, moduleModels)
}

func TestRegexRulesFromAPI(t *testing.T) {
	rules, diags := regexRulesFromAPI(context.Background(), nil)
	require.False(t, diags.HasError())
	assert.False(t, rules.IsNull())
	assert.Empty(t, rules.Elements())
}

func TestLatestBackupStatusFromAPI(t *testing.T) {
	ctx := context.Background()

	empty, diags := latestBackupStatusFromAPI(ctx, nil)
	require.False(t, diags.HasError())
	assert.Empty(t, empty.Elements())

	status, diags := latestBackupStatusFromAPI(ctx, &latest_backups.LatestBackupStatus{
		Response: &latest_backups.Response{
			Error: &latest_backups.Error{Type: redis.String("BACKUP_NOT_FOUND"), Status: redis.String("404")},
		},
	})
	require.False(t, diags.HasError())

	var models []LatestBackupStatusModel
	require.False(t, status.ElementsAs(ctx, &models, false).HasError())
	require.Len(t, models, 1)
	assert.Empty(t, models[0].Response.Elements())
	assert.Len(t, models[0].Error.Elements(), 1)
}
