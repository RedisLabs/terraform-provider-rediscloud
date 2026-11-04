package psc_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/psc"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestEndpointReplacementPlansNewIdentity(t *testing.T) {
	tests := []struct {
		name    string
		factory func() resource.Resource
		typeID  string
	}{
		{"Pro", psc.NewPrivateServiceConnectEndpointResource, "test_private_service_connect_endpoint"},
		{"Active-Active", psc.NewActiveActivePrivateServiceConnectEndpointResource, "test_active_active_private_service_connect_endpoint"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertEndpointReplacementPlansNewIdentity(t, test.factory, test.typeID)
		})
	}
}

func assertEndpointReplacementPlansNewIdentity(t *testing.T, factory func() resource.Resource, typeID string) {
	t.Helper()
	ctx := t.Context()
	provider, err := testhelpers.FrameworkProviderFactories(nil, factory)["test"]()
	require.NoError(t, err)
	schemas, err := provider.GetProviderSchema(ctx, &tfprotov5.GetProviderSchemaRequest{})
	require.NoError(t, err)
	schemaType := schemas.ResourceSchemas[typeID].ValueType().(tftypes.Object)

	prior := make(map[string]tftypes.Value, len(schemaType.AttributeTypes))
	for name, attributeType := range schemaType.AttributeTypes {
		prior[name] = tftypes.NewValue(attributeType, nil)
	}
	prior["subscription_id"] = tftypes.NewValue(tftypes.String, "1")
	prior["private_service_connect_service_id"] = tftypes.NewValue(tftypes.Number, int64(2))
	prior["gcp_project_id"] = tftypes.NewValue(tftypes.String, "project")
	prior["gcp_vpc_name"] = tftypes.NewValue(tftypes.String, "network")
	prior["gcp_vpc_subnet_name"] = tftypes.NewValue(tftypes.String, "subnet")
	prior["endpoint_connection_name"] = tftypes.NewValue(tftypes.String, "endpoint")
	prior["id"] = tftypes.NewValue(tftypes.String, "1/2/3")
	prior["private_service_connect_endpoint_id"] = tftypes.NewValue(tftypes.Number, int64(3))
	prior["service_attachments"] = tftypes.NewValue(schemaType.AttributeTypes["service_attachments"], []tftypes.Value{})
	if regionType, ok := schemaType.AttributeTypes["region_id"]; ok {
		prior["region_id"] = tftypes.NewValue(regionType, int64(4))
	}

	proposed := make(map[string]tftypes.Value, len(prior))
	config := make(map[string]tftypes.Value, len(prior))
	for name, value := range prior {
		proposed[name] = value
		config[name] = value
	}
	proposed["gcp_vpc_subnet_name"] = tftypes.NewValue(tftypes.String, "new-subnet")
	config["gcp_vpc_subnet_name"] = proposed["gcp_vpc_subnet_name"]
	computedFields := []string{"id", "private_service_connect_endpoint_id", "service_attachments"}
	for _, name := range computedFields {
		config[name] = tftypes.NewValue(schemaType.AttributeTypes[name], nil)
	}

	priorValue, err := tfprotov5.NewDynamicValue(schemaType, tftypes.NewValue(schemaType, prior))
	require.NoError(t, err)
	proposedValue, err := tfprotov5.NewDynamicValue(schemaType, tftypes.NewValue(schemaType, proposed))
	require.NoError(t, err)
	configValue, err := tfprotov5.NewDynamicValue(schemaType, tftypes.NewValue(schemaType, config))
	require.NoError(t, err)
	response, err := provider.PlanResourceChange(ctx, &tfprotov5.PlanResourceChangeRequest{
		TypeName:         typeID,
		PriorState:       &priorValue,
		ProposedNewState: &proposedValue,
		Config:           &configValue,
	})
	require.NoError(t, err)
	require.Empty(t, response.Diagnostics)
	require.Len(t, response.RequiresReplace, 1)
	assert.True(t, response.RequiresReplace[0].Equal(tftypes.NewAttributePath().WithAttributeName("gcp_vpc_subnet_name")))

	plannedValue, err := response.PlannedState.Unmarshal(schemaType)
	require.NoError(t, err)
	var planned map[string]tftypes.Value
	require.NoError(t, plannedValue.As(&planned))
	for _, name := range computedFields {
		assert.False(t, planned[name].IsKnown(), name)
	}
}
