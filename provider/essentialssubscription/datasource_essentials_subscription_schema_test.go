package essentialssubscription_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/essentialssubscription"
)

func TestEssentialsSubscriptionDataSourceSchemaMatchesModel(t *testing.T) {
	dataSource := essentialssubscription.NewEssentialsSubscriptionDataSource()
	var response datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &response)

	modelType := types.ObjectType{AttrTypes: customtypes.AttrTypesOf(essentialssubscription.EssentialsSubscriptionModel{})}
	assert.Equal(t, modelType, response.Schema.Type(), "data-source schema must match its model type")
}
