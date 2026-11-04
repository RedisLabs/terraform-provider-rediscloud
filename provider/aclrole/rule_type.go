package aclrole

import (
	"context"
	"fmt"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/access_control_lists/roles"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
)

var (
	_ basetypes.SetTypable  = RuleDatabaseSetType{}
	_ basetypes.SetValuable = RuleDatabaseSetValue{}
	_ basetypes.SetTypable  = RuleSetType{}
	_ basetypes.SetValuable = RuleSetValue{}
)

// RuleDatabaseModel represents one database a role rule applies to.
type RuleDatabaseModel struct {
	//TODO(TF3.0) make subscription a string id
	Subscription types.Int64 `tfsdk:"subscription"`
	//TODO(TF3.0) make database a string id
	Database types.Int64 `tfsdk:"database"`
	Regions  types.Set   `tfsdk:"regions"`
}

// RuleModel represents one rule of a role and the databases it applies to.
type RuleModel struct {
	Name     types.String         `tfsdk:"name"`
	Database RuleDatabaseSetValue `tfsdk:"database"`
}

var ruleDatabaseElemType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(RuleDatabaseModel{
	Regions: types.SetNull(types.StringType),
})}

var ruleElemType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(RuleModel{})}

// RuleDatabaseSetType gives the databases of a role rule their provider-specific value type.
type RuleDatabaseSetType struct {
	basetypes.SetType
}

// NewRuleDatabaseSetType returns the custom type for the databases of a role rule.
func NewRuleDatabaseSetType() RuleDatabaseSetType {
	return RuleDatabaseSetType{SetType: basetypes.SetType{ElemType: ruleDatabaseElemType}}
}

func (t RuleDatabaseSetType) Equal(other attr.Type) bool {
	otherType, ok := other.(RuleDatabaseSetType)
	if !ok {
		return false
	}
	return t.SetType.Equal(otherType.SetType)
}

func (t RuleDatabaseSetType) String() string {
	return "rediscloud.aclrole.RuleDatabaseSetType"
}

func (t RuleDatabaseSetType) ValueFromSet(_ context.Context, value basetypes.SetValue) (basetypes.SetValuable, diag.Diagnostics) {
	return RuleDatabaseSetValue{SetValue: value}, nil
}

func (t RuleDatabaseSetType) ValueFromTerraform(ctx context.Context, value tftypes.Value) (attr.Value, error) {
	terraformValue, err := t.SetType.ValueFromTerraform(ctx, value)
	if err != nil {
		return nil, err
	}

	setValue, ok := terraformValue.(basetypes.SetValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T, expected basetypes.SetValue", terraformValue)
	}

	return RuleDatabaseSetValue{SetValue: setValue}, nil
}

func (t RuleDatabaseSetType) ValueType(context.Context) attr.Value {
	return RuleDatabaseSetValue{}
}

// RuleDatabaseSetValue stores the databases of a role rule.
type RuleDatabaseSetValue struct {
	basetypes.SetValue
}

func (v RuleDatabaseSetValue) Equal(other attr.Value) bool {
	otherValue, ok := other.(RuleDatabaseSetValue)
	if !ok {
		return false
	}
	return v.SetValue.Equal(otherValue.SetValue)
}

func (v RuleDatabaseSetValue) Type(context.Context) attr.Type {
	return NewRuleDatabaseSetType()
}

// AsModels decodes a known set of rule databases. Callers must handle null and unknown values first.
func (v RuleDatabaseSetValue) AsModels(ctx context.Context) ([]RuleDatabaseModel, diag.Diagnostics) {
	var models []RuleDatabaseModel
	diags := v.ElementsAs(ctx, &models, false)
	return models, diags
}

// RuleSetType gives role rules their provider-specific value type.
type RuleSetType struct {
	basetypes.SetType
}

// NewRuleSetType returns the custom type for the rules of a role.
func NewRuleSetType() RuleSetType {
	return RuleSetType{SetType: basetypes.SetType{ElemType: ruleElemType}}
}

func (t RuleSetType) Equal(other attr.Type) bool {
	otherType, ok := other.(RuleSetType)
	if !ok {
		return false
	}
	return t.SetType.Equal(otherType.SetType)
}

func (t RuleSetType) String() string {
	return "rediscloud.aclrole.RuleSetType"
}

func (t RuleSetType) ValueFromSet(_ context.Context, value basetypes.SetValue) (basetypes.SetValuable, diag.Diagnostics) {
	return RuleSetValue{SetValue: value}, nil
}

func (t RuleSetType) ValueFromTerraform(ctx context.Context, value tftypes.Value) (attr.Value, error) {
	terraformValue, err := t.SetType.ValueFromTerraform(ctx, value)
	if err != nil {
		return nil, err
	}

	setValue, ok := terraformValue.(basetypes.SetValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T, expected basetypes.SetValue", terraformValue)
	}

	return RuleSetValue{SetValue: setValue}, nil
}

func (t RuleSetType) ValueType(context.Context) attr.Value {
	return RuleSetValue{}
}

// RuleSetValue stores the rules of a role.
type RuleSetValue struct {
	basetypes.SetValue
}

func (v RuleSetValue) Equal(other attr.Value) bool {
	otherValue, ok := other.(RuleSetValue)
	if !ok {
		return false
	}
	return v.SetValue.Equal(otherValue.SetValue)
}

func (v RuleSetValue) Type(context.Context) attr.Type {
	return NewRuleSetType()
}

// AsModels decodes a known set of role rules. Callers must handle null and unknown values first.
func (v RuleSetValue) AsModels(ctx context.Context) ([]RuleModel, diag.Diagnostics) {
	var models []RuleModel
	diags := v.ElementsAs(ctx, &models, false)
	return models, diags
}

// RulesFromAPI converts the rules of an API role into a rule set. Databases without
// regions get a known empty regions set, as the SDKv2 implementation wrote.
func RulesFromAPI(ctx context.Context, apiRules []*roles.GetRuleInRoleResponse) (RuleSetValue, diag.Diagnostics) {
	return rulesFromAPI(ctx, apiRules, func(string, int64, int64) types.Set {
		return types.SetValueMust(types.StringType, []attr.Value{})
	})
}

// rulesFromAPI converts API rules, asking emptyRegions for the value to store when the
// API returns no regions for a database of the named rule.
func rulesFromAPI(ctx context.Context, apiRules []*roles.GetRuleInRoleResponse, emptyRegions func(ruleName string, subscription, database int64) types.Set) (RuleSetValue, diag.Diagnostics) {
	var diags diag.Diagnostics
	rules := make([]RuleModel, 0, len(apiRules))
	for _, apiRule := range apiRules {
		if apiRule == nil {
			continue
		}
		ruleName := redis.StringValue(apiRule.RuleName)

		databases := make([]RuleDatabaseModel, 0, len(apiRule.Databases))
		for _, apiDatabase := range apiRule.Databases {
			if apiDatabase == nil {
				continue
			}
			subscription := int64(redis.IntValue(apiDatabase.SubscriptionId))
			database := int64(redis.IntValue(apiDatabase.DatabaseId))

			regions := emptyRegions(ruleName, subscription, database)
			if len(apiDatabase.Regions) > 0 {
				var regionDiags diag.Diagnostics
				regions, regionDiags = types.SetValueFrom(ctx, types.StringType, redis.StringSliceValue(apiDatabase.Regions...))
				diags.Append(regionDiags...)
				if diags.HasError() {
					return nullRuleSetValue(), diags
				}
			}

			databases = append(databases, RuleDatabaseModel{
				Subscription: types.Int64Value(subscription),
				Database:     types.Int64Value(database),
				Regions:      regions,
			})
		}

		databaseSet, databaseDiags := types.SetValueFrom(ctx, ruleDatabaseElemType, databases)
		diags.Append(databaseDiags...)
		if diags.HasError() {
			return nullRuleSetValue(), diags
		}

		rules = append(rules, RuleModel{
			Name:     types.StringValue(ruleName),
			Database: RuleDatabaseSetValue{SetValue: databaseSet},
		})
	}

	ruleSet, ruleDiags := types.SetValueFrom(ctx, ruleElemType, rules)
	diags.Append(ruleDiags...)
	if diags.HasError() {
		return nullRuleSetValue(), diags
	}

	return RuleSetValue{SetValue: ruleSet}, diags
}

func nullRuleSetValue() RuleSetValue {
	return RuleSetValue{SetValue: types.SetNull(ruleElemType)}
}
