package privatelink_test

import (
	"context"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	pl "github.com/RedisLabs/rediscloud-go-api/service/privatelink"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/privatelink"
)

func TestPrincipalsFromAPI(t *testing.T) {
	ctx := context.Background()

	t.Run("nil yields a known empty set", func(t *testing.T) {
		principals, diags := privatelink.PrincipalsFromAPI(ctx, nil)
		require.False(t, diags.HasError())
		assert.False(t, principals.IsNull())
		assert.False(t, principals.IsUnknown())
		assert.Empty(t, principals.Elements())
	})

	t.Run("missing fields map to empty strings", func(t *testing.T) {
		principals, diags := privatelink.PrincipalsFromAPI(ctx, []*pl.PrivateLinkPrincipal{
			{Principal: redis.String("123456789012"), Type: redis.String("aws_account")},
			{Principal: redis.String("234567890123"), Type: redis.String("aws_account"), Alias: redis.String("second")},
		})
		require.False(t, diags.HasError())

		models, diags := principals.AsModels(ctx)
		require.False(t, diags.HasError())
		assert.ElementsMatch(t, []privatelink.PrincipalModel{
			{Principal: types.StringValue("123456789012"), PrincipalType: types.StringValue("aws_account"), PrincipalAlias: types.StringValue("")},
			{Principal: types.StringValue("234567890123"), PrincipalType: types.StringValue("aws_account"), PrincipalAlias: types.StringValue("second")},
		}, models)
	})
}

func TestPrincipalSetType(t *testing.T) {
	ctx := context.Background()
	setType := privatelink.NewPrincipalSetType()

	assert.True(t, setType.Equal(privatelink.NewPrincipalSetType()))
	assert.False(t, setType.Equal(types.SetType{ElemType: setType.ElementType()}))

	principals, diags := privatelink.PrincipalsFromAPI(ctx, []*pl.PrivateLinkPrincipal{{Principal: redis.String("123456789012"), Type: redis.String("aws_account")}})
	require.False(t, diags.HasError())

	terraformValue, err := principals.ToTerraformValue(ctx)
	require.NoError(t, err)
	value, err := setType.ValueFromTerraform(ctx, terraformValue)
	require.NoError(t, err)
	assert.True(t, principals.Equal(value))
	assert.Equal(t, setType, value.Type(ctx))

	null := privatelink.NullPrincipalSet()
	assert.True(t, null.IsNull())
	assert.Equal(t, setType, null.Type(ctx))
}

func TestConnectionsFromAPI(t *testing.T) {
	ctx := context.Background()

	empty, diags := privatelink.ConnectionsFromAPI(ctx, nil)
	require.False(t, diags.HasError())
	assert.False(t, empty.IsNull())
	assert.Empty(t, empty.Elements())

	connections, diags := privatelink.ConnectionsFromAPI(ctx, []*pl.PrivateLinkConnection{{
		AssociationId:   redis.String("assoc-123"),
		ConnectionId:    redis.String("conn-456"),
		Type:            redis.String("vpc-endpoint"),
		OwnerId:         redis.String("owner-789"),
		AssociationDate: redis.String("2024-01-01"),
	}, {}})
	require.False(t, diags.HasError())

	var models []privatelink.ConnectionModel
	require.False(t, connections.ElementsAs(ctx, &models, false).HasError())
	assert.ElementsMatch(t, []privatelink.ConnectionModel{
		{
			AssociationID:   types.StringValue("assoc-123"),
			ConnectionID:    types.StringValue("conn-456"),
			ConnectionType:  types.StringValue("vpc-endpoint"),
			OwnerID:         types.StringValue("owner-789"),
			AssociationDate: types.StringValue("2024-01-01"),
		},
		{
			AssociationID:   types.StringValue(""),
			ConnectionID:    types.StringValue(""),
			ConnectionType:  types.StringValue(""),
			OwnerID:         types.StringValue(""),
			AssociationDate: types.StringValue(""),
		},
	}, models)
}

func TestDatabasesFromAPI(t *testing.T) {
	ctx := context.Background()

	empty, diags := privatelink.DatabasesFromAPI(ctx, nil)
	require.False(t, diags.HasError())
	assert.False(t, empty.IsNull())
	assert.Empty(t, empty.Elements())

	databases, diags := privatelink.DatabasesFromAPI(ctx, []*pl.PrivateLinkDatabase{{
		DatabaseId:           redis.Int(123),
		Port:                 redis.Int(6379),
		ResourceLinkEndpoint: redis.String("endpoint.example.com"),
	}})
	require.False(t, diags.HasError())

	var models []privatelink.DatabaseModel
	require.False(t, databases.ElementsAs(ctx, &models, false).HasError())
	assert.Equal(t, []privatelink.DatabaseModel{{
		DatabaseID:           types.Int64Value(123),
		Port:                 types.Int64Value(6379),
		ResourceLinkEndpoint: types.StringValue("endpoint.example.com"),
	}}, models)
}
