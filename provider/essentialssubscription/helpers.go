package essentialssubscription

import (
	fs "github.com/RedisLabs/rediscloud-go-api/service/fixed/subscriptions"
)

func filterFixedSubscriptions(subs []*fs.FixedSubscriptionResponse, filters []func(sub *fs.FixedSubscriptionResponse) bool) []*fs.FixedSubscriptionResponse {
	var filteredSubs []*fs.FixedSubscriptionResponse
	for _, sub := range subs {
		if filterFixedSub(sub, filters) {
			filteredSubs = append(filteredSubs, sub)
		}
	}

	return filteredSubs
}

func filterFixedSub(sub *fs.FixedSubscriptionResponse, filters []func(sub *fs.FixedSubscriptionResponse) bool) bool {
	for _, f := range filters {
		if !f(sub) {
			return false
		}
	}
	return true
}
