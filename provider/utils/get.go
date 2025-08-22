package utils

import (
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// GetString safely retrieves a string value from schema.ResourceData.
func GetString(d *schema.ResourceData, key string) *string {
	if v, ok := d.GetOk(key); ok {
		return redis.String(v.(string))
	}
	return redis.String("")
}

// GetBool safely retrieves a bool value from schema.ResourceData.
func GetBool(d *schema.ResourceData, key string) *bool {
	if v, ok := d.GetOk(key); ok {
		return redis.Bool(v.(bool))
	}
	return redis.Bool(false)
}

// GetInt safely retrieves an int value from schema.ResourceData.
func GetInt(d *schema.ResourceData, key string) *int {
	if v, ok := d.GetOk(key); ok {
		return redis.Int(v.(int))
	}
	return redis.Int(0)
}
