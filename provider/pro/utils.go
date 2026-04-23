package pro

import (
	"context"
	"strings"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	redisTags "github.com/RedisLabs/rediscloud-go-api/service/tags"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

func ReadTags(ctx context.Context, api *client.ApiClient, subId int, databaseId int, d *schema.ResourceData) error {
	tags := make(map[string]string)
	tagResponse, err := api.Client.Tags.Get(ctx, subId, databaseId)
	if err != nil {
		return err
	}
	if tagResponse.Tags != nil {
		for _, t := range *tagResponse.Tags {
			tags[redis.StringValue(t.Key)] = redis.StringValue(t.Value)
		}
	}
	return d.Set("tags", tags)
}

func WriteTags(ctx context.Context, api *client.ApiClient, subId int, databaseId int, d *schema.ResourceData) error {
	tags := make([]*redisTags.Tag, 0)
	tState := d.Get("tags").(map[string]interface{})
	for k, v := range tState {
		tags = append(tags, &redisTags.Tag{
			Key:   redis.String(k),
			Value: redis.String(v.(string)),
		})
	}
	return api.Client.Tags.Put(ctx, subId, databaseId, redisTags.AllTags{Tags: &tags})
}

func ValidateTagsfunc(tagsRaw interface{}, _ cty.Path) diag.Diagnostics {
	tags := tagsRaw.(map[string]interface{})
	invalid := make([]string, 0)
	for k, v := range tags {
		if k != strings.ToLower(k) {
			invalid = append(invalid, k)
		}
		vStr := v.(string)
		if vStr != strings.ToLower(vStr) {
			invalid = append(invalid, vStr)
		}
	}

	if len(invalid) > 0 {
		return diag.Errorf("tag keys and values must be lower case, invalid entries: %s", strings.Join(invalid, ", "))
	}
	return nil
}
