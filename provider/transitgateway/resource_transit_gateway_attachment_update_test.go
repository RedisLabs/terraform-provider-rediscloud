package transitgateway_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/transitgateway"
)

func TestTransitGatewayAttachmentUpdateRejectsIdentityChanges(t *testing.T) {
	for _, tt := range []struct {
		name        string
		newResource func() resource.Resource
		changes     map[string]attr.Value
	}{
		{
			name:        "Pro",
			newResource: transitgateway.NewTransitGatewayAttachmentResource,
			changes: map[string]attr.Value{
				"subscription_id": types.StringValue("2"),
				"tgw_id":          types.Int64Value(4),
			},
		},
		{
			name:        "Active-Active",
			newResource: transitgateway.NewActiveActiveTransitGatewayAttachmentResource,
			changes: map[string]attr.Value{
				"subscription_id": types.StringValue("2"),
				"region_id":       types.StringValue("3"),
				"tgw_id":          types.Int64Value(4),
			},
		},
	} {
		for field, value := range tt.changes {
			t.Run(tt.name+"/"+field, func(t *testing.T) {
				ctx := t.Context()
				apiCalls := 0
				fake := testhelpers.NewFakeAPI().Handle("/subscriptions/", func(_ *http.Request) (any, error) {
					apiCalls++
					return nil, testhelpers.StatusError{Code: http.StatusBadRequest}
				})
				r := tt.newResource()
				var configured resource.ConfigureResponse
				r.(resource.ResourceWithConfigure).Configure(ctx, resource.ConfigureRequest{ProviderData: testhelpers.NewAPIClient(t, fake)}, &configured)
				require.False(t, configured.Diagnostics.HasError(), "%v", configured.Diagnostics)
				schema := resourceSchema(t, r)
				prior := tfsdk.State{Schema: schema.Schema}
				cidrs, diags := types.ListValueFrom(ctx, types.StringType, []string{"10.0.0.0/24"})
				require.False(t, diags.HasError(), "%v", diags)
				if tt.name == "Pro" {
					diags = prior.Set(ctx, transitgateway.TransitGatewayAttachmentResourceModel{
						ID: types.StringValue("1/3"), SubscriptionID: types.StringValue("1"),
						TgwID: types.Int64Value(3), Cidrs: cidrs, Timeouts: timeoutsValue(t, schema),
					})
				} else {
					diags = prior.Set(ctx, transitgateway.ActiveActiveTransitGatewayAttachmentResourceModel{
						ID: types.StringValue("1/2/3"), SubscriptionID: types.StringValue("1"), RegionID: types.StringValue("2"),
						TgwID: types.Int64Value(3), Cidrs: cidrs, Timeouts: timeoutsValue(t, schema),
					})
				}
				require.False(t, diags.HasError(), "%v", diags)
				plan := tfsdk.Plan{Schema: schema.Schema, Raw: prior.Raw}
				diags = plan.SetAttribute(ctx, path.Root(field), value)
				require.False(t, diags.HasError(), "%v", diags)

				response := resource.UpdateResponse{State: prior}
				r.Update(ctx, resource.UpdateRequest{Plan: plan, State: prior}, &response)

				assert.Zero(t, apiCalls, "identity changes must not send CIDRs to the API")
				require.True(t, response.Diagnostics.HasError())
				assert.Equal(t, "Transit Gateway attachment identity cannot change in place", response.Diagnostics.Errors()[0].Summary())
				assert.True(t, prior.Raw.Equal(response.State.Raw))
			})
		}
	}
}
