package iris

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

type identityAPIKeyCreateRequest struct {
	ExpiresAt *int64          `json:"expiresAt,omitempty"`
	Grants    []identityGrant `json:"grants,omitempty"`
	Name      *string         `json:"name,omitempty"`
	UserID    *string         `json:"userId,omitempty"`
}
type identityAPIKeyUpdateRequest struct {
	ExpiresAt *int64          `json:"expiresAt,omitempty"`
	Grants    []identityGrant `json:"grants,omitempty"`
	Name      *string         `json:"name,omitempty"`
}
type identityAPIKeyResponse struct {
	CreatedAt *int64                   `json:"createdAt"`
	ExpiresAt *int64                   `json:"expiresAt"`
	Grants    *[]identityGrantResponse `json:"grants"`
	KeyID     *string                  `json:"keyId"`
	Name      *string                  `json:"name"`
	TenantID  *string                  `json:"tenantId"`
	Token     *string                  `json:"token"`
	UserID    *string                  `json:"userId"`
}
type identityAPIKeyResource struct{ client *client.IrisServiceClient }

var (
	_ resource.Resource              = (*identityAPIKeyResource)(nil)
	_ resource.ResourceWithConfigure = (*identityAPIKeyResource)(nil)
)

func NewIdentityAPIKeyResource() resource.Resource { return &identityAPIKeyResource{} }
func (r *identityAPIKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iris_identity_api_key"
}
func (r *identityAPIKeyResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = IdentityAPIKeyResourceSchema(ctx)
}
func (r *identityAPIKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configuredClient(req, "identity", &resp.Diagnostics)
}
func (r *identityAPIKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data, config IdentityAPIKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	ctx, cancel := operationContext(ctx, data.Timeouts.Create, defaultOperationTimeout, &resp.Diagnostics)
	defer cancel()
	if resp.Diagnostics.HasError() {
		return
	}

	grants, grantDiags := grantsToAPI(ctx, config.Grants, optionalString(config.Tenant))
	resp.Diagnostics.Append(grantDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	body := identityAPIKeyCreateRequest{
		ExpiresAt: optionalInt64(data.ExpiresAt),
		Grants:    grants,
		Name:      optionalString(config.Name),
		UserID:    optionalString(config.UserID),
	}
	var created identityAPIKeyResponse
	if err := r.client.Do(ctx, http.MethodPost, "/v1/api-keys", nil, body, &created); err != nil {
		resp.Diagnostics.AddError("Failed to create Iris resource", err.Error())
		return
	}
	if created.KeyID == nil || *created.KeyID == "" {
		resp.Diagnostics.AddError("Missing Iris resource identifier", "The resource may have been created, but the response omitted its identifier. Inspect the service before retrying.")
		return
	}
	data.ID = types.StringPointerValue(created.KeyID)
	data.Token = types.StringPointerValue(created.Token)
	data.CreatedAt = types.Int64PointerValue(created.CreatedAt)
	resp.Diagnostics.Append(r.refresh(ctx, &resp.State, &data, "created")...)
}
func (r *identityAPIKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data IdentityAPIKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	ctx, cancel := operationContext(ctx, data.Timeouts.Read, defaultReadTimeout, &resp.Diagnostics)
	defer cancel()
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := r.read(ctx, &data)
	if client.IrisNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Iris resource", err.Error())
		return
	}
	resp.Diagnostics.Append(data.setFromAPI(ctx, current)...)
	resp.Diagnostics.Append(saveState(ctx, &resp.State, &data)...)
}
func (r *identityAPIKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, config IdentityAPIKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	var previous IdentityAPIKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &previous)...)
	ctx, cancel := operationContext(ctx, data.Timeouts.Update, defaultOperationTimeout, &resp.Diagnostics)
	defer cancel()
	if resp.Diagnostics.HasError() {
		return
	}

	var grants []identityGrant
	if !data.Grants.Equal(previous.Grants) {
		current, err := r.read(ctx, &previous)
		if err != nil {
			resp.Diagnostics.AddError("Unable to verify existing grant policy", err.Error())
			return
		}
		if current.Grants == nil {
			resp.Diagnostics.AddError("Unable to verify existing grant policy", "The read response omitted grants; no policy update was sent.")
			return
		}
		if hasGrantConstraints(*current.Grants) {
			resp.Diagnostics.AddError("Cannot overwrite constrained grants", "This key has service-managed constraints that Terraform cannot represent. Metadata updates and deletion are allowed, but grant changes must be made through the owning service.")
			return
		}

		var d diag.Diagnostics
		// A tenant change replaces the key, so an update always keeps the stored tenant.
		grants, d = grantsToAPI(ctx, data.Grants, optionalString(previous.Tenant))
		resp.Diagnostics.Append(d...)
	}
	if resp.Diagnostics.HasError() {
		return
	}
	body := identityAPIKeyUpdateRequest{
		ExpiresAt: changedInt64(data.ExpiresAt, previous.ExpiresAt),
		Grants:    grants,
		Name:      changedString(config.Name, previous.Name),
	}
	if body.Name == nil && body.ExpiresAt == nil && body.Grants == nil {
		previous.Timeouts = data.Timeouts
		resp.Diagnostics.Append(saveState(ctx, &resp.State, &previous)...)
		return
	}
	if err := r.client.Do(ctx, "PATCH", "/v1/api-keys/"+escaped(data.ID), nil, body, nil); err != nil {
		resp.Diagnostics.AddError("Failed to update Iris resource", err.Error())
		return
	}
	resp.Diagnostics.Append(r.refresh(ctx, &resp.State, &data, "updated")...)
}
func (r *identityAPIKeyResource) refresh(ctx context.Context, state *tfsdk.State, data *IdentityAPIKeyModel, action string) diag.Diagnostics {
	return readAfterWrite(ctx, state, data, action, false,
		func() (*identityAPIKeyResponse, error) { return r.read(ctx, data) },
		func(current *identityAPIKeyResponse) diag.Diagnostics { return data.setFromAPI(ctx, current) })
}

func (r *identityAPIKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data IdentityAPIKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	ctx, cancel := operationContext(ctx, data.Timeouts.Delete, defaultOperationTimeout, &resp.Diagnostics)
	defer cancel()
	if resp.Diagnostics.HasError() {
		return
	}

	query := url.Values{}
	if err := r.client.Do(ctx, http.MethodDelete, "/v1/api-keys/"+escaped(data.ID), query, nil, nil); err != nil && !client.IrisNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete Iris resource", err.Error())
	}
}
func (r *identityAPIKeyResource) read(ctx context.Context, data *IdentityAPIKeyModel) (*identityAPIKeyResponse, error) {
	var current identityAPIKeyResponse
	if err := r.client.Do(ctx, http.MethodGet, "/v1/api-keys/"+escaped(data.ID), nil, nil, &current); err != nil {
		return nil, err
	}
	if err := verifyID(data.ID, current.KeyID); err != nil {
		return nil, err
	}
	return &current, nil
}
func (m *IdentityAPIKeyModel) setFromAPI(ctx context.Context, current *identityAPIKeyResponse) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	m.CreatedAt = types.Int64PointerValue(current.CreatedAt)
	if current.ExpiresAt != nil {
		m.ExpiresAt = types.Int64PointerValue(current.ExpiresAt)
	} else {
		m.ExpiresAt = types.Int64Value(0)
	}
	if current.KeyID != nil {
		m.ID = types.StringPointerValue(current.KeyID)
	}
	m.Name = types.StringPointerValue(current.Name)
	m.Tenant = types.StringPointerValue(current.TenantID)
	m.UserID = types.StringPointerValue(current.UserID)
	var remote []identityGrantResponse
	if current.Grants != nil {
		remote = *current.Grants
	}
	grants, d := grantsFromAPI(ctx, remote)
	diagnostics.Append(d...)
	if !diagnostics.HasError() {
		m.Grants = grants
	}
	return diagnostics
}

var _ resource.ResourceWithModifyPlan = (*identityAPIKeyResource)(nil)

func (*identityAPIKeyResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// A new key needs no plan-time checks. Iris rejects a past expiry on create
	// before anything exists, whereas failing here would block the whole
	// configuration once a fixed expires_at passes and the key leaves state.
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}
	var planned, previous IdentityAPIKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planned)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &previous)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Changing the expiry, or replacing the key through its user, must not end
	// with an expiry Iris rejects after the old key has already been deleted.
	changingExpiry := !planned.ExpiresAt.Equal(previous.ExpiresAt) || !planned.UserID.Equal(previous.UserID)
	if value := planned.ExpiresAt.ValueInt64(); changingExpiry && !planned.ExpiresAt.IsUnknown() && value != 0 && value <= time.Now().Unix() {
		resp.Diagnostics.AddAttributeError(path.Root("expires_at"), "Invalid API key expiration", "expires_at must be a future Unix timestamp or 0 for a key that never expires.")
	}
	// A new tenant, or one only known at apply, replaces the key. Terraform deletes
	// the old key first, and Iris rejects the new one unless every granted store
	// belongs to the new tenant, which the provider cannot check.
	if !planned.Tenant.Equal(previous.Tenant) {
		resp.Diagnostics.AddAttributeWarning(path.Root("tenant"), "Changing the tenant replaces the key", "Terraform deletes the current key before it creates the new one, and Iris rejects the new key unless every store in its grants belongs to the new tenant, which would leave no key. Change the tenant together with the grants, and consider `lifecycle { create_before_destroy = true }` so a rejected replacement keeps the current key.")
	}
}
