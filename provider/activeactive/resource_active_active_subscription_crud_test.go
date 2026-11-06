package activeactive_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/activeactive"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func TestActiveActiveSubscriptionRead(t *testing.T) {
	ctx := context.Background()
	fake := testhelpers.NewFakeAPI()
	fake.Handle("GET /subscriptions/123", func(r *http.Request) (any, error) {
		return activeSubscriptionPayload(), nil
	})
	fake.Handle("GET /subscriptions/123/maintenance-windows", func(r *http.Request) (any, error) {
		return map[string]any{"mode": "automatic", "windows": []any{}}, nil
	})
	fake.Handle("GET /subscriptions/123/pricing", func(r *http.Request) (any, error) {
		return map[string]any{"pricing": []any{
			map[string]any{"region": "eu-west-1", "type": "Shards"},
			map[string]any{"region": "us-east-1"},
		}}, nil
	})

	stateModel := blankSubscriptionModel(t)
	stateModel.ID = types.StringValue("123")
	stateModel.RedisVersion = types.StringValue("7.2")
	stateModel.CreationPlan = testCreationPlan(t, ctx)
	keys, keyDiags := activeactive.NewCustomerManagedKeyListValue(ctx, []activeactive.CustomerManagedKeyModel{{
		ResourceName: types.StringValue("key"),
		Region:       types.StringValue("us-east1"),
	}})
	require.False(t, keyDiags.HasError())
	stateModel.CustomerManagedKey = keys

	res := configuredSubscription(t, fake)
	request := resource.ReadRequest{State: subscriptionState(t, stateModel)}
	var response resource.ReadResponse
	response.State = request.State
	res.Read(ctx, request, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %s", response.Diagnostics)
	}

	var got activeactive.ActiveActiveSubscriptionResourceModel
	require.False(t, response.State.Get(ctx, &got).HasError())
	assert.Equal(t, "prod", got.Name.ValueString())
	assert.Equal(t, "7", got.PaymentMethodID.ValueString())
	assert.Equal(t, "credit-card", got.PaymentMethod.ValueString())
	assert.Equal(t, "123456789012", got.AwsAccountID.ValueString())
	assert.Equal(t, "7.2", got.RedisVersion.ValueString())
	assert.True(t, stateModel.CreationPlan.Equal(got.CreationPlan))
	assert.True(t, stateModel.CustomerManagedKey.Equal(got.CustomerManagedKey))

	var prices []utils.PricingModel
	require.False(t, got.Pricing.ElementsAs(ctx, &prices, false).HasError())
	require.Len(t, prices, 2)
	assert.Equal(t, "eu-west-1", prices[0].Region.ValueString())
	assert.Equal(t, "", prices[1].DatabaseName.ValueString())
	assert.False(t, prices[1].DatabaseName.IsNull())
}

func TestActiveActiveSubscriptionReadNotFoundRemovesResource(t *testing.T) {
	fake := testhelpers.NewFakeAPI()
	fake.Handle("GET /subscriptions/123", func(r *http.Request) (any, error) {
		return nil, testhelpers.NotFound()
	})
	stateModel := blankSubscriptionModel(t)
	stateModel.ID = types.StringValue("123")

	res := configuredSubscription(t, fake)
	request := resource.ReadRequest{State: subscriptionState(t, stateModel)}
	var response resource.ReadResponse
	response.State = request.State
	res.Read(context.Background(), request, &response)
	require.False(t, response.Diagnostics.HasError(), response.Diagnostics)
	assert.True(t, response.State.Raw.IsNull())
}

func TestActiveActiveSubscriptionReadEmptyCloudDetails(t *testing.T) {
	fake := testhelpers.NewFakeAPI()
	fake.Handle("GET /subscriptions/123", func(r *http.Request) (any, error) {
		return map[string]any{"id": 123, "status": "active"}, nil
	})
	stateModel := blankSubscriptionModel(t)
	stateModel.ID = types.StringValue("123")

	res := configuredSubscription(t, fake)
	var response resource.ReadResponse
	res.Read(context.Background(), resource.ReadRequest{State: subscriptionState(t, stateModel)}, &response)
	require.True(t, response.Diagnostics.HasError())
	assert.Contains(t, response.Diagnostics[0].Summary(), "cloud details is empty")
}

func TestActiveActiveSubscriptionCreateCustomerManagedKey(t *testing.T) {
	ctx := context.Background()
	fake := testhelpers.NewFakeAPI()
	var body map[string]any
	fake.Handle("POST /subscriptions", func(r *http.Request) (any, error) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		return fake.RegisterTask("create", 321), nil
	})
	fake.Handle("GET /subscriptions/321", func(r *http.Request) (any, error) {
		payload := activeSubscriptionPayload()
		payload["id"] = 321
		payload["status"] = "encryption_key_pending"
		return payload, nil
	})

	plan := blankSubscriptionModel(t)
	plan.Name = types.StringValue("prod")
	plan.PaymentMethod = types.StringValue("credit-card")
	plan.CloudProvider = types.StringValue("AWS")
	plan.CustomerManagedKeyEnabled = types.BoolValue(true)
	plan.PublicEndpointAccess = types.BoolValue(true)
	plan.CreationPlan = testCreationPlan(t, ctx)

	res := configuredSubscription(t, fake)
	var response resource.CreateResponse
	res.Create(ctx, resource.CreateRequest{Plan: subscriptionPlan(t, plan)}, &response)
	require.False(t, response.Diagnostics.HasError(), response.Diagnostics)

	var got activeactive.ActiveActiveSubscriptionResourceModel
	require.False(t, response.State.Get(ctx, &got).HasError())
	assert.Equal(t, "321", got.ID.ValueString())
	assert.Equal(t, "active-active", body["deploymentType"])
	assert.Equal(t, "customer-managed-key", body["persistentStorageEncryptionType"])
	assert.True(t, got.CreationPlan.Equal(plan.CreationPlan))
}

func TestActiveActiveSubscriptionCreateKeepsIDWhenWaitFails(t *testing.T) {
	ctx := context.Background()
	fake := testhelpers.NewFakeAPI()
	fake.Handle("POST /subscriptions", func(r *http.Request) (any, error) {
		return fake.RegisterTask("create", 55), nil
	})
	fake.Handle("GET /subscriptions/55", func(r *http.Request) (any, error) {
		return map[string]any{"id": 55, "status": "pending"}, nil
	})

	plan := blankSubscriptionModel(t)
	plan.Name = types.StringValue("prod")
	plan.PaymentMethod = types.StringValue("credit-card")
	plan.CloudProvider = types.StringValue("AWS")
	plan.CustomerManagedKeyEnabled = types.BoolValue(false)
	plan.PublicEndpointAccess = types.BoolValue(true)
	plan.CreationPlan = testCreationPlan(t, ctx)

	res := configuredSubscription(t, fake)
	short, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()
	var response resource.CreateResponse
	res.Create(short, resource.CreateRequest{Plan: subscriptionPlan(t, plan)}, &response)
	require.True(t, response.Diagnostics.HasError())

	var got activeactive.ActiveActiveSubscriptionResourceModel
	require.False(t, response.State.Get(ctx, &got).HasError())
	assert.Equal(t, "55", got.ID.ValueString())
}

func TestActiveActiveSubscriptionCreateAPIFailureLeavesNoID(t *testing.T) {
	ctx := context.Background()
	fake := testhelpers.NewFakeAPI()
	fake.Handle("POST /subscriptions", func(r *http.Request) (any, error) {
		return nil, testhelpers.StatusError{Code: http.StatusForbidden, Body: map[string]any{"description": "denied"}}
	})

	plan := blankSubscriptionModel(t)
	plan.CreationPlan = testCreationPlan(t, ctx)
	plan.PaymentMethod = types.StringValue("credit-card")
	plan.CloudProvider = types.StringValue("AWS")

	res := configuredSubscription(t, fake)
	var response resource.CreateResponse
	res.Create(ctx, resource.CreateRequest{Plan: subscriptionPlan(t, plan)}, &response)
	require.True(t, response.Diagnostics.HasError())
	assert.True(t, response.State.Raw.IsNull())
}

func TestActiveActiveSubscriptionGracePeriodRequest(t *testing.T) {
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
				require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
				return nil, testhelpers.StatusError{Code: http.StatusForbidden, Body: map[string]any{"description": "grace-period-update-denied"}}
			})

			state := blankSubscriptionModel(t)
			state.ID = types.StringValue("123")
			state.CustomerManagedKeyEnabled = types.BoolValue(true)
			state.CustomerManagedKeyDeletionGracePeriod = types.StringValue("immediate")
			plan := state
			plan.CustomerManagedKeyDeletionGracePeriod = types.StringValue("15-minutes")
			if activation {
				keys, diags := activeactive.NewCustomerManagedKeyListValue(context.Background(), []activeactive.CustomerManagedKeyModel{{
					ResourceName: types.StringValue("projects/test/keys/key"),
					Region:       types.StringValue("us-east1"),
				}})
				require.False(t, diags.HasError())
				plan.CustomerManagedKey = keys
			}

			res := configuredSubscription(t, fake)
			var response resource.UpdateResponse
			res.Update(context.Background(), resource.UpdateRequest{
				Plan:  subscriptionPlan(t, plan),
				State: subscriptionState(t, state),
			}, &response)
			require.True(t, response.Diagnostics.HasError())
			require.Contains(t, response.Diagnostics[0].Summary(), "grace-period-update-denied")
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
			cancel()
			return map[string]any{"id": 123, "status": "active"}, nil
		})
		calls := 0
		fake.Handle("PUT /subscriptions/123", func(r *http.Request) (any, error) {
			calls++
			return nil, testhelpers.StatusError{Code: http.StatusBadRequest, Body: map[string]any{"description": "unexpected update"}}
		})

		state := blankSubscriptionModel(t)
		state.ID = types.StringValue("123")
		state.CustomerManagedKeyEnabled = types.BoolValue(true)
		state.CustomerManagedKeyDeletionGracePeriod = types.StringValue("15-minutes")

		res := configuredSubscription(t, fake)
		var response resource.UpdateResponse
		res.Update(ctx, resource.UpdateRequest{
			Plan:  subscriptionPlan(t, state),
			State: subscriptionState(t, state),
		}, &response)
		require.True(t, response.Diagnostics.HasError())
		fake.WithHandlersPaused(func() { require.Zero(t, calls) })
	})
}

func TestActiveActiveSubscriptionUpdateNameTagsAndEndpoint(t *testing.T) {
	ctx := context.Background()
	fake := testhelpers.NewFakeAPI()
	fake.Handle("GET /subscriptions/123", func(r *http.Request) (any, error) {
		payload := activeSubscriptionPayload()
		payload["publicEndpointAccess"] = false
		return payload, nil
	})
	fake.Handle("GET /subscriptions/123/maintenance-windows", func(r *http.Request) (any, error) {
		return map[string]any{"mode": "automatic"}, nil
	})
	fake.Handle("GET /subscriptions/123/pricing", func(r *http.Request) (any, error) {
		return map[string]any{"pricing": []any{}}, nil
	})
	var subscriptionBody map[string]any
	var tagBody map[string]any
	fake.Handle("PUT /subscriptions/123", func(r *http.Request) (any, error) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&subscriptionBody))
		return fake.RegisterTask("update", 123), nil
	})
	fake.Handle("PUT /subscriptions/123/resource-tags", func(r *http.Request) (any, error) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&tagBody))
		return fake.RegisterTask("tags", 123), nil
	})

	state := blankSubscriptionModel(t)
	state.ID = types.StringValue("123")
	state.Name = types.StringValue("old")
	state.PublicEndpointAccess = types.BoolValue(true)
	plan := state
	plan.Name = types.StringValue("new")
	plan.PublicEndpointAccess = types.BoolValue(false)
	tags, diags := types.MapValueFrom(ctx, types.StringType, map[string]string{"env": "qa"})
	require.False(t, diags.HasError())
	plan.ResourceTags = tags

	res := configuredSubscription(t, fake)
	var response resource.UpdateResponse
	res.Update(ctx, resource.UpdateRequest{
		Plan:  subscriptionPlan(t, plan),
		State: subscriptionState(t, state),
	}, &response)
	require.False(t, response.Diagnostics.HasError(), response.Diagnostics)
	assert.Equal(t, "new", subscriptionBody["name"])
	assert.Equal(t, false, subscriptionBody["publicEndpointAccess"])
	require.NotEmpty(t, tagBody["resourceTags"])
}

func TestActiveActiveSubscriptionDeletionTransitions(t *testing.T) {
	cases := []struct {
		name         string
		states       []string
		deleteResult string
		deletes      int
		wantError    bool
	}{
		{name: "already absent", states: []string{"notfound"}},
		{name: "explicit deleted", states: []string{"deleted"}},
		{name: "already deleting", states: []string{"deleting", "notfound"}},
		{name: "pending to automatic deletion", states: []string{"pending", "deleting", "notfound"}},
		{name: "CMK to active", states: []string{"encryption_key_pending", "active", "active", "deleting", "notfound"}, deletes: 1},
		{name: "CMK to deleting", states: []string{"encryption_key_pending", "deleting", "notfound"}},
		{name: "CMK to absent", states: []string{"encryption_key_pending", "notfound"}},
		{name: "CMK timeout", states: []string{"encryption_key_pending"}, wantError: true},
		{name: "automatic deletion at recheck", states: []string{"active", "deleting", "notfound"}},
		{name: "absent at recheck", states: []string{"active", "notfound"}},
		{name: "CMK at recheck", states: []string{"active", "encryption_key_pending", "active", "notfound"}, deletes: 1},
		{name: "delete returns not found", states: []string{"active"}, deleteResult: "notfound", deletes: 1, wantError: true},
		{name: "delete fails", states: []string{"active"}, deleteResult: "failure", deletes: 1, wantError: true},
		{name: "polling fails after delete", states: []string{"active", "active", "failure"}, deletes: 1, wantError: true},
		{name: "deletion times out", states: []string{"active", "active", "deleting"}, deletes: 1, wantError: true},
		{name: "readiness fails", states: []string{"failure"}, wantError: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := testhelpers.NewFakeAPI()
			fake.Handle("GET /subscriptions/123", subscriptionStatusSequence(tc.states))
			deletes := 0
			fake.Handle("DELETE /subscriptions/123", recordSubscriptionDelete(fake, tc.deleteResult, &deletes))

			state := blankSubscriptionModel(t)
			state.ID = types.StringValue("123")
			res := configuredSubscription(t, fake)
			short, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
			defer cancel()
			var response resource.DeleteResponse
			res.Delete(short, resource.DeleteRequest{State: subscriptionState(t, state)}, &response)
			require.Equal(t, tc.wantError, response.Diagnostics.HasError(), "%v", response.Diagnostics)
			fake.WithHandlersPaused(func() {
				require.Equal(t, tc.deletes, deletes)
			})
		})
	}
}

func TestActiveActiveSubscriptionNilClient(t *testing.T) {
	ctx := context.Background()
	model := blankSubscriptionModel(t)
	model.ID = types.StringValue("123")
	res := activeactive.NewActiveActiveSubscriptionResource()

	var created resource.CreateResponse
	res.Create(ctx, resource.CreateRequest{Plan: subscriptionPlan(t, model)}, &created)
	assert.True(t, created.Diagnostics.HasError())

	var updated resource.UpdateResponse
	res.Update(ctx, resource.UpdateRequest{Plan: subscriptionPlan(t, model), State: subscriptionState(t, model)}, &updated)
	assert.True(t, updated.Diagnostics.HasError())

	var deleted resource.DeleteResponse
	res.Delete(ctx, resource.DeleteRequest{State: subscriptionState(t, model)}, &deleted)
	assert.True(t, deleted.Diagnostics.HasError())
}

func configuredSubscription(t *testing.T, fake *testhelpers.FakeAPI) resource.Resource {
	t.Helper()
	res := activeactive.NewActiveActiveSubscriptionResource()
	var response resource.ConfigureResponse
	res.(resource.ResourceWithConfigure).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: testhelpers.NewAPIClient(t, fake),
	}, &response)
	require.False(t, response.Diagnostics.HasError(), response.Diagnostics)
	return res
}

func subscriptionResourceSchema(t *testing.T) schema.Schema {
	t.Helper()
	var response resource.SchemaResponse
	activeactive.NewActiveActiveSubscriptionResource().Schema(context.Background(), resource.SchemaRequest{}, &response)
	require.False(t, response.Diagnostics.HasError(), response.Diagnostics)
	return response.Schema
}

func blankSubscriptionModel(t *testing.T) activeactive.ActiveActiveSubscriptionResourceModel {
	t.Helper()
	ctx := context.Background()
	maintenance, diags := customtypes.NewMaintenanceList(ctx, nil)
	require.False(t, diags.HasError())
	tags, diags := utils.ResourceTagsFromAPI(ctx, nil)
	require.False(t, diags.HasError())
	return activeactive.ActiveActiveSubscriptionResourceModel{
		CreationPlan:       nullCreationPlan(t, ctx),
		MaintenanceWindows: maintenance,
		Pricing:            types.ListNull(utils.PricingElemType),
		CustomerManagedKey: nullCustomerManagedKeys(t, ctx),
		ResourceTags:       tags,
		Timeouts: timeouts.Value{Object: types.ObjectNull(map[string]attr.Type{
			"create": types.StringType,
			"read":   types.StringType,
			"update": types.StringType,
			"delete": types.StringType,
		})},
	}
}

func nullCreationPlan(t *testing.T, ctx context.Context) activeactive.CreationPlanListValue {
	t.Helper()
	typ := activeactive.NewCreationPlanListType()
	value, err := typ.ValueFromTerraform(ctx, tftypes.NewValue(typ.TerraformType(ctx), nil))
	require.NoError(t, err)
	return value.(activeactive.CreationPlanListValue)
}

func nullCustomerManagedKeys(t *testing.T, ctx context.Context) activeactive.CustomerManagedKeyListValue {
	t.Helper()
	typ := activeactive.NewCustomerManagedKeyListType()
	value, err := typ.ValueFromTerraform(ctx, tftypes.NewValue(typ.TerraformType(ctx), nil))
	require.NoError(t, err)
	return value.(activeactive.CustomerManagedKeyListValue)
}

func subscriptionState(t *testing.T, model activeactive.ActiveActiveSubscriptionResourceModel) tfsdk.State {
	t.Helper()
	state := tfsdk.State{Schema: subscriptionResourceSchema(t)}
	diags := state.Set(context.Background(), &model)
	require.False(t, diags.HasError(), diags)
	return state
}

func subscriptionPlan(t *testing.T, model activeactive.ActiveActiveSubscriptionResourceModel) tfsdk.Plan {
	t.Helper()
	plan := tfsdk.Plan{Schema: subscriptionResourceSchema(t)}
	diags := plan.Set(context.Background(), &model)
	require.False(t, diags.HasError(), diags)
	return plan
}

func testCreationPlan(t *testing.T, ctx context.Context) activeactive.CreationPlanListValue {
	t.Helper()
	regions, diags := activeactive.NewCreationPlanRegionSetValue(ctx, []activeactive.CreationPlanRegionModel{
		{
			Region:                   types.StringValue("us-east-1"),
			NetworkingDeploymentCIDR: types.StringValue("10.0.0.0/24"),
			WriteOperationsPerSecond: types.Int64Value(1000),
			ReadOperationsPerSecond:  types.Int64Value(2000),
		},
		{
			Region:                   types.StringValue("eu-west-1"),
			NetworkingDeploymentCIDR: types.StringValue("10.1.0.0/24"),
			WriteOperationsPerSecond: types.Int64Value(3000),
			ReadOperationsPerSecond:  types.Int64Value(4000),
		},
	})
	require.False(t, diags.HasError())
	modules, diags := types.ListValueFrom(ctx, types.StringType, []string{"RedisJSON"})
	require.False(t, diags.HasError())
	plan, diags := activeactive.NewCreationPlanListValue(ctx, []activeactive.CreationPlanModel{{
		DatasetSizeInGB: types.Float64Value(1.5),
		Quantity:        types.Int64Value(2),
		Modules:         modules,
		Region:          regions,
	}})
	require.False(t, diags.HasError())
	return plan
}

func activeSubscriptionPayload() map[string]any {
	return map[string]any{
		"id":                123,
		"name":              "prod",
		"status":            "active",
		"paymentMethodType": "credit-card",
		"paymentMethodId":   7,
		"cloudDetails": []any{map[string]any{
			"provider":     "AWS",
			"awsAccountId": "123456789012",
			"resourceTags": []any{map[string]any{"key": "env", "value": "qa"}},
		}},
		"publicEndpointAccess": true,
	}
}

func subscriptionStatusSequence(states []string) testhelpers.HandlerFunc {
	index := 0
	return func(r *http.Request) (any, error) {
		state := states[index]
		if index < len(states)-1 {
			index++
		}
		switch state {
		case "notfound":
			return nil, testhelpers.NotFound()
		case "failure":
			return nil, testhelpers.StatusError{Code: http.StatusForbidden, Body: map[string]any{"description": "denied"}}
		}
		return map[string]any{"id": 123, "status": state}, nil
	}
}

func recordSubscriptionDelete(fake *testhelpers.FakeAPI, result string, calls *int) testhelpers.HandlerFunc {
	return func(r *http.Request) (any, error) {
		*calls++
		switch result {
		case "notfound":
			return nil, testhelpers.NotFound()
		case "failure":
			return nil, testhelpers.StatusError{Code: http.StatusForbidden, Body: map[string]any{"description": "denied"}}
		}
		return fake.RegisterTask("subscriptionDelete", 123), nil
	}
}
