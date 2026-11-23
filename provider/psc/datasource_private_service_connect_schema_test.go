package psc_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/psc"
)

func TestPrivateServiceConnectDataSourceSchemaMatchesModel(t *testing.T) {
	var response datasource.SchemaResponse
	psc.NewPrivateServiceConnectDataSource().Schema(context.Background(), datasource.SchemaRequest{}, &response)

	modelType := types.ObjectType{AttrTypes: customtypes.AttrTypesOf(psc.PrivateServiceConnectModel{})}
	assert.Equal(t, modelType, response.Schema.Type(), "data-source schema must match its model type")
}
