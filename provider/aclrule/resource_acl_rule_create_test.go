package aclrule_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/aclrule"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAclRuleCreateState(t *testing.T) {
	active := map[string]any{"redisRules": []map[string]any{{"id": 42, "status": "active", "name": "test-rule", "acl": "+@all"}}}
	absent := map[string]any{"redisRules": []any{}}
	rejected := testhelpers.StatusError{Code: http.StatusBadRequest, Body: map[string]string{"description": "request rejected"}}

	for _, tc := range []struct {
		name       string
		postError  error
		reads      []any
		diagnostic string
	}{
		{name: "rejected create", postError: rejected, diagnostic: "Failed to create ACL rule"},
		{name: "readiness check fails", reads: []any{rejected}, diagnostic: "ACL rule failed to become active"},
		{name: "rule absent on final read", reads: []any{active, absent}, diagnostic: "ACL rule not found"},
		{name: "final read fails", reads: []any{active, rejected}, diagnostic: "Failed to read ACL rule"},
		{name: "successful create", reads: []any{active, active}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := testhelpers.NewFakeAPI()
			creates, reads := 0, 0
			var payload map[string]string
			fake.Handle("POST /acl/redisRules", func(req *http.Request) (any, error) {
				creates++
				if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
					return nil, err
				}
				if tc.postError != nil {
					return nil, tc.postError
				}
				return fake.RegisterTask("createRedisRule", 42), nil
			})
			fake.Handle("GET /acl/redisRules", func(*http.Request) (any, error) {
				reads++
				if reads > len(tc.reads) {
					return nil, fmt.Errorf("unexpected rule read %d", reads)
				}
				result := tc.reads[reads-1]
				if err, ok := result.(error); ok {
					return nil, err
				}
				return result, nil
			})

			ctx := t.Context()
			r := aclrule.NewAclRuleResource()
			var configured resource.ConfigureResponse
			r.(resource.ResourceWithConfigure).Configure(ctx, resource.ConfigureRequest{ProviderData: testhelpers.NewAPIClient(t, fake)}, &configured)
			require.Empty(t, configured.Diagnostics)
			var schema resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schema)
			require.Empty(t, schema.Diagnostics)
			plan := tfsdk.Plan{Schema: schema.Schema}
			model := aclrule.AclRuleResourceModel{
				ID:       types.StringUnknown(),
				Name:     types.StringValue("test-rule"),
				Rule:     types.StringValue("+@all"),
				Timeouts: timeouts.Value{Object: types.ObjectNull(schema.Schema.Blocks["timeouts"].Type().(timeouts.Type).AttrTypes)},
			}
			require.Empty(t, plan.Set(ctx, &model))
			response := resource.CreateResponse{State: tfsdk.State{Schema: schema.Schema}}
			r.Create(ctx, resource.CreateRequest{Plan: plan}, &response)

			require.Equal(t, 1, creates)
			require.Equal(t, map[string]string{"name": "test-rule", "redisRule": "+@all"}, payload)
			require.Equal(t, len(tc.reads), reads)
			if tc.diagnostic == "" {
				require.Empty(t, response.Diagnostics)
			} else {
				require.Len(t, response.Diagnostics.Errors(), 1)
				require.Equal(t, tc.diagnostic, response.Diagnostics.Errors()[0].Summary())
			}
			if tc.postError != nil {
				require.True(t, response.State.Raw.IsNull())
				return
			}
			require.False(t, response.State.Raw.IsNull(), "create lost state after CAPI returned rule ID 42")
			var state aclrule.AclRuleResourceModel
			require.Empty(t, response.State.Get(ctx, &state))
			model.ID = types.StringValue("42")
			require.Equal(t, model, state)
		})
	}
}
