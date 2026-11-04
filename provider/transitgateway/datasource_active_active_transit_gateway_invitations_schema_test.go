package transitgateway_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/transitgateway"
)

func TestActiveActiveTransitGatewayInvitationsDataSourceSchemaMatchesModel(t *testing.T) {
	ctx := context.Background()
	dataSource := transitgateway.NewActiveActiveTransitGatewayInvitationsDataSource()
	var response datasource.SchemaResponse
	dataSource.Schema(ctx, datasource.SchemaRequest{}, &response)

	invitations, diags := transitgateway.TransitGatewayInvitationsFromAPI(ctx, nil)
	require.False(t, diags.HasError())

	modelType := types.ObjectType{AttrTypes: customtypes.AttrTypesOf(transitgateway.ActiveActiveTransitGatewayInvitationsDataSourceModel{
		Invitations: invitations,
	})}
	assert.True(t, response.Schema.Type().Equal(modelType), "data-source schema must match its model type")
	assert.Empty(t, response.Schema.Blocks, "data-source schema must not declare blocks")
}
