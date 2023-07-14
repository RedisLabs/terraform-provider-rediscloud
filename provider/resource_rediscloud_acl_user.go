package provider

import (
	"context"
	"fmt"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/access_control_lists/users"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"strconv"
	"time"
)

func resourceRedisCloudAclUser() *schema.Resource {
	return &schema.Resource{
		Description:   "Create an ACL User within your Redis Enterprise Cloud Account",
		CreateContext: resourceRedisCloudAclUserCreate,
		ReadContext:   resourceRedisCloudAclUserRead,
		UpdateContext: resourceRedisCloudAclUserUpdate,
		DeleteContext: resourceRedisCloudAclUserDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Read:   schema.DefaultTimeout(3 * time.Minute),
			Update: schema.DefaultTimeout(5 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "A meaningful name to identify the user",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"role": {
				Description: "The role which the user has",
				Type:        schema.TypeString,
				Required:    true,
			},
			"password": {
				Description: "The user's password",
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
				Sensitive:   true,
			},
		},
	}
}

func resourceRedisCloudAclUserCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)
	var diags diag.Diagnostics

	name := d.Get("name").(string)
	role := d.Get("role").(string)
	password := d.Get("password").(string)

	createUser := users.CreateUserRequest{
		Name:     redis.String(name),
		Role:     redis.String(role),
		Password: redis.String(password),
	}

	id, err := api.client.Users.Create(ctx, createUser)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(id))

	return diags
}

func resourceRedisCloudAclUserRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)
	var diags diag.Diagnostics

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	user, err := api.client.Users.Get(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("name", redis.StringValue(user.Name)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("role", redis.StringValue(user.Role)); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceRedisCloudAclUserUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)
	var diags diag.Diagnostics

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChanges("role", "password") {
		updateUserRequest := users.UpdateUserRequest{}

		role := d.Get("role").(string)
		updateUserRequest.Role = &role
		password := d.Get("password").(string)
		updateUserRequest.Password = &password

		err = api.client.Users.Update(ctx, id, updateUserRequest)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}

func resourceRedisCloudAclUserDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)
	var diags diag.Diagnostics

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = api.client.Users.Delete(ctx, id)

	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	err = retry.RetryContext(ctx, 5*time.Minute, func() *retry.RetryError {
		user, err := api.client.Users.Get(ctx, id)

		if err != nil {
			if _, ok := err.(*users.NotFound); ok {
				// All good, the resource is gone
				return nil
			}
			// This was an unexpected error
			return retry.NonRetryableError(fmt.Errorf("error getting user: %s", err))
		}

		if user != nil {
			return retry.RetryableError(fmt.Errorf("expected user %d to be deleted but was not", id))
		}
		// Unclear at this point what's going on!
		return retry.NonRetryableError(fmt.Errorf("unexpected error getting user"))
	})
	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}
