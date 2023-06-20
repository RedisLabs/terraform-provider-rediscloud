package provider

import (
	"fmt"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"time"
)

func setToStringSlice(set *schema.Set) []*string {
	var ret []*string
	for _, s := range set.List() {
		ret = append(ret, redis.String(s.(string)))
	}
	return ret
}

func interfaceToStringSlice(list []interface{}) []*string {
	var ret []*string
	for _, i := range list {
		ret = append(ret, redis.String(i.(string)))
	}
	return ret
}

// IDs of any resources dependent on a subscription need to be divided by a slash. In this format: <sub id>/<resource id>.
func buildResourceId(subId int, id int) string {
	return fmt.Sprintf("%d/%d", subId, id)
}

func isTime() schema.SchemaValidateDiagFunc {
	return func(i interface{}, path cty.Path) diag.Diagnostics {
		var diags diag.Diagnostics

		v, ok := i.(string)
		if !ok {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Value not a string",
				Detail:   fmt.Sprintf("Value should be a string rather than %T", i),
			})
		} else if _, err := time.Parse("15:04", v); err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Value is not a time",
				Detail:   fmt.Sprintf("Value should be a valid time, got: %q: %s", i, err),
			})
		}

		return diags
	}
}
