package provider

import (
	"fmt"
	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"
	"log"
	"strings"
)

// ApiClient will be offered by both Providers
type ApiClient struct {
	client *rediscloudApi.Client
}

type DebugLogger struct{}

func (d *DebugLogger) Printf(format string, v ...interface{}) {
	log.Printf("[DEBUG] [rediscloud-go-api] "+format, v...)
}

func (d *DebugLogger) Println(v ...interface{}) {
	var items []string
	for _, i := range v {
		items = append(items, fmt.Sprintf("%s", i))
	}
	log.Printf("[DEBUG] [rediscloud-go-api] %s", strings.Join(items, " "))
}
