package transitgateway

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/transit_gateway/attachments"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
)

// ParseTransitGatewayAttachmentId parses a Pro TGW attachment ID with format "subscription_id/tgw_id".
func ParseTransitGatewayAttachmentId(id string) (subscriptionId int, tgwId int, err error) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected format of ID (%q), expected <subscription_id>/<tgw_id>", id)
	}

	subscriptionId, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse subscription_id: %w", err)
	}
	tgwId, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse tgw_id: %w", err)
	}

	return subscriptionId, tgwId, nil
}

// ParseActiveActiveTransitGatewayAttachmentId parses an AA TGW attachment ID with format "subscription_id/region_id/tgw_id".
func ParseActiveActiveTransitGatewayAttachmentId(id string) (subscriptionId int, regionId int, tgwId int, err error) {
	parts := strings.Split(id, "/")
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("unexpected format of ID (%q), expected <subscription_id>/<region_id>/<tgw_id>", id)
	}

	subscriptionId, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse subscription_id: %w", err)
	}
	regionId, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse region_id: %w", err)
	}
	tgwId, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse tgw_id: %w", err)
	}

	return subscriptionId, regionId, tgwId, nil
}

// BuildActiveActiveTransitGatewayAttachmentId builds an AA TGW attachment ID with format "subscription_id/region_id/tgw_id".
func BuildActiveActiveTransitGatewayAttachmentId(subscriptionId, regionId, tgwId int) string {
	return fmt.Sprintf("%d/%d/%d", subscriptionId, regionId, tgwId)
}

// TransitGatewayInvitationModel describes one element of the computed invitations list.
type TransitGatewayInvitationModel struct {
	ID               types.Int64  `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	ResourceShareUID types.String `tfsdk:"resource_share_uid"`
	AwsAccountID     types.String `tfsdk:"aws_account_id"`
	Status           types.String `tfsdk:"status"`
	SharedDate       types.String `tfsdk:"shared_date"`
}

// TransitGatewayInvitationElemType is the element type of the computed invitations list.
var TransitGatewayInvitationElemType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(TransitGatewayInvitationModel{})}

// TransitGatewayInvitationsFromAPI maps API invitations to a known list ordered by invitation ID.
func TransitGatewayInvitationsFromAPI(ctx context.Context, invitations []*attachments.TransitGatewayInvitation) (types.List, diag.Diagnostics) {
	sorted := make([]*attachments.TransitGatewayInvitation, 0, len(invitations))
	for _, invitation := range invitations {
		if invitation != nil {
			sorted = append(sorted, invitation)
		}
	}
	sort.SliceStable(sorted, func(i, j int) bool {
		return redis.IntValue(sorted[i].Id) < redis.IntValue(sorted[j].Id)
	})

	elems := make([]TransitGatewayInvitationModel, 0, len(sorted))
	for _, invitation := range sorted {
		elems = append(elems, TransitGatewayInvitationModel{
			ID:               types.Int64Value(int64(redis.IntValue(invitation.Id))),
			Name:             types.StringValue(redis.StringValue(invitation.Name)),
			ResourceShareUID: types.StringValue(redis.StringValue(invitation.ResourceShareUid)),
			AwsAccountID:     types.StringValue(redis.StringValue(invitation.AwsAccountId)),
			Status:           types.StringValue(redis.StringValue(invitation.Status)),
			SharedDate:       types.StringValue(redis.StringValue(invitation.SharedDate)),
		})
	}

	return types.ListValueFrom(ctx, TransitGatewayInvitationElemType, elems)
}

// waitForInvitations polls fetch until it returns at least one invitation or the timeout expires.
func waitForInvitations(
	ctx context.Context,
	api *client.ApiClient,
	timeout time.Duration,
	label string,
	location string,
	fetch func(context.Context) ([]*attachments.TransitGatewayInvitation, error),
) ([]*attachments.TransitGatewayInvitation, diag.Diagnostics) {
	var diags diag.Diagnostics
	wait := &retry.StateChangeConf{
		Pending:      []string{"waiting"},
		Target:       []string{"found"},
		Timeout:      timeout,
		Delay:        api.WaitDelay(5 * time.Second),
		PollInterval: api.WaitPollInterval(10 * time.Second),

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for %s invitations to appear for %s", label, location)

			inv, err := fetch(ctx)
			if err != nil {
				return nil, "", err
			}

			if len(inv) == 0 {
				return nil, "waiting", nil
			}

			return inv, "found", nil
		},
	}

	result, err := wait.WaitForStateContext(ctx)
	if err != nil {
		diags.AddError("Unable to Read Transit Gateway Invitations", fmt.Sprintf("Timeout waiting for %s invitations to appear for %s: %s", label, location, err))
		return nil, diags
	}

	invitations, ok := result.([]*attachments.TransitGatewayInvitation)
	if !ok {
		diags.AddError("Unable to Read Transit Gateway Invitations", fmt.Sprintf("Internal error: unexpected result type from wait operation for %s", location))
		return nil, diags
	}

	return invitations, diags
}

// TransitGatewayAttachmentFilter selects Transit Gateway attachments returned by the API.
type TransitGatewayAttachmentFilter func(tgwa *attachments.TransitGatewayAttachment) bool

// FilterTransitGatewayAttachments returns the attachments in the task that satisfy every filter.
func FilterTransitGatewayAttachments(getAttachmentsTask *attachments.GetAttachmentsTask, filters []TransitGatewayAttachmentFilter) []*attachments.TransitGatewayAttachment {
	var filtered []*attachments.TransitGatewayAttachment

	if getAttachmentsTask == nil || getAttachmentsTask.Response == nil || getAttachmentsTask.Response.Resource == nil {
		return filtered
	}

	for _, tgwa := range getAttachmentsTask.Response.Resource.TransitGatewayAttachment {
		if transitGatewayAttachmentMatches(tgwa, filters) {
			filtered = append(filtered, tgwa)
		}
	}
	return filtered
}

func transitGatewayAttachmentMatches(tgwa *attachments.TransitGatewayAttachment, filters []TransitGatewayAttachmentFilter) bool {
	for _, filter := range filters {
		if !filter(tgwa) {
			return false
		}
	}
	return true
}

// CidrsFromAPI maps attachment CIDRs to a known list of addresses, keeping the API order.
func CidrsFromAPI(ctx context.Context, cidrs []*attachments.Cidr) (types.List, diag.Diagnostics) {
	addresses := make([]string, 0, len(cidrs))
	for _, cidr := range cidrs {
		addresses = append(addresses, redis.StringValue(cidr.CidrAddress))
	}
	return types.ListValueFrom(ctx, types.StringType, addresses)
}

// transitGatewayDataSourceFilters mirrors the SDKv2 GetOk semantics, where zero values do not filter.
func transitGatewayDataSourceFilters(tgwId types.Int64, awsTgwUid types.String) []TransitGatewayAttachmentFilter {
	var filters []TransitGatewayAttachmentFilter

	if v := tgwId.ValueInt64(); v != 0 {
		filters = append(filters, func(tgwa *attachments.TransitGatewayAttachment) bool {
			return int64(redis.IntValue(tgwa.Id)) == v
		})
	}

	if v := awsTgwUid.ValueString(); v != "" {
		filters = append(filters, func(tgwa *attachments.TransitGatewayAttachment) bool {
			return redis.StringValue(tgwa.AwsTgwUid) == v
		})
	}

	return filters
}

// waitForMatchingTransitGateways polls fetch until at least one attachment satisfies the filters or the timeout expires.
func waitForMatchingTransitGateways(
	ctx context.Context,
	api *client.ApiClient,
	timeout time.Duration,
	label string,
	location string,
	fetch func(context.Context) (*attachments.GetAttachmentsTask, error),
	filters []TransitGatewayAttachmentFilter,
) ([]*attachments.TransitGatewayAttachment, diag.Diagnostics) {
	var diags diag.Diagnostics
	wait := &retry.StateChangeConf{
		Pending:      []string{"waiting"},
		Target:       []string{"found"},
		Timeout:      timeout,
		Delay:        api.WaitDelay(5 * time.Second),
		PollInterval: api.WaitPollInterval(10 * time.Second),

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for %s to appear for %s", label, location)

			tgwTask, err := fetch(ctx)
			if err != nil {
				return nil, "", err
			}

			if tgwTask == nil || tgwTask.Response == nil || tgwTask.Response.Resource == nil {
				return nil, "waiting", nil
			}

			filtered := FilterTransitGatewayAttachments(tgwTask, filters)
			if len(filtered) == 0 {
				return nil, "waiting", nil
			}

			return filtered, "found", nil
		},
	}

	result, err := wait.WaitForStateContext(ctx)
	if err != nil {
		diags.AddError("Unable to Read Transit Gateway", fmt.Sprintf("Timeout waiting for %s to appear for %s: %s", label, location, err))
		return nil, diags
	}

	filtered, ok := result.([]*attachments.TransitGatewayAttachment)
	if !ok {
		diags.AddError("Unable to Read Transit Gateway", fmt.Sprintf("Internal error: unexpected result type from wait operation for %s", location))
		return nil, diags
	}

	return filtered, diags
}

// singleTransitGateway reports the SDKv2 errors for empty or ambiguous data source queries.
func singleTransitGateway(filtered []*attachments.TransitGatewayAttachment) (*attachments.TransitGatewayAttachment, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(filtered) == 0 {
		diags.AddError("No Transit Gateway Found", "Your query returned no results. Please change your search criteria and try again.")
		return nil, diags
	}
	if len(filtered) > 1 {
		diags.AddError("Multiple Transit Gateways Found", "Your query returned more than one result. Please change try a more specific search criteria and try again.")
		return nil, diags
	}
	return filtered[0], diags
}

// cidrsToAPI converts a configured CIDR list, sending null elements as empty strings like the SDKv2 implementation did.
func cidrsToAPI(ctx context.Context, cidrs types.List) ([]*string, diag.Diagnostics) {
	result := make([]*string, 0)
	if cidrs.IsNull() || cidrs.IsUnknown() {
		return result, nil
	}

	var values []types.String
	diags := cidrs.ElementsAs(ctx, &values, false)
	if diags.HasError() {
		return nil, diags
	}

	for _, value := range values {
		result = append(result, redis.String(value.ValueString()))
	}
	return result, diags
}

// findTransitGatewayAttachment returns the attachment with tgwId, or nil when the API does not list it.
func findTransitGatewayAttachment(tgwTask *attachments.GetAttachmentsTask, tgwId int, idDescription string) (*attachments.TransitGatewayAttachment, diag.Diagnostics) {
	var diags diag.Diagnostics
	tgws := FilterTransitGatewayAttachments(tgwTask, []TransitGatewayAttachmentFilter{
		func(tgw *attachments.TransitGatewayAttachment) bool {
			return redis.IntValue(tgw.Id) == tgwId
		},
	})

	if len(tgws) == 0 {
		return nil, diags
	}

	if len(tgws) > 1 {
		diags.AddError("Unable to Read Transit Gateway", "More than one Transit Gateway identified! "+idDescription)
		return nil, diags
	}

	return tgws[0], diags
}

func isTransitGatewayAttachmentNotFound(err error) bool {
	return strings.Contains(err.Error(), "TGW_ATTACHMENT_DOES_NOT_EXIST")
}
