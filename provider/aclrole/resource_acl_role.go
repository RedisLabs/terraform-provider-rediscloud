package aclrole

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

var (
	_ resource.Resource                 = &aclRoleResource{}
	_ resource.ResourceWithConfigure    = &aclRoleResource{}
	_ resource.ResourceWithImportState  = &aclRoleResource{}
	_ resource.ResourceWithUpgradeState = &aclRoleResource{}
)

type aclRoleResource struct {
	client *client.ApiClient
}

// NewAclRoleResource returns a new resource instance.
func NewAclRoleResource() resource.Resource {
	return &aclRoleResource{}
}

func (r *aclRoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_acl_role"
}

func (r *aclRoleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.ApiClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.ApiClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = c
}

func (r *aclRoleResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create an ACL Role within your Redis Enterprise Cloud Account",
		Version:     1,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the ACL role",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "A meaningful name to identify the role, must be unique",
				Required:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"rule": schema.SetNestedBlock{
				Description: "A set of rules which apply to the role",
				CustomType:  NewRuleSetType(),
				Validators: []validator.Set{
					setvalidator.IsRequired(),
					setvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The name of the rule",
							Required:    true,
						},
					},
					Blocks: map[string]schema.Block{
						"database": schema.SetNestedBlock{
							Description: "A set of databases to whom this rule applies within the role",
							CustomType:  NewRuleDatabaseSetType(),
							Validators: []validator.Set{
								setvalidator.IsRequired(),
								setvalidator.SizeAtLeast(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									//TODO(TF3.0) make subscription a string id
									"subscription": schema.Int64Attribute{
										Description: "The subscription (id) to which the database belongs",
										Required:    true,
									},
									//TODO(TF3.0) make database a string id
									"database": schema.Int64Attribute{
										Description: "The database (id)",
										Required:    true,
									},
									"regions": schema.SetAttribute{
										Description: "For ActiveActive databases only",
										Optional:    true,
										ElementType: types.StringType,
									},
								},
							},
						},
					},
				},
			},
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Read:   true,
				Update: true,
				Delete: true,
			}),
		},
	}
}

func (r *aclRoleResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {StateUpgrader: upgradeSDKAclRoleState},
	}
}

// upgradeSDKAclRoleState reads a state stored by SDKv2 (version 0), whose attributes
// match the current schema. SDKv2 stored an omitted regions argument as an empty set,
// and a non-computed attribute omitted from the configuration is planned as null, so a
// plan without refresh would update the role. Empty regions are stored as null here.
func upgradeSDKAclRoleState(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	raw, err := req.RawState.UnmarshalWithOpts(resp.State.Schema.Type().TerraformType(ctx), tfprotov6.UnmarshalOpts{
		ValueFromJSONOpts: tftypes.ValueFromJSONOpts{IgnoreUndefinedAttributes: true},
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to read the SDKv2 state", err.Error())
		return
	}
	resp.State.Raw = raw

	var rules RuleSetValue
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("rule"), &rules)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rules, diags := withNullEmptyRegions(ctx, rules)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("rule"), rules)...)
}

func withNullEmptyRegions(ctx context.Context, ruleSet RuleSetValue) (RuleSetValue, diag.Diagnostics) {
	if ruleSet.IsNull() || ruleSet.IsUnknown() {
		return ruleSet, nil
	}

	rules, diags := ruleSet.AsModels(ctx)
	if diags.HasError() {
		return ruleSet, diags
	}
	for i, rule := range rules {
		if rule.Database.IsNull() || rule.Database.IsUnknown() {
			continue
		}
		databases, databaseDiags := rule.Database.AsModels(ctx)
		diags.Append(databaseDiags...)
		if diags.HasError() {
			return ruleSet, diags
		}
		for j, database := range databases {
			if !database.Regions.IsNull() && !database.Regions.IsUnknown() && len(database.Regions.Elements()) == 0 {
				databases[j].Regions = types.SetNull(types.StringType)
			}
		}
		databaseSet, databaseDiags := types.SetValueFrom(ctx, ruleDatabaseElemType, databases)
		diags.Append(databaseDiags...)
		if diags.HasError() {
			return ruleSet, diags
		}
		rules[i].Database = RuleDatabaseSetValue{SetValue: databaseSet}
	}

	upgraded, ruleDiags := types.SetValueFrom(ctx, ruleElemType, rules)
	diags.Append(ruleDiags...)
	if diags.HasError() {
		return ruleSet, diags
	}
	return RuleSetValue{SetValue: upgraded}, diags
}
