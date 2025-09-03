package client

import (
	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"
)

type ApiClient struct {
	Client *rediscloudApi.Client
}
