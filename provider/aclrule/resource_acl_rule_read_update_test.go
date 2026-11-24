package aclrule_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"testing/synctest"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/aclrule"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAclRuleReadAndUpdateAfterMissingListEntry(t *testing.T) {
	active := json.RawMessage(`{"redisRules":[{"id":42,"status":"active","name":"test-rule","acl":"+@all"}]}`)
	missing := json.RawMessage(`{"redisRules":[{"id":77,"status":"active","name":"another-rule","acl":"+@read"}]}`)

	for _, operation := range []string{"read", "update"} {
		for _, scenario := range []string{"reappears", "stays absent", "permission denied"} {
			t.Run(operation+"/"+scenario, func(t *testing.T) {
				fake := testhelpers.NewFakeAPI()
				refreshes, writes := 0, 0
				var payload map[string]string
				fake.Handle("PUT /acl/redisRules/42", func(req *http.Request) (any, error) {
					writes++
					if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
						return nil, err
					}
					return fake.RegisterTask("updateRedisRule", 42), nil
				})
				waitingForActive := operation == "update"
				fake.Handle("GET /acl/redisRules", func(*http.Request) (any, error) {
					if waitingForActive {
						waitingForActive = false
						return active, nil
					}
					refreshes++
					switch scenario {
					case "permission denied":
						return nil, testhelpers.StatusError{Code: http.StatusForbidden, Body: map[string]string{"description": "access denied"}}
					case "reappears":
						if refreshes > 1 {
							return active, nil
						}
					}
					return missing, nil
				})

				ctx := t.Context()
				r := aclrule.NewAclRuleResource()
				var configured resource.ConfigureResponse
				r.(resource.ResourceWithConfigure).Configure(ctx, resource.ConfigureRequest{ProviderData: testhelpers.NewAPIClient(t, fake)}, &configured)
				require.Empty(t, configured.Diagnostics)
				var schema resource.SchemaResponse
				r.Schema(ctx, resource.SchemaRequest{}, &schema)
				require.Empty(t, schema.Diagnostics)
				before := aclrule.AclRuleResourceModel{
					ID:   types.StringValue("42"),
					Name: types.StringValue("test-rule"),
					Rule: types.StringValue("+@read"),
					Timeouts: timeouts.Value{Object: types.ObjectValueMust(schema.Schema.Blocks["timeouts"].Type().(timeouts.Type).AttrTypes, map[string]attr.Value{
						"create": types.StringNull(), "read": types.StringValue("2s"), "update": types.StringValue("2s"), "delete": types.StringNull(),
					})},
				}
				prior := tfsdk.State{Schema: schema.Schema}
				require.Empty(t, prior.Set(ctx, &before))
				after := before
				after.Rule = types.StringValue("+@all")
				result := prior
				var diagnostics diag.Diagnostics
				var elapsed time.Duration
				synctest.Test(t, func(t *testing.T) {
					ctx := t.Context()
					start := time.Now()
					defer func() { elapsed = time.Since(start) }()
					if operation == "read" {
						response := resource.ReadResponse{State: prior}
						r.Read(ctx, resource.ReadRequest{State: prior}, &response)
						result, diagnostics = response.State, response.Diagnostics
						return
					}
					plan := tfsdk.Plan{Schema: schema.Schema}
					require.Empty(t, plan.Set(ctx, &after))
					response := resource.UpdateResponse{State: prior}
					r.Update(ctx, resource.UpdateRequest{State: prior, Plan: plan}, &response)
					result, diagnostics = response.State, response.Diagnostics
				})

				if operation == "update" {
					require.Equal(t, 1, writes)
					require.Equal(t, map[string]string{"redisRule": "+@all"}, payload)
				} else {
					require.Zero(t, writes)
				}
				want := before
				switch scenario {
				case "reappears":
					require.Empty(t, diagnostics)
					require.Equal(t, 2, refreshes)
					want = after
				case "stays absent":
					require.Len(t, diagnostics.Errors(), 1, "persistent list omission must be reported")
					require.Equal(t, "Failed to read ACL rule", diagnostics.Errors()[0].Summary())
					require.Contains(t, diagnostics.Errors()[0].Detail(), "42")
					require.Greater(t, refreshes, 1)
					require.LessOrEqual(t, elapsed, 2*time.Second)
				case "permission denied":
					require.Len(t, diagnostics.Errors(), 1)
					require.Contains(t, diagnostics.Errors()[0].Detail(), "403")
					require.Equal(t, 1, refreshes)
				}
				require.False(t, result.Raw.IsNull(), "a missing list entry must not release ownership of ID 42")
				var got aclrule.AclRuleResourceModel
				require.Empty(t, result.Get(ctx, &got))
				require.Equal(t, want, got)
			})
		}
	}
}
