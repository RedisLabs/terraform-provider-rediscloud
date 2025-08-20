package acl

import (
	"context"
	"fmt"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/access_control_lists/users"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"log"
	"strconv"
	"time"
)

func ResourceRedisCloudAclUser() *schema.Resource {
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
	api := meta.(*utils.ApiClient)

	name := d.Get("name").(string)
	role := d.Get("role").(string)
	password := d.Get("password").(string)

	createUser := users.CreateUserRequest{
		Name:     redis.String(name),
		Role:     redis.String(role),
		Password: redis.String(password),
	}

	id, err := api.Client.Users.Create(ctx, createUser)
	if err != nil {
		return diag.FromErr(err)
	}

	err = waitForAclUserToBeActive(ctx, id, api)
	if err != nil {
		return diag.FromErr(err)
	}

	// Sometimes ACL Users and Roles flip between Active and Pending a few times after creation/update.
	// This delay gives the API a chance to settle
	// TODO Ultimately this is an API problem
	time.Sleep(15 * time.Second) //lintignore:R018

	err = waitForAclUserToBeActive(ctx, id, api)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(id))

	return resourceRedisCloudAclUserRead(ctx, d, meta)
}

func resourceRedisCloudAclUserRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*utils.ApiClient)
	var diags diag.Diagnostics

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	user, err := api.Client.Users.Get(ctx, id)
	if err != nil {
		if _, ok := err.(*users.NotFound); ok {
			d.SetId("")
			return diags
		}
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
	api := meta.(*utils.ApiClient)

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

		err = api.Client.Users.Update(ctx, id, updateUserRequest)
		if err != nil {
			return diag.FromErr(err)
		}

		err = waitForAclUserToBeActive(ctx, id, api)
		if err != nil {
			return diag.FromErr(err)
		}

		// Sometimes ACL Users and Roles flip between Active and Pending a few times after creation/update.
		// This delay gives the API a chance to settle
		// TODO Ultimately this is an API problem
		time.Sleep(15 * time.Second) //lintignore:R018

		err = waitForAclUserToBeActive(ctx, id, api)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceRedisCloudAclUserRead(ctx, d, meta)
}

func resourceRedisCloudAclUserDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*utils.ApiClient)
	var diags diag.Diagnostics

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Sometimes ACL Users and Roles flip between Active and Pending a few times after creation/update.
	// This delay gives the API a chance to settle
	// TODO Ultimately this is an API problem
	err = waitForAclUserToBeActive(ctx, id, api)
	if err != nil {
		return diag.FromErr(err)
	}

	err = api.Client.Users.Delete(ctx, id)

	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	// Wait until it's really disappeared
	err = retry.RetryContext(ctx, 5*time.Minute, func() *retry.RetryError {
		user, err := api.Client.Users.Get(ctx, id)

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

func waitForAclUserToBeActive(ctx context.Context, id int, api *utils.ApiClient) error {
	wait := &retry.StateChangeConf{
		Delay:   5 * time.Second,
		Pending: []string{users.StatusPending},
		Target:  []string{users.StatusActive},
		Timeout: 5 * time.Minute,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for user %d to be active", id)

			user, err := api.Client.Users.Get(ctx, id)
			if err != nil {
				return nil, "", err
			}

			return redis.StringValue(user.Status), redis.StringValue(user.Status), nil
		},
	}
	if _, err := wait.WaitForStateContext(ctx); err != nil {
		return err
	}

	return nil
}
