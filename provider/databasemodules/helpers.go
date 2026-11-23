package databasemodules

import (
	"context"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/account"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ModulesFromAPI maps the account database modules to the computed modules set.
func ModulesFromAPI(ctx context.Context, apiModules []*account.DatabaseModule) (types.Set, diag.Diagnostics) {
	models := make([]ModuleModel, 0, len(apiModules))
	for _, m := range apiModules {
		if m == nil {
			continue
		}
		models = append(models, ModuleModel{
			Name:        types.StringValue(redis.StringValue(m.Name)),
			Description: types.StringValue(redis.StringValue(m.Description)),
		})
	}
	return types.SetValueFrom(ctx, ModuleElemType, models)
}
