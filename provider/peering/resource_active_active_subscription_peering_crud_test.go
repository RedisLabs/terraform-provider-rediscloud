package peering

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func activeActivePeeringPlanModel(s resource.SchemaResponse) ActiveActiveSubscriptionPeeringResourceModel {
	unknown := types.StringUnknown()
	return ActiveActiveSubscriptionPeeringResourceModel{
		ID: unknown, SubscriptionID: types.StringValue("1"), SourceRegion: unknown, DestinationRegion: unknown, AWSAccountID: unknown,
		VPCID: unknown, VPCCidr: unknown, VPCCidrs: types.SetUnknown(types.StringType), GCPProjectID: unknown, GCPNetworkName: unknown,
		Status: unknown, AWSPeeringID: unknown, GCPRedisProjectID: unknown, GCPRedisNetworkName: unknown, GCPPeeringID: unknown,
		Timeouts: nullTimeouts(s),
	}
}

func TestActiveActiveSubscriptionPeeringStateMatchesSDK(t *testing.T) {
	ctx := context.Background()

	for name, tc := range map[string]struct {
		region string
		plan   func(m *ActiveActiveSubscriptionPeeringResourceModel)
		null   []string
	}{
		"aws": {
			region: `{"region":"us-east-1","vpcPeerings":[{"id":2,"status":"pending-acceptance","regionName":"eu-west-1","awsAccountId":"123456789012","awsPeeringUid":"pcx-1","vpcUid":"vpc-1","vpcCidr":"10.0.0.0/16"}]}`,
			plan: func(m *ActiveActiveSubscriptionPeeringResourceModel) {
				m.ProviderName = types.StringValue(providerAWS)
				m.SourceRegion = types.StringValue("us-east-1")
				m.DestinationRegion = types.StringValue("eu-west-1")
				m.AWSAccountID = types.StringValue("123456789012")
				m.VPCID = types.StringValue("vpc-1")
				m.VPCCidr = types.StringValue("10.0.0.0/16")
			},
			null: []string{"gcp_project_id", "gcp_network_name", "gcp_redis_project_id", "gcp_redis_network_name", "gcp_peering_id"},
		},
		"gcp": {
			region: `{"region":"europe-west2","vpcPeerings":[{"id":2,"status":"inactive","vpcProjectUid":"project","vpcNetworkName":"network","redisProjectUid":"redis-project","redisNetworkName":"redis-network","cloudPeeringId":"peering"}]}`,
			plan: func(m *ActiveActiveSubscriptionPeeringResourceModel) {
				m.ProviderName = types.StringValue(providerGCP)
				m.SourceRegion = types.StringValue("europe-west2")
				m.GCPProjectID = types.StringValue("project")
				m.GCPNetworkName = types.StringValue("network")
			},
			null: []string{"destination_region", "aws_account_id", "vpc_id", "vpc_cidr", "aws_peering_id"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			r := &activeActiveSubscriptionPeeringResource{client: newFakeAPIClient(t, map[string]string{
				"POST /subscriptions/1/regions/peerings/": `{"taskId":"create"}`,
				"GET /tasks/create":                       `{"status":"processing-completed","response":{"resourceId":2}}`,
				"GET /subscriptions/1/regions/peerings/":  `{"taskId":"list"}`,
				"GET /tasks/list":                         `{"status":"processing-completed","response":{"resource":{"subscriptionId":1,"regions":[` + tc.region + `]}}}`,
			})}
			var s resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &s)

			model := activeActivePeeringPlanModel(s)
			tc.plan(&model)
			plan := tfsdk.Plan{Schema: s.Schema}
			require.False(t, plan.Set(ctx, &model).HasError())

			created := resource.CreateResponse{State: tfsdk.State{Schema: s.Schema}}
			r.Create(ctx, resource.CreateRequest{Plan: plan}, &created)
			require.False(t, created.Diagnostics.HasError(), created.Diagnostics)

			assertStateKeepsPlannedValues(t, plan, created.State)
			assertNullAttributes(t, created.State, tc.null)

			imported := importAndRead(t, ctx, r, s, "1/2")
			assert.True(t, created.State.Raw.Equal(imported.Raw), "imported state must match the created state")
		})
	}
}

func TestActiveActiveSubscriptionPeeringModifyPlanRejectsGCPChanges(t *testing.T) {
	ctx := context.Background()
	r := &activeActiveSubscriptionPeeringResource{}
	var s resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &s)

	prior := activeActivePeeringPlanModel(s)
	prior.ID = types.StringValue("1/2")
	prior.ProviderName = types.StringValue(providerGCP)
	prior.SourceRegion = types.StringValue("europe-west2")
	prior.GCPProjectID = types.StringValue("project")
	prior.GCPNetworkName = types.StringValue("network")
	prior.nullUnknowns()
	state := tfsdk.State{Schema: s.Schema}
	require.False(t, state.Set(ctx, &prior).HasError())

	for name, tc := range map[string]struct {
		projectID, networkName types.String
		errorPaths             []path.Path
	}{
		"unchanged":       {types.StringValue("project"), types.StringValue("network"), nil},
		"removed":         {types.StringNull(), types.StringNull(), nil},
		"unknown":         {types.StringUnknown(), types.StringUnknown(), nil},
		"project changed": {types.StringValue("other"), types.StringValue("network"), []path.Path{path.Root("gcp_project_id")}},
		"network changed": {types.StringValue("project"), types.StringValue("other"), []path.Path{path.Root("gcp_network_name")}},
		"both changed":    {types.StringValue("other"), types.StringValue("other"), []path.Path{path.Root("gcp_project_id"), path.Root("gcp_network_name")}},
	} {
		t.Run(name, func(t *testing.T) {
			configModel := prior
			configModel.GCPProjectID = tc.projectID
			configModel.GCPNetworkName = tc.networkName
			config := configFrom(t, ctx, s, &configModel)
			plan := tfsdk.Plan{Schema: s.Schema, Raw: state.Raw}

			resp := resource.ModifyPlanResponse{Plan: plan}
			r.ModifyPlan(ctx, resource.ModifyPlanRequest{Config: config, Plan: plan, State: state}, &resp)

			var got []path.Path
			for _, d := range resp.Diagnostics.Errors() {
				got = append(got, d.(interface{ Path() path.Path }).Path())
			}
			assert.Equal(t, tc.errorPaths, got)
		})
	}

	t.Run("create", func(t *testing.T) {
		configModel := prior
		config := configFrom(t, ctx, s, &configModel)
		plan := tfsdk.Plan{Schema: s.Schema, Raw: state.Raw}
		empty := tfsdk.State{Schema: s.Schema, Raw: state.Raw.Copy()}
		empty.RemoveResource(ctx)

		resp := resource.ModifyPlanResponse{Plan: plan}
		r.ModifyPlan(ctx, resource.ModifyPlanRequest{Config: config, Plan: plan, State: empty}, &resp)
		assert.False(t, resp.Diagnostics.HasError())
	})
}

func configFrom(t *testing.T, ctx context.Context, s resource.SchemaResponse, model *ActiveActiveSubscriptionPeeringResourceModel) tfsdk.Config {
	t.Helper()
	values := tfsdk.State{Schema: s.Schema}
	require.False(t, values.Set(ctx, model).HasError())
	return tfsdk.Config{Schema: s.Schema, Raw: values.Raw}
}
