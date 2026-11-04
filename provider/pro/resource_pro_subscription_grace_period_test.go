package pro_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/pro"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func cmkSubscriptionModel(gracePeriod string) *pro.ProSubscriptionResourceModel {
	return &pro.ProSubscriptionResourceModel{
		CustomerManagedKeyEnabled:             types.BoolValue(true),
		CustomerManagedKeyDeletionGracePeriod: types.StringValue(gracePeriod),
		CustomerManagedKey: types.ListValueMust(pro.CustomerManagedKeyElemType, []attr.Value{
			types.ObjectValueMust(pro.CustomerManagedKeyElemType.AttrTypes, map[string]attr.Value{
				"resource_name": types.StringValue("projects/test/keys/key"),
			}),
		}),
	}
}

func TestProSubscriptionGracePeriodRequest(t *testing.T) {
	for _, activation := range []bool{false, true} {
		label := "grace period only"
		if activation {
			label = "activation includes keys"
		}
		t.Run(label, func(t *testing.T) {
			fake := testhelpers.NewFakeAPI()
			fake.Handle("GET /subscriptions/123", func(r *http.Request) (any, error) {
				status := "active"
				if activation {
					status = "encryption_key_pending"
				}
				return map[string]any{"id": 123, "status": status}, nil
			})
			var payload map[string]any
			calls := 0
			fake.Handle("PUT /subscriptions/123", func(r *http.Request) (any, error) {
				calls++
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					return nil, err
				}
				if !activation {
					if _, exists := payload["customerManagedKeys"]; exists {
						return nil, testhelpers.StatusError{Code: http.StatusBadRequest, Body: map[string]any{"description": "customer keys must be omitted"}}
					}
				}
				// Stop here to also verify the resource propagates an API failure.
				return nil, testhelpers.StatusError{Code: http.StatusForbidden, Body: map[string]any{"description": "grace-period-update-denied"}}
			})

			diags := pro.ApplyProSubscriptionUpdates(context.Background(), testhelpers.NewAPIClient(t, fake), 123, cmkSubscriptionModel("15-minutes"), cmkSubscriptionModel("immediate"))
			require.True(t, diags.HasError())
			require.Contains(t, diags.Errors()[0].Detail(), "grace-period-update-denied")
			fake.WithHandlersPaused(func() {
				require.Equal(t, 1, calls)
				require.Equal(t, "15-minutes", payload["deletionGracePeriod"])
				if activation {
					require.NotEmpty(t, payload["customerManagedKeys"])
				} else {
					require.Equal(t, map[string]any{"deletionGracePeriod": "15-minutes"}, payload)
				}
			})
		})
	}

	t.Run("unchanged grace period", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		fake := testhelpers.NewFakeAPI()
		fake.Handle("GET /subscriptions/123", func(r *http.Request) (any, error) {
			cancel() // Stop the subsequent active waiter; no update should be sent.
			return map[string]any{"id": 123, "status": "active"}, nil
		})
		calls := 0
		fake.Handle("PUT /subscriptions/123", func(r *http.Request) (any, error) {
			calls++
			return nil, testhelpers.StatusError{Code: http.StatusBadRequest, Body: map[string]any{"description": "unexpected update"}}
		})
		diags := pro.ApplyProSubscriptionUpdates(ctx, testhelpers.NewAPIClient(t, fake), 123, cmkSubscriptionModel("15-minutes"), cmkSubscriptionModel("15-minutes"))
		require.True(t, diags.HasError())
		fake.WithHandlersPaused(func() { require.Zero(t, calls) })
	})
}

func TestProSubscriptionImportDefaultsGracePeriod(t *testing.T) {
	for _, tc := range []struct{ name, apiValue, want string }{
		{"no API value", "", "immediate"},
		{"API value", "15-minutes", "15-minutes"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			fake := testhelpers.NewFakeAPI()
			fake.Handle("GET /subscriptions/123", func(*http.Request) (any, error) {
				sub := map[string]any{
					"id": 123, "name": "sub", "status": "active", "paymentMethodType": "credit-card", "memoryStorage": "ram",
					"cloudDetails": []any{map[string]any{
						"provider": "AWS", "cloudAccountId": 1,
						"regions": []any{map[string]any{"region": "eu-west-1", "multipleAvailabilityZones": false}},
					}},
				}
				if tc.apiValue != "" {
					sub["deletionGracePeriod"] = tc.apiValue
				}
				return sub, nil
			})
			fake.Handle("GET /subscriptions/123/maintenance-windows", func(*http.Request) (any, error) {
				return map[string]any{"mode": "automatic"}, nil
			})
			fake.Handle("GET /subscriptions/123/pricing", func(*http.Request) (any, error) {
				return map[string]any{"pricing": []any{}}, nil
			})
			server, err := testhelpers.FrameworkProviderFactories(testhelpers.NewAPIClient(t, fake), pro.NewProSubscriptionResource)[testhelpers.FrameworkTestProviderTypeName]()
			require.NoError(t, err)
			_, err = server.ConfigureProvider(ctx, &tfprotov5.ConfigureProviderRequest{})
			require.NoError(t, err)

			typeName := testhelpers.FrameworkTestProviderTypeName + "_subscription"
			imported, err := server.ImportResourceState(ctx, &tfprotov5.ImportResourceStateRequest{TypeName: typeName, ID: "123"})
			require.NoError(t, err)
			require.Empty(t, imported.Diagnostics)
			require.Len(t, imported.ImportedResources, 1)
			read, err := server.ReadResource(ctx, &tfprotov5.ReadResourceRequest{
				TypeName:     typeName,
				CurrentState: imported.ImportedResources[0].State,
				Private:      imported.ImportedResources[0].Private,
			})
			require.NoError(t, err)
			for _, d := range read.Diagnostics {
				t.Fatalf("read diagnostic: %s: %s", d.Summary, d.Detail)
			}

			var resp resource.SchemaResponse
			pro.NewProSubscriptionResource().Schema(ctx, resource.SchemaRequest{}, &resp)
			raw, err := read.NewState.Unmarshal(resp.Schema.Type().TerraformType(ctx))
			require.NoError(t, err)
			var model pro.ProSubscriptionResourceModel
			require.False(t, (&tfsdk.State{Schema: resp.Schema, Raw: raw}).Get(ctx, &model).HasError())
			require.Equal(t, types.StringValue(tc.want), model.CustomerManagedKeyDeletionGracePeriod)
		})
	}
}
