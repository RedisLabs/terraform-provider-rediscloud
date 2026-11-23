package peering

import (
	"context"
	"testing"

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
		status string
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
			status: "pending-acceptance",
			null:   []string{"gcp_project_id", "gcp_network_name", "gcp_redis_project_id", "gcp_redis_network_name", "gcp_peering_id"},
		},
		"gcp": {
			region: `{"region":"europe-west2","vpcPeerings":[{"id":2,"status":"inactive","vpcProjectUid":"project","vpcNetworkName":"network","redisProjectUid":"redis-project","redisNetworkName":"redis-network","cloudPeeringId":"peering"}]}`,
			plan: func(m *ActiveActiveSubscriptionPeeringResourceModel) {
				m.ProviderName = types.StringValue(providerGCP)
				m.SourceRegion = types.StringValue("europe-west2")
				m.GCPProjectID = types.StringValue("project")
				m.GCPNetworkName = types.StringValue("network")
			},
			status: "inactive",
			null:   []string{"destination_region", "aws_account_id", "vpc_id", "vpc_cidr", "aws_peering_id"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			routes := map[string]string{
				"POST /subscriptions/1/regions/peerings/": `{"taskId":"create"}`,
				"GET /tasks/create":                       `{"status":"processing-completed","response":{"resourceId":2}}`,
				"GET /subscriptions/1/regions/peerings/":  `{"taskId":"list"}`,
				"GET /tasks/list":                         `{"status":"processing-completed","response":{"resource":{"subscriptionId":1,"regions":[` + tc.region + `]}}}`,
			}
			r := &activeActiveSubscriptionPeeringResource{client: newFakeAPIClient(t, routes)}
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

			r.client = peeringAPIClient(t, routes, "GET /subscriptions/1/regions/peerings/")
			partial := resource.CreateResponse{State: tfsdk.State{Schema: s.Schema}}
			r.Create(ctx, resource.CreateRequest{Plan: plan}, &partial)
			require.True(t, partial.Diagnostics.HasError(), partial.Diagnostics)
			assertAcknowledgedPeeringState(t, ctx, partial.State, tc.status)
		})
	}
}

func TestActiveActiveSubscriptionPeeringUpdateRejectsGCPChanges(t *testing.T) {
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
		project, network string
		wantError        bool
	}{
		"unchanged":       {"project", "network", false},
		"project changed": {"other", "network", true},
		"network changed": {"project", "other", true},
	} {
		t.Run(name, func(t *testing.T) {
			model := prior
			model.GCPProjectID = types.StringValue(tc.project)
			model.GCPNetworkName = types.StringValue(tc.network)
			plan := tfsdk.Plan{Schema: s.Schema}
			require.False(t, plan.Set(ctx, &model).HasError())

			resp := resource.UpdateResponse{State: state}
			r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, &resp)
			require.Equal(t, tc.wantError, resp.Diagnostics.HasError(), resp.Diagnostics)
			assert.True(t, resp.State.Raw.Equal(state.Raw))
		})
	}
}
