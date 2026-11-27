package iris

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

const tenantDescription = "Tenant that owns the key and all of its grants; Iris allows one tenant per key and requires every store in a grant to belong to it. Omit it to use the service default at creation. Changing it replaces the key, as does a value that is only known at apply time, because Iris cannot move a key between tenants. Case is preserved; surrounding whitespace is not allowed."

func IdentityAPIKeyResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Manages an API key and its permission grants through the direct Iris Identity Service API. The generated token is returned only at creation and retained in Terraform state through refresh and updates. To replace a token, create another key resource, update consumers, then delete the old resource. Import is not supported because existing tokens cannot be recovered. Grant and action order does not matter. Constraint configuration is not supported. Keys with service-managed constraints can be refreshed, renamed and deleted with a warning. Grant-policy updates are blocked while constraints are present so Terraform cannot discard those restrictions.",
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
			"grant": schema.SetNestedBlock{
				Description: "Permissions granted to the key. At least one grant is required; constraints are not supported.",
				Validators:  []validator.Set{setvalidator.SizeAtLeast(1)},
				NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
					"product":       schema.StringAttribute{Required: true, Description: "Lowercase Iris product without surrounding whitespace, for example memory.", Validators: []validator.String{stringvalidator.LengthAtLeast(1), canonicalGrantNameValidator{}}},
					"resource_type": schema.StringAttribute{Required: true, Description: "Lowercase product resource type without surrounding whitespace. Use mem-store for Agent Memory.", Validators: []validator.String{stringvalidator.LengthAtLeast(1), canonicalGrantNameValidator{}}},
					"resource_id":   schema.StringAttribute{Required: true, Description: "The exact resource identifier.", Validators: []validator.String{stringvalidator.LengthAtLeast(1)}},
					"actions":       schema.SetAttribute{Required: true, ElementType: types.StringType, Description: "Allowed actions for this resource.", Validators: []validator.Set{setvalidator.SizeAtLeast(1), setvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1))}},
				}},
			},
		},
		Attributes: map[string]schema.Attribute{
			"created_at": schema.Int64Attribute{
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Computed:      true,
			},
			"expires_at": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
				Description:         "Future Unix timestamp at which Iris deletes the key; 0 or omitted means never. After expiry, configure a new future timestamp or 0 before recreating the key; Iris rejects a past timestamp at creation, and changing the expiry of an existing key to a past timestamp is rejected during planning. Fixed past timestamps are never extended automatically.",
				MarkdownDescription: "Future Unix timestamp at which Iris deletes the key; 0 or omitted means never. After expiry, configure a new future timestamp or 0 before recreating the key; Iris rejects a past timestamp at creation, and changing the expiry of an existing key to a past timestamp is rejected during planning. Fixed past timestamps are never extended automatically.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Identity Service key identifier.",
				MarkdownDescription: "Identity Service key identifier.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"name": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"tenant": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         tenantDescription,
				MarkdownDescription: tenantDescription,
				// Without a configured value keep the tenant Iris assigned. A value that is
				// only known at apply, or that differs from the state, replaces the key:
				// Iris cannot move a key between tenants.
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown(), stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1), tenantValidator{}},
			},
			"token": schema.StringAttribute{
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Computed:      true,
				Sensitive:     true,
				Description:   "Generated API key token, available only at creation and retained in Terraform state.",
			},
			"user_id": schema.StringAttribute{
				Optional:            true,
				Description:         "The user that the key acts as. Optional: a key without a user acts as no user. Changing or removing the user replaces the key. Requires API key user binding to be enabled in the deployment. Administrators can set any user; any other caller can set only their own user ID.",
				MarkdownDescription: "The user that the key acts as. Optional: a key without a user acts as no user. Changing or removing the user replaces the key. Requires API key user binding to be enabled in the deployment. Administrators can set any user; any other caller can set only their own user ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
		},
	}
}

type IdentityAPIKeyModel struct {
	Timeouts  timeouts.Value `tfsdk:"timeouts"`
	CreatedAt types.Int64    `tfsdk:"created_at"`
	ExpiresAt types.Int64    `tfsdk:"expires_at"`
	Grants    types.Set      `tfsdk:"grant"`
	ID        types.String   `tfsdk:"id"`
	Name      types.String   `tfsdk:"name"`
	Tenant    types.String   `tfsdk:"tenant"`
	Token     types.String   `tfsdk:"token"`
	UserID    types.String   `tfsdk:"user_id"`
}
