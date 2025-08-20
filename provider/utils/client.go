package utils

import rediscloudApi "github.com/RedisLabs/rediscloud-go-api"

// Lock that must be acquired when modifying something related to a subscription as only one _thing_ can modify a subscription and all sub-resources at any time
var SubscriptionMutex = NewPerIdLock()

type ApiClient struct {
	Client *rediscloudApi.Client
}
