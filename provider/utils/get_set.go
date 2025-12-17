package utils

import (
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// This timeout is an absolute maximum used in some of the waitForStatus operations concerning creation and updating
// Subscriptions and Databases. Reads and Deletions have their own, stricter timeouts because they consistently behave
// well. The Terraform operation-level timeout should kick in way before we hit this and kill the task.
// Unfortunately there's no "time-remaining-before-timeout" utility, or we could use that in the wait blocks.
const SafetyTimeout = 6 * time.Hour

// TransitGatewayProvisioningTimeout is used when waiting for Transit Gateway resources to become available during
// subscription provisioning. This is shorter than SafetyTimeout as tests typically complete within 45 minutes.
const TransitGatewayProvisioningTimeout = 40 * time.Minute

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
