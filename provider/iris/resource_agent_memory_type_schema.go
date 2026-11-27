package iris

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func AgentMemoryTypeResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Manages a custom memory type. Description and fields are immutable within a store. Destroy disables extraction before unregistering the type, permanently retiring its name in that store. Use a new name for a different schema. Extraction can be updated only if a strategy was configured when the type was created.",
		Attributes: map[string]schema.Attribute{
			"description": schema.StringAttribute{
				Required:            true,
				Description:         "Human-readable description of what this memory type captures.",
				MarkdownDescription: "Human-readable description of what this memory type captures.",

				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"extraction_prompt":  schema.StringAttribute{Optional: true, Computed: true, Description: "Prompt for extracting memories of this type."},
			"extraction_enabled": schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether extraction is active. Defaults to true when a prompt is supplied."},
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Memory-type identifier in the form `<store_id>/<name>`, also used for import.",
				MarkdownDescription: "Memory-type identifier in the form `<store_id>/<name>`, also used for import.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				Description:         "Unique type name within the store. Must not collide with built-in types\n(semantic, episodic, message, session_summary, session_summary_view).",
				MarkdownDescription: "Unique type name within the store. Must not collide with built-in types\n(semantic, episodic, message, session_summary, session_summary_view).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{pathComponentValidator{}},
			},
			"revision": schema.StringAttribute{
				Computed:            true,
				Description:         "Opaque revision returned on reads. Server assigned; ignored on creation.",
				MarkdownDescription: "Opaque revision returned on reads. Server assigned; ignored on creation.",
			},
			"store_id": schema.StringAttribute{
				Required:            true,
				Description:         "Identifier of the parent memory store.",
				MarkdownDescription: "Identifier of the parent memory store.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{pathComponentValidator{}},
			},
		}, Blocks: map[string]schema.Block{"field": memoryFieldBlock(), "timeouts": timeouts.BlockAll(ctx)},
	}
}

type AgentMemoryTypeModel struct {
	Timeouts          timeouts.Value `tfsdk:"timeouts"`
	Description       types.String   `tfsdk:"description"`
	ExtractionPrompt  types.String   `tfsdk:"extraction_prompt"`
	ExtractionEnabled types.Bool     `tfsdk:"extraction_enabled"`
	Fields            types.Set      `tfsdk:"field"`
	ID                types.String   `tfsdk:"id"`
	Name              types.String   `tfsdk:"name"`
	Revision          types.String   `tfsdk:"revision"`
	StoreID           types.String   `tfsdk:"store_id"`
}
