package essentialssubscription_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	fixedSubscriptions "github.com/RedisLabs/rediscloud-go-api/service/fixed/subscriptions"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/essentialssubscription"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestEssentialsSubscriptionCreateKeepsIDWhenFinalReadFails(t *testing.T) {
	for _, test := range []struct {
		name    string
		status  int
		summary string
	}{
		{"failed read", http.StatusBadRequest, "Unable to Read Essentials Subscription"},
		{"missing subscription", http.StatusNotFound, "Essentials Subscription Not Found"},
	} {
		t.Run(test.name, func(t *testing.T) {
			fake := testhelpers.NewFakeAPI()
			fake.Handle("POST /fixed/subscriptions", func(*http.Request) (any, error) {
				return fake.RegisterTask("create-fixed-subscription", 42), nil
			})
			reads := 0
			fake.Handle("GET /fixed/subscriptions/42", func(*http.Request) (any, error) {
				reads++
				if reads == 1 {
					return fixedSubscriptions.FixedSubscriptionResponse{Status: redis.String("active")}, nil
				}
				return nil, testhelpers.StatusError{Code: test.status}
			})
			server := newEssentialsSubscriptionTestServer(t, fake)
			plan := essentialssubscription.EssentialsSubscriptionResourceModel{
				ID:              types.StringUnknown(),
				Name:            types.StringValue("original"),
				PlanID:          types.Int64Value(2),
				Status:          types.StringUnknown(),
				PaymentMethod:   types.StringUnknown(),
				PaymentMethodID: types.Int64Unknown(),
				CreationDate:    types.StringUnknown(),
			}

			result := applyEssentialsSubscriptionTestChange(t, server, nil, plan, nil)

			require.Len(t, result.Diagnostics, 1)
			require.Equal(t, tfprotov5.DiagnosticSeverityError, result.Diagnostics[0].Severity)
			require.Equal(t, test.summary, result.Diagnostics[0].Summary)
			recovered := readEssentialsSubscriptionTestState(t, result.NewState)
			require.Equal(t, "42", recovered.ID.ValueString())
			require.Equal(t, plan.Name, recovered.Name)
			require.Equal(t, plan.PlanID, recovered.PlanID)
			require.Equal(t, "credit-card", recovered.PaymentMethod.ValueString())
			fake.WithHandlersPaused(func() { require.Equal(t, 2, reads) })
		})
	}
}

func TestEssentialsSubscriptionUpdateKeepsEffectivePaymentMethod(t *testing.T) {
	for _, test := range []struct {
		name         string
		create       bool
		legacyMethod string
		wantMethod   *string
	}{
		{name: "legacy marketplace subscription", legacyMethod: "marketplace", wantMethod: redis.String("marketplace")},
		{name: "legacy subscription without a method"},
		{name: "new subscription defaults to credit card", create: true, wantMethod: redis.String("credit-card")},
	} {
		t.Run(test.name, func(t *testing.T) {
			fake := testhelpers.NewFakeAPI()
			name := "original"
			fake.Handle("POST /fixed/subscriptions", func(*http.Request) (any, error) {
				return fake.RegisterTask("create-fixed-subscription", 42), nil
			})
			fake.Handle("GET /fixed/subscriptions/42", func(*http.Request) (any, error) {
				return fixedSubscriptions.FixedSubscriptionResponse{
					Name: redis.String(name), Status: redis.String("active"), PlanId: redis.Int(2),
				}, nil
			})
			var updates []fixedSubscriptions.FixedSubscriptionRequest
			fake.Handle("PUT /fixed/subscriptions/42", func(req *http.Request) (any, error) {
				var update fixedSubscriptions.FixedSubscriptionRequest
				if err := json.NewDecoder(req.Body).Decode(&update); err != nil {
					return nil, err
				}
				updates = append(updates, update)
				name = redis.StringValue(update.Name)
				return fake.RegisterTask("update-fixed-subscription", 42), nil
			})
			server := newEssentialsSubscriptionTestServer(t, fake)
			plan := essentialssubscription.EssentialsSubscriptionResourceModel{
				ID:              types.StringValue("42"),
				Name:            types.StringValue(name),
				Status:          types.StringValue("active"),
				PlanID:          types.Int64Value(2),
				PaymentMethod:   types.StringValue(test.legacyMethod),
				PaymentMethodID: types.Int64Value(0),
				CreationDate:    types.StringValue(redis.TimeValue(nil).String()),
			}
			prior := essentialsSubscriptionTestPlan(t, plan)
			var private []byte
			if test.create {
				plan.ID = types.StringUnknown()
				plan.PaymentMethod = types.StringUnknown()
				created := applyEssentialsSubscriptionTestChange(t, server, nil, plan, nil)
				require.Empty(t, created.Diagnostics)
				plan = readEssentialsSubscriptionTestState(t, created.NewState)
				prior, private = created.NewState, created.Private
			}

			paymentMethodEdit := "marketplace"
			if test.legacyMethod == "marketplace" {
				paymentMethodEdit = "credit-card"
			}
			plan.PaymentMethod = types.StringValue(paymentMethodEdit)
			ignored := applyEssentialsSubscriptionTestChange(t, server, prior, plan, private)
			require.Empty(t, ignored.Diagnostics)
			afterEdit := readEssentialsSubscriptionTestState(t, ignored.NewState)
			require.Equal(t, plan.PaymentMethod, afterEdit.PaymentMethod)
			fake.WithHandlersPaused(func() { require.Empty(t, updates) })

			afterEdit.Name = types.StringValue("renamed")
			renamed := applyEssentialsSubscriptionTestChange(t, server, ignored.NewState, afterEdit, ignored.Private)
			require.Empty(t, renamed.Diagnostics)
			require.Equal(t, afterEdit.PaymentMethod, readEssentialsSubscriptionTestState(t, renamed.NewState).PaymentMethod)
			fake.WithHandlersPaused(func() {
				require.Len(t, updates, 1)
				require.Equal(t, "renamed", redis.StringValue(updates[0].Name))
				require.Equal(t, test.wantMethod, updates[0].PaymentMethod)
			})
		})
	}
}

func newEssentialsSubscriptionTestServer(t *testing.T, fake *testhelpers.FakeAPI) tfprotov5.ProviderServer {
	t.Helper()
	factories := testhelpers.FrameworkProviderFactories(testhelpers.NewAPIClient(t, fake), essentialssubscription.NewEssentialsSubscriptionResource)
	server, err := factories[testhelpers.FrameworkTestProviderTypeName]()
	require.NoError(t, err)
	response, err := server.ConfigureProvider(t.Context(), &tfprotov5.ConfigureProviderRequest{})
	require.NoError(t, err)
	require.Empty(t, response.Diagnostics)
	return server
}

func applyEssentialsSubscriptionTestChange(t *testing.T, server tfprotov5.ProviderServer, prior *tfprotov5.DynamicValue, plan essentialssubscription.EssentialsSubscriptionResourceModel, private []byte) *tfprotov5.ApplyResourceChangeResponse {
	t.Helper()
	result, err := server.ApplyResourceChange(t.Context(), &tfprotov5.ApplyResourceChangeRequest{
		TypeName:       testhelpers.FrameworkTestProviderTypeName + "_essentials_subscription",
		PriorState:     prior,
		PlannedState:   essentialsSubscriptionTestPlan(t, plan),
		PlannedPrivate: private,
	})
	require.NoError(t, err)
	return result
}

func essentialsSubscriptionTestPlan(t *testing.T, model essentialssubscription.EssentialsSubscriptionResourceModel) *tfprotov5.DynamicValue {
	t.Helper()
	s := essentialsSubscriptionTestSchema(t)
	model.Timeouts = timeouts.Value{Object: types.ObjectNull(s.Blocks["timeouts"].Type().(timeouts.Type).AttrTypes)}
	plan := tfsdk.Plan{Schema: s}
	diags := plan.Set(t.Context(), model)
	require.False(t, diags.HasError(), "%v", diags)
	value, err := tfprotov5.NewDynamicValue(plan.Raw.Type(), plan.Raw)
	require.NoError(t, err)
	return &value
}

func readEssentialsSubscriptionTestState(t *testing.T, value *tfprotov5.DynamicValue) essentialssubscription.EssentialsSubscriptionResourceModel {
	t.Helper()
	s := essentialsSubscriptionTestSchema(t)
	raw, err := value.Unmarshal(s.Type().TerraformType(t.Context()))
	require.NoError(t, err)
	require.False(t, raw.IsNull())
	require.True(t, raw.IsFullyKnown())
	state := tfsdk.State{Schema: s, Raw: raw}
	var model essentialssubscription.EssentialsSubscriptionResourceModel
	diags := state.Get(t.Context(), &model)
	require.False(t, diags.HasError(), "%v", diags)
	return model
}

func essentialsSubscriptionTestSchema(t *testing.T) schema.Schema {
	t.Helper()
	var response resource.SchemaResponse
	essentialssubscription.NewEssentialsSubscriptionResource().Schema(t.Context(), resource.SchemaRequest{}, &response)
	require.False(t, response.Diagnostics.HasError())
	return response.Schema
}
