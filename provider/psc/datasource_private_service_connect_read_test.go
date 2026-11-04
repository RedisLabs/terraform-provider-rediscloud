package psc_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/psc"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestPrivateServiceConnectDataSourcePreservesEmptyStrings(t *testing.T) {
	ctx := t.Context()
	api := testhelpers.NewFakeAPI()
	api.Handle("GET /subscriptions/1/private-service-connect", func(_ *http.Request) (any, error) {
		return map[string]any{"taskId": "psc-task"}, nil
	})
	api.Handle("GET /tasks/psc-task", func(_ *http.Request) (any, error) {
		return map[string]any{
			"taskId":   "psc-task",
			"status":   "processing-completed",
			"response": map[string]any{"resource": map[string]any{"id": 2}},
		}, nil
	})

	source := psc.NewPrivateServiceConnectDataSource()
	var configure datasource.ConfigureResponse
	source.(datasource.DataSourceWithConfigure).Configure(ctx, datasource.ConfigureRequest{ProviderData: testhelpers.NewAPIClient(t, api)}, &configure)
	require.Empty(t, configure.Diagnostics)

	var responseSchema datasource.SchemaResponse
	source.Schema(ctx, datasource.SchemaRequest{}, &responseSchema)
	schemaType := responseSchema.Schema.Type().TerraformType(ctx).(tftypes.Object)
	fields := make(map[string]tftypes.Value, len(schemaType.AttributeTypes))
	for name, typ := range schemaType.AttributeTypes {
		fields[name] = tftypes.NewValue(typ, nil)
	}
	fields["subscription_id"] = tftypes.NewValue(tftypes.String, "1")
	config := tfsdk.Config{Schema: responseSchema.Schema, Raw: tftypes.NewValue(schemaType, fields)}
	response := datasource.ReadResponse{State: tfsdk.State{Schema: responseSchema.Schema}}
	source.Read(ctx, datasource.ReadRequest{Config: config}, &response)
	require.Empty(t, response.Diagnostics)

	var model psc.PrivateServiceConnectModel
	require.Empty(t, response.State.Get(ctx, &model))
	assert.Equal(t, types.StringValue("1/2"), model.ID)
	assert.Equal(t, types.Int64Value(2), model.PrivateServiceConnectServiceID)
	assert.Equal(t, types.StringValue(""), model.ConnectionHostName)
	assert.Equal(t, types.StringValue(""), model.ServiceAttachmentName)
	assert.Equal(t, types.StringValue(""), model.Status)
}
