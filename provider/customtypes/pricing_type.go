package customtypes

import (
	"context"
	"fmt"
	"sort"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/pricing"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// Ensure the custom type/value satisfy the framework interfaces at compile time.
var (
	_ basetypes.ObjectTypable  = PricingType{}
	_ basetypes.ObjectValuable = PricingValue{}
)

// PricingModel is the typed, tfsdk-tagged view of a pricing entry and the single source
// of truth for its shape: pricingAttrTypes is derived from it (see below), and it's the
// compile-checked surface for building/reading PricingValue (NewPricingValue /
// PricingValue.As) as opposed to using string-keyed maps. Its tfsdk tags must still agree with the
// pricing block's schema Attributes.
type PricingModel struct {
	DatabaseName        types.String  `tfsdk:"database_name"`
	Type                types.String  `tfsdk:"type"`
	TypeDetails         types.String  `tfsdk:"type_details"`
	Quantity            types.Int64   `tfsdk:"quantity"`
	QuantityMeasurement types.String  `tfsdk:"quantity_measurement"`
	PricePerUnit        types.Float64 `tfsdk:"price_per_unit"`
	PriceCurrency       types.String  `tfsdk:"price_currency"`
	PricePeriod         types.String  `tfsdk:"price_period"`
	Region              types.String  `tfsdk:"region"`
}

// pricingAttrTypes is derived from PricingModel's tfsdk tags + field types, so the map
// can't drift from the struct.
var pricingAttrTypes = attrTypesOf(PricingModel{})

// PricingType is the custom framework type for a single pricing entry. It embeds
// basetypes.ObjectType so it inherits standard object behaviour and stays current with
// the framework, overriding only the identity/conversion methods so values round-trip
// as PricingValue rather than a bare types.Object.
type PricingType struct {
	basetypes.ObjectType
}

// NewPricingType returns a PricingType with its attribute types populated. Use this
// wherever the type is needed (schema CustomType, list element type, null values).
func NewPricingType() PricingType {
	return PricingType{basetypes.ObjectType{AttrTypes: pricingAttrTypes}}
}

func (t PricingType) Equal(o attr.Type) bool {
	other, ok := o.(PricingType)
	if !ok {
		return false
	}
	return t.ObjectType.Equal(other.ObjectType)
}

// String is the type's human-facing label in diagnostics/logs. Fully qualified on
// purpose so a failure involving this type points clearly at the provider + package.
func (t PricingType) String() string {
	return "rediscloud.customtypes.PricingType"
}

func (t PricingType) ValueFromObject(ctx context.Context, in basetypes.ObjectValue) (basetypes.ObjectValuable, diag.Diagnostics) {
	return PricingValue{ObjectValue: in}, nil
}

func (t PricingType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	val, err := t.ObjectType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}
	obj, ok := val.(basetypes.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T, expected basetypes.ObjectValue", val)
	}
	return PricingValue{ObjectValue: obj}, nil
}

func (t PricingType) ValueType(ctx context.Context) attr.Value {
	return PricingValue{}
}

// PricingValue is the strongly typed state value for a single pricing entry. It embeds
// basetypes.ObjectValue for the standard object behaviour (ToObjectValue,
// ToTerraformValue, null/unknown handling) and overrides identity so it reports itself
// as a PricingType.
type PricingValue struct {
	basetypes.ObjectValue
}

func (v PricingValue) Equal(o attr.Value) bool {
	other, ok := o.(PricingValue)
	if !ok {
		return false
	}
	return v.ObjectValue.Equal(other.ObjectValue)
}

func (v PricingValue) Type(ctx context.Context) attr.Type {
	return NewPricingType()
}

// As decodes the value into a typed PricingModel so callers read fields by name
// (v.As(ctx) → m.DatabaseName) instead of reaching into the string-keyed attribute map.
func (v PricingValue) As(ctx context.Context) (PricingModel, diag.Diagnostics) {
	var m PricingModel
	diags := v.ObjectValue.As(ctx, &m, basetypes.ObjectAsOptions{})
	return m, diags
}

// NewPricingValue builds a PricingValue from a pricing API entry, mirroring the SDKv2
// flatten's value mapping — empty-string for nil strings, 0 for nil numbers — so the
// Framework migration stays backward compatible. It builds through the tfsdk-tagged
// PricingModel, so the field mapping is compile-checked rather than a string-keyed map.
func NewPricingValue(ctx context.Context, p *pricing.Pricing) (PricingValue, diag.Diagnostics) {
	//TODO(TF3.0) make the pricing string fields (and price_per_unit) nullable
	obj, diags := types.ObjectValueFrom(ctx, pricingAttrTypes, PricingModel{
		DatabaseName:        types.StringValue(redis.StringValue(p.DatabaseName)),
		Type:                types.StringValue(redis.StringValue(p.Type)),
		TypeDetails:         types.StringValue(redis.StringValue(p.TypeDetails)),
		Quantity:            types.Int64Value(int64(redis.IntValue(p.Quantity))),
		QuantityMeasurement: types.StringValue(redis.StringValue(p.QuantityMeasurement)),
		PricePerUnit:        types.Float64Value(redis.Float64Value(p.PricePerUnit)),
		PriceCurrency:       types.StringValue(redis.StringValue(p.PriceCurrency)),
		PricePeriod:         types.StringValue(redis.StringValue(p.PricePeriod)),
		Region:              types.StringValue(redis.StringValue(p.Region)),
	})
	return PricingValue{ObjectValue: obj}, diags
}

// NewPricingList builds the pricing list (elements of the custom PricingType) from a
// pricing API response. Entries are sorted by a composite key so the ordered list is
// stable across reads: the Pricing.List API does not guarantee a consistent order, which
// would otherwise churn the list and produce a perpetual plan diff.
func NewPricingList(ctx context.Context, prices []*pricing.Pricing) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	pricingType := NewPricingType()

	sorted := make([]*pricing.Pricing, len(prices))
	copy(sorted, prices)
	sort.SliceStable(sorted, func(i, j int) bool {
		return pricingSortKey(sorted[i]) < pricingSortKey(sorted[j])
	})

	elems := make([]attr.Value, 0, len(sorted))
	for _, p := range sorted {
		entry, d := NewPricingValue(ctx, p)
		diags.Append(d...)
		elems = append(elems, entry)
	}

	list, d := types.ListValue(pricingType, elems)
	diags.Append(d...)
	if diags.HasError() {
		return types.ListNull(pricingType), diags
	}
	return list, diags
}

// pricingSortKey builds a deterministic ordering key for a pricing entry, combining every
// field so the sort is total even when entries differ only in a single value.
func pricingSortKey(p *pricing.Pricing) string {
	return fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%d|%f",
		redis.StringValue(p.Region),
		redis.StringValue(p.Type),
		redis.StringValue(p.TypeDetails),
		redis.StringValue(p.DatabaseName),
		redis.StringValue(p.QuantityMeasurement),
		redis.StringValue(p.PricePeriod),
		redis.StringValue(p.PriceCurrency),
		redis.IntValue(p.Quantity),
		redis.Float64Value(p.PricePerUnit),
	)
}
