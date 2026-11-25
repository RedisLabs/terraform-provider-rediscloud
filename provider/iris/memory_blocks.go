package iris

import (
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"

	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func memoryFieldBlock() schema.SetNestedBlock {
	return schema.SetNestedBlock{
		Description: "Structured fields of this memory type. Immutable after creation.",
		NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Field name.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"description": schema.StringAttribute{
				Required:    true,
				Description: "Description of the field.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "Value type.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
		}},
	}
}
func customMemoryTypeBlock() schema.SetNestedBlock {
	return schema.SetNestedBlock{
		Description:   "Initial custom memory types. Retained from configuration because the store API does not return them; changes replace the store. Use the memory-type resource for ongoing CRUD.",
		PlanModifiers: []planmodifier.Set{setplanmodifier.RequiresReplace()},
		NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Unique type name.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"description": schema.StringAttribute{
				Required:    true,
				Description: "What this memory type captures.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"extraction_prompt": schema.StringAttribute{
				Optional:    true,
				Description: "Prompt for extracting this memory type.",
			},
			"extraction_enabled": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether extraction is enabled. The service defaults to true when a prompt is supplied.",
				Validators:  []validator.Bool{boolvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("extraction_prompt"))},
			},
		}, Blocks: map[string]schema.Block{"field": memoryFieldBlock()}},
	}
}
func builtInDetectorBlock() schema.SetNestedBlock {
	return schema.SetNestedBlock{
		Description: "A selected built-in sensitive-data detector.",
		NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "Built-in detector ID.",
			},
			"enabled": schema.BoolAttribute{
				Required:    true,
				Description: "Whether this detector is active.",
			},
			"action": schema.StringAttribute{
				Optional:    true,
				Description: "Action on a match. Defaults to REDACT.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
		}},
	}
}
func customDetectorBlock() schema.SetNestedBlock {
	return schema.SetNestedBlock{
		Description: "A custom regular-expression sensitive-data detector.",
		NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Unique detector name.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"enabled": schema.BoolAttribute{
				Required:    true,
				Description: "Whether this detector is active.",
			},
			"action": schema.StringAttribute{
				Optional:    true,
				Description: "Action on a match. Defaults to REDACT.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"pattern": schema.StringAttribute{
				Required:    true,
				Description: "RE2 regular expression. It must not match empty text.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
		}},
	}
}

type MemoryFieldModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Type        types.String `tfsdk:"type"`
}

type CustomMemoryTypeModel struct {
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	Fields            types.Set    `tfsdk:"field"`
	ExtractionPrompt  types.String `tfsdk:"extraction_prompt"`
	ExtractionEnabled types.Bool   `tfsdk:"extraction_enabled"`
}

type BuiltInDetectorModel struct {
	ID      types.String `tfsdk:"id"`
	Enabled types.Bool   `tfsdk:"enabled"`
	Action  types.String `tfsdk:"action"`
}

type CustomDetectorModel struct {
	Name    types.String `tfsdk:"name"`
	Enabled types.Bool   `tfsdk:"enabled"`
	Action  types.String `tfsdk:"action"`
	Pattern types.String `tfsdk:"pattern"`
}

var memoryFieldElementType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(MemoryFieldModel{})}
var customMemoryTypeElementType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(CustomMemoryTypeModel{Fields: types.SetNull(memoryFieldElementType)})}
var builtInDetectorElementType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(BuiltInDetectorModel{})}
var customDetectorElementType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(CustomDetectorModel{})}
