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

// Ensure the custom list type/value satisfy the framework interfaces at compile time.
var (
	_ basetypes.ListTypable  = PricingListType{}
	_ basetypes.ListValuable = PricingListValue{}
)

// PricingModel is the typed, tfsdk-tagged view of a pricing entry and the single source of
// truth for its shape: pricingAttrTypes is derived from the tfsdk taged fields (see below), and it's the
// compile-checked surface for building each entry (newPricingObject) and for decoding the
// list (PricingListValue.ElementsAs into []PricingModel).
// Note that the tfsdk tags must still agree with the pricing block's Attributes in the TF schema.
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

// pricingObjectType is the element type of the pricing list: a plain object whose attribute
// types come from PricingModel
var pricingObjectType = types.ObjectType{AttrTypes: pricingAttrTypes}

// PricingListType is the custom list type for pricing entries, where the entries themselves
// are generic objects with element attributes derived from PricingModel.
// PricingListType embeds basetypes.ListType, overriding only the identity/conversion methods.
type PricingListType struct {
	basetypes.ListType
}

// NewPricingListType returns a PricingListType whose element type is the pricing object. Use
// it as the schema block's list-level CustomType and wherever the list type is needed.
func NewPricingListType() PricingListType {
	return PricingListType{basetypes.ListType{ElemType: pricingObjectType}}
}

func (t PricingListType) Equal(o attr.Type) bool {
	other, ok := o.(PricingListType)
	if !ok {
		return false
	}
	return t.ListType.Equal(other.ListType)
}

// String is the type's human-facing label in diagnostics/logs. Fully qualified on purpose so
// a failure involving this type points clearly at the provider + package.
func (t PricingListType) String() string {
	return "rediscloud.customtypes.PricingListType"
}

func (t PricingListType) ValueFromList(ctx context.Context, in basetypes.ListValue) (basetypes.ListValuable, diag.Diagnostics) {
	return PricingListValue{ListValue: in}, nil
}

func (t PricingListType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	val, err := t.ListType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}
	list, ok := val.(basetypes.ListValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T, expected basetypes.ListValue", val)
	}
	return PricingListValue{ListValue: list}, nil
}

func (t PricingListType) ValueType(ctx context.Context) attr.Value {
	return PricingListValue{}
}

// PricingListValue is the strongly typed list value holding the pricing-entry objects. It
// embeds basetypes.ListValue for the standard list behaviour and overrides identity so it
// reports itself as a PricingListType.
type PricingListValue struct {
	basetypes.ListValue
}

func (v PricingListValue) Equal(o attr.Value) bool {
	other, ok := o.(PricingListValue)
	if !ok {
		return false
	}
	return v.ListValue.Equal(other.ListValue)
}

func (v PricingListValue) Type(ctx context.Context) attr.Type {
	return NewPricingListType()
}

// newPricingObject builds one pricing entry as a generic object value, mirroring the SDKv2
// flatten's value mapping — empty string for nil strings, 0 for nil numbers — so the
// Framework migration stays backward compatible. It builds through the tfsdk-tagged
// PricingModel, so the field mapping is compile-checked rather than a string-keyed map.
func newPricingObject(ctx context.Context, p *pricing.Pricing) (types.Object, diag.Diagnostics) {
	//TODO(TF3.0) make the pricing string fields (and price_per_unit) nullable
	return types.ObjectValueFrom(ctx, pricingAttrTypes, PricingModel{
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
}

// NewPricingList builds the pricing list value (a PricingListValue of pricing objects) from a
// pricing API response. Entries are sorted by a composite key so the ordered list is stable
// across reads: the Pricing.List API does not guarantee a consistent order, which would
// otherwise churn the list and produce a perpetual plan diff.
func NewPricingList(ctx context.Context, prices []*pricing.Pricing) (PricingListValue, diag.Diagnostics) {
	var diags diag.Diagnostics

	sorted := make([]*pricing.Pricing, len(prices))
	copy(sorted, prices)
	sort.SliceStable(sorted, func(i, j int) bool {
		return pricingSortKey(sorted[i]) < pricingSortKey(sorted[j])
	})

	elems := make([]attr.Value, 0, len(sorted))
	for _, p := range sorted {
		entry, d := newPricingObject(ctx, p)
		diags.Append(d...)
		elems = append(elems, entry)
	}

	list, d := types.ListValue(pricingObjectType, elems)
	diags.Append(d...)
	if diags.HasError() {
		return PricingListValue{ListValue: types.ListNull(pricingObjectType)}, diags
	}
	return PricingListValue{ListValue: list}, diags
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
