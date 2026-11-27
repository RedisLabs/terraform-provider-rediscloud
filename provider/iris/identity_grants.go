package iris

import (
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"

	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Request grants deliberately have no constraints field. Tenant is the tenant of
// the whole key, repeated on each grant because that is where the API carries it.
type identityGrant struct {
	Tenant       *string  `json:"tenant,omitempty"`
	Product      string   `json:"product"`
	ResourceType string   `json:"resourceType"`
	ResourceID   string   `json:"resourceId"`
	Actions      []string `json:"actions"`
}
type identityGrantResponse struct {
	Product      string   `json:"product"`
	ResourceType string   `json:"resourceType"`
	ResourceID   string   `json:"resourceId"`
	Actions      []string `json:"actions"`
	// Detect unsupported narrowing policies on managed keys rather than silently dropping them.
	Constraints json.RawMessage `json:"constraints"`
}

// grantsToAPI converts grants to requests. Iris requires every grant of a key to
// share the key's tenant, so the one key-level tenant is stamped on all of them;
// a nil tenant lets Iris apply its installation tenant.
func grantsToAPI(ctx context.Context, v types.Set, tenant *string) ([]identityGrant, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	if v.IsNull() || v.IsUnknown() {
		diagnostics.AddError("Missing API key grants", "At least one fully known grant block is required.")
		return nil, diagnostics
	}
	var models []IdentityGrantModel
	d := v.ElementsAs(ctx, &models, false)
	diagnostics.Append(d...)
	if diagnostics.HasError() {
		return nil, diagnostics
	}
	if len(models) == 0 {
		diagnostics.AddError("Missing API key grants", "At least one grant block is required.")
		return nil, diagnostics
	}
	grants := make([]identityGrant, 0, len(models))
	for _, m := range models {
		var actions []string
		diagnostics.Append(m.Actions.ElementsAs(ctx, &actions, false)...)
		grants = append(grants, identityGrant{Tenant: tenant, Product: m.Product.ValueString(), ResourceType: m.ResourceType.ValueString(), ResourceID: m.ResourceID.ValueString(), Actions: actions})
	}
	return grants, diagnostics
}
func grantsFromAPI(ctx context.Context, grants []identityGrantResponse) (types.Set, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	models := make([]IdentityGrantModel, 0, len(grants))
	for _, grant := range grants {
		// An unsupported constrained key must never be rewritten as an unconstrained one.
		if hasGrantConstraints([]identityGrantResponse{grant}) {
			diagnostics.AddWarning("API key has service-managed constraints", "Constraint details are not exposed by this resource. Refresh, metadata updates and deletion remain available; grant-policy updates are blocked to preserve these restrictions.")
		}
		actions, d := types.SetValueFrom(ctx, types.StringType, grant.Actions)
		diagnostics.Append(d...)
		if diagnostics.HasError() {
			return types.Set{}, diagnostics
		}
		models = append(models, IdentityGrantModel{Product: types.StringValue(grant.Product), ResourceType: types.StringValue(grant.ResourceType), ResourceID: types.StringValue(grant.ResourceID), Actions: actions})
	}
	value, d := types.SetValueFrom(ctx, grantElementType, models)
	diagnostics.Append(d...)
	return value, diagnostics
}

func hasGrantConstraints(grants []identityGrantResponse) bool {
	for _, grant := range grants {
		var constraints map[string]json.RawMessage
		if len(grant.Constraints) > 0 && (json.Unmarshal(grant.Constraints, &constraints) != nil || len(constraints) > 0) {
			return true
		}
	}
	return false
}

type IdentityGrantModel struct {
	Product      types.String `tfsdk:"product"`
	ResourceType types.String `tfsdk:"resource_type"`
	ResourceID   types.String `tfsdk:"resource_id"`
	Actions      types.Set    `tfsdk:"actions"`
}

var grantElementType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(IdentityGrantModel{Actions: types.SetNull(types.StringType)})}
