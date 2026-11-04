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
