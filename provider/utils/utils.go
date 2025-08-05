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

func SetStringIfNotEmpty(d *schema.ResourceData, key string, setter func(*string)) {
	if v, ok := d.GetOk(key); ok {
		if s, valid := v.(string); valid && s != "" {
			setter(redis.String(s))
		}
	}
}

func SetIntIfPositive(d *schema.ResourceData, key string, setter func(*int)) {
	if v, ok := d.GetOk(key); ok {
		if i, valid := v.(int); valid && i > 0 {
			setter(redis.Int(i))
		}
	}
}

func SetInt(d *schema.ResourceData, key string, setter func(*int)) {
	if v, ok := d.GetOk(key); ok {
		if i, valid := v.(int); valid {
			setter(redis.Int(i))
		}
	}
}

func SetFloat64(d *schema.ResourceData, key string, setter func(*float64)) {
	if v, ok := d.GetOk(key); ok {
		if f, valid := v.(float64); valid {
			setter(redis.Float64(f))
		}
	}
}

func SetBool(d *schema.ResourceData, key string, setter func(*bool)) {
	if v, ok := d.GetOk(key); ok {
		if b, valid := v.(bool); valid {
			setter(redis.Bool(b))
		}
	}
}
