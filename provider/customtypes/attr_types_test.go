package customtypes_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
)

// sampleModel documents, by example, what AttrTypesOf reads: a flat struct of framework
// value types. Note GoName vs its tag — AttrTypesOf keys the map by the `tfsdk` **tag**,
// never the Go field name — and the mix of value kinds it maps to attr.Types.
type sampleModel struct {
	GoName  types.String  `tfsdk:"tag_name"`
	Count   types.Int64   `tfsdk:"count"`
	Ratio   types.Float64 `tfsdk:"ratio"`
	Enabled types.Bool    `tfsdk:"enabled"`
}

// TestAttrTypesOf is the canonical "what does this function do" example: model in,
// attr-type map out — keyed by tfsdk tag, valued by each field's attr.Type.
func TestAttrTypesOf(t *testing.T) {
	got := customtypes.AttrTypesOf(sampleModel{})

	assert.Equal(t, map[string]attr.Type{
		"tag_name": types.StringType,  // types.String  -> StringType, keyed by the tag
		"count":    types.Int64Type,   // types.Int64   -> Int64Type
		"ratio":    types.Float64Type, // types.Float64 -> Float64Type
		"enabled":  types.BoolType,    // types.Bool    -> BoolType
	}, got)
}

// TestAttrTypesOfSkipsUntaggedAndDashFields documents the two fields it ignores:
// those with no `tfsdk` tag and those tagged `-`.
func TestAttrTypesOfSkipsUntaggedAndDashFields(t *testing.T) {
	type withSkips struct {
		Kept    types.String `tfsdk:"kept"`
		Ignored types.String `tfsdk:"-"` // explicitly excluded
		NoTag   types.String // no tfsdk tag -> excluded
	}

	got := customtypes.AttrTypesOf(withSkips{})

	assert.Equal(t, map[string]attr.Type{"kept": types.StringType}, got)
	assert.NotContains(t, got, "-")
	assert.Len(t, got, 1)
}

// TestAttrTypesOfEmptyStruct: no tagged fields yields an empty, non-nil map (safe to
// pass to types.ObjectValue / a schema without a nil check).
func TestAttrTypesOfEmptyStruct(t *testing.T) {
	got := customtypes.AttrTypesOf(struct{}{})

	assert.NotNil(t, got)
	assert.Empty(t, got)
}
