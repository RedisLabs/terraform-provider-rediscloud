package pro

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

func ApplyProSubscriptionUpdates(ctx context.Context, api *client.ApiClient, subId int, plan, prior *ProSubscriptionResourceModel) diag.Diagnostics {
	r := &proSubscriptionResource{client: api}
	return r.applyUpdates(ctx, subId, plan, prior, time.Minute)
}

func DeleteProSubscription(ctx context.Context, api *client.ApiClient, subId int, timeout time.Duration) diag.Diagnostics {
	r := &proSubscriptionResource{client: api}
	return r.deleteSubscription(ctx, subId, timeout)
}

var CustomerManagedKeyElemType = customerManagedKeyElemType

var (
	SubscriptionAllowlistElemType     = subscriptionAllowlistElemType
	SubscriptionCloudProviderElemType = subscriptionCloudProviderElemType
	SubscriptionRegionElemType        = subscriptionRegionElemType
	SubscriptionNetworkElemType       = subscriptionNetworkElemType
	CreationPlanElemType              = creationPlanElemType
	EmptyMaintenanceList              = emptyMaintenanceList
)

func ReadProSubscriptionAfterApply(ctx context.Context, api *client.ApiClient, subId int, plan *ProSubscriptionResourceModel, state *tfsdk.State) diag.Diagnostics {
	r := &proSubscriptionResource{client: api}
	return r.readAfterApply(ctx, subId, plan, state)
}
