package peering

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	providerAWS = "AWS"
	providerGCP = "GCP"
)

// parsePeeringID splits a `<subscription_id>/<peering_id>` resource ID.
func parsePeeringID(id string) (int, int, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid id: %s", id)
	}

	sub, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}

	peering, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}

	return sub, peering, nil
}

func findVPCPeering(id int, peerings []*subscriptions.VPCPeering) *subscriptions.VPCPeering {
	for _, peering := range peerings {
		if redis.IntValue(peering.ID) == id {
			return peering
		}
	}
	return nil
}

// vpcCidrsFromAPI mirrors the SDKv2 read: a peering created with a single-item
// `vpc_cidrs` is read back with `VPCCidr` set and `VPCCidrs` unset, so each
// field falls back to the other.
func vpcCidrsFromAPI(ctx context.Context, vpcCidr *string, vpcCidrs []*subscriptions.CIDR) (types.String, types.Set, diag.Diagnostics) {
	var cidrs []string
	if len(vpcCidrs) != 0 {
		for _, cidr := range vpcCidrs {
			if vpcCidr == nil {
				vpcCidr = cidr.VPCCidr
			}
			cidrs = append(cidrs, redis.StringValue(cidr.VPCCidr))
		}
	} else {
		cidrs = []string{redis.StringValue(vpcCidr)}
	}

	set, diags := types.SetValueFrom(ctx, types.StringType, cidrs)
	return types.StringValue(redis.StringValue(vpcCidr)), set, diags
}

// isSet mirrors SDKv2 GetOk for strings: null, unknown and empty values are not set.
func isSet(v types.String) bool {
	return !v.IsNull() && !v.IsUnknown() && v.ValueString() != ""
}

// nullIfUnknown resolves computed values that the API did not report, as SDKv2
// left them null in state.
func nullIfUnknown(v *types.String) {
	if v.IsUnknown() {
		*v = types.StringNull()
	}
}

// emptySetIfUnset matches SDKv2, which stored vpc_cidrs as an empty set on
// peerings whose Read never set it (GCP).
func emptySetIfUnset(v *types.Set) {
	if v.IsNull() || v.IsUnknown() {
		*v = types.SetValueMust(types.StringType, []attr.Value{})
	}
}

func vpcCidrsFromPlan(ctx context.Context, v types.Set) ([]*string, diag.Diagnostics) {
	if v.IsNull() || v.IsUnknown() {
		return nil, nil
	}
	var cidrs []string
	diags := v.ElementsAs(ctx, &cidrs, false)
	result := make([]*string, 0, len(cidrs))
	for _, cidr := range cidrs {
		result = append(result, redis.String(cidr))
	}
	return result, diags
}

var _ validator.String = cidrValidator{}

type cidrValidator struct{}

func (v cidrValidator) Description(_ context.Context) string {
	return "value must be a valid CIDR range"
}

func (v cidrValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v cidrValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if _, ipnet, err := net.ParseCIDR(value); err != nil || ipnet == nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid CIDR", fmt.Sprintf("expected %q to be a valid IPv4 Value, got %v: %v", req.Path, value, err))
	}
}
