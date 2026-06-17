package essentialsplan

import (
	"context"
	"strings"

	"github.com/RedisLabs/rediscloud-go-api/service/fixed/plans"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

func getPlanList(ctx context.Context, model EssentialsPlanModel, api *client.ApiClient) ([]*plans.GetPlanResponse, error) {
	var list []*plans.GetPlanResponse
	var err error

	if !model.SubscriptionID.IsNull() {
		list, err = api.Client.FixedPlanSubscriptions.List(ctx, int(model.SubscriptionID.ValueInt64()))
	} else if !model.CloudProvider.IsNull() && model.CloudProvider.ValueString() != "" {
		list, err = api.Client.FixedPlans.ListWithProvider(ctx, strings.ToUpper(model.CloudProvider.ValueString()))
	} else {
		list, err = api.Client.FixedPlans.List(ctx)
	}

	return list, err
}

func filterPlans(allPlans []*plans.GetPlanResponse, filters []func(plan *plans.GetPlanResponse) bool) []*plans.GetPlanResponse {
	var filtered []*plans.GetPlanResponse
	for _, candidatePlan := range allPlans {
		if filterPlan(candidatePlan, filters) {
			filtered = append(filtered, candidatePlan)
		}
	}

	return filtered
}

func filterPlan(plan *plans.GetPlanResponse, filters []func(plan *plans.GetPlanResponse) bool) bool {
	for _, f := range filters {
		if !f(plan) {
			return false
		}
	}
	return true
}
