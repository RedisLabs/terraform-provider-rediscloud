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

func TestSubscriptionPeeringStateMatchesSDK(t *testing.T) {
	ctx := context.Background()
	unknown := types.StringUnknown()

	for name, tc := range map[string]struct {
		peering string
		plan    func(m *SubscriptionPeeringResourceModel)
		null    []string
	}{
		"aws": {
			peering: `{"vpcPeeringId":2,"status":"pending-acceptance","awsAccountId":"123456789012","awsPeeringUid":"pcx-1","vpcUid":"vpc-1","vpcCidr":"10.0.0.0/16","regionName":"eu-west-1"}`,
			plan: func(m *SubscriptionPeeringResourceModel) {
				m.ProviderName = types.StringValue(providerAWS)
				m.Region = types.StringValue("eu-west-1")
				m.AWSAccountID = types.StringValue("123456789012")
				m.VPCID = types.StringValue("vpc-1")
				m.VPCCidr = types.StringValue("10.0.0.0/16")
			},
			null: []string{"gcp_project_id", "gcp_network_name", "gcp_redis_project_id", "gcp_redis_network_name", "gcp_peering_id"},
		},
		"gcp": {
			peering: `{"vpcPeeringId":2,"status":"inactive","projectUid":"project","networkName":"network","redisProjectUid":"redis-project","redisNetworkName":"redis-network","cloudPeeringId":"peering"}`,
			plan: func(m *SubscriptionPeeringResourceModel) {
				m.ProviderName = types.StringValue(providerGCP)
				m.GCPProjectID = types.StringValue("project")
				m.GCPNetworkName = types.StringValue("network")
			},
			null: []string{"region", "aws_account_id", "vpc_id", "vpc_cidr", "aws_peering_id"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			r := &subscriptionPeeringResource{client: newFakeAPIClient(t, map[string]string{
				"POST /subscriptions/1/peerings": `{"taskId":"create"}`,
				"GET /tasks/create":              `{"status":"processing-completed","response":{"resourceId":2}}`,
				"GET /subscriptions/1/peerings":  `{"taskId":"list"}`,
				"GET /tasks/list":                `{"status":"processing-completed","response":{"resource":{"peerings":[` + tc.peering + `]}}}`,
			})}
			var s resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &s)

			model := SubscriptionPeeringResourceModel{
				ID: unknown, SubscriptionID: types.StringValue("1"), Region: unknown, AWSAccountID: unknown, VPCID: unknown,
				VPCCidr: unknown, VPCCidrs: types.SetUnknown(types.StringType), GCPProjectID: unknown, GCPNetworkName: unknown,
				Status: unknown, AWSPeeringID: unknown, GCPRedisProjectID: unknown, GCPRedisNetworkName: unknown, GCPPeeringID: unknown,
				Timeouts: nullTimeouts(s),
			}
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
