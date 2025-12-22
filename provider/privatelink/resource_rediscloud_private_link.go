package privatelink

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	pl "github.com/RedisLabs/rediscloud-go-api/service/privatelink"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func ResourceRedisCloudPrivateLink() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a Private Link to a pro Subscription in your Redis Enterprise Cloud Account.",
		CreateContext: resourceRedisCloudPrivateLinkCreate,
		ReadContext:   resourceRedisCloudPrivateLinkRead,
		UpdateContext: resourceRedisCloudPrivateLinkUpdate,
		DeleteContext: resourceRedisCloudPrivateLinkDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Read:   schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"subscription_id": {
				Description: "The ID of the Pro subscription to attach",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"share_name": {
				Description: "Name of this PrivateLink share",
				Type:        schema.TypeString,
				Required:    true,
			},
			"principal": {
				Description: "List of principals attached to this PrivateLink",
				Type:        schema.TypeSet,
				Required:    true,
				MinItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"principal": {
							Type:     schema.TypeString,
							Required: true,
						},
						"principal_type": {
							Type:             schema.TypeString,
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringMatch(regexp.MustCompile("^(aws_account|organization|organization_unit|iam_role|iam_user|service_principal)$"), "must be one of 'aws_account', 'organization', 'organization_unit', 'iam_role', 'iam_user', 'service_principal'")),
							Required:         true,
						},
						"principal_alias": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"resource_configuration_id": {
				Description: "ID of the resource configuration to attach to this PrivateLink",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"resource_configuration_arn": {
				Description: "ARN of the resource configuration attached to this PrivateLink",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"share_arn": {
				Description: "ARN of the share attached to this Private Link",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"connections": {
				Description: "Connections attached to this PrivateLink",
				Type:        schema.TypeSet,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"association_id": {
							Description: "Association ID of the PrivateLink connection",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"connection_id": {
							Description: "Connection ID of the PrivateLink connection",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"connection_type": {
							Description: "Connection type of the PrivateLink connection",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"owner_id": {
							Description: "Owner ID of the PrivateLink connection",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"association_date": {
							Description: "Date the connection was associated",
							Type:        schema.TypeString,
							Computed:    true,
						},
					},
				},
			},
			"databases": {
				Description: "The databases attached to this PrivateLink",
				Type:        schema.TypeSet,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"database_id": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"port": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"resource_link_endpoint": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func resourceRedisCloudPrivateLinkCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*client.ApiClient)

	subId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	utils.SubscriptionMutex.Lock(subId)
	defer utils.SubscriptionMutex.Unlock(subId)

	shareName := d.Get("share_name").(string)

	principals := principalsFromSet(d.Get("principal").(*schema.Set))
	firstPrincipal := principals[0]

	link := pl.CreatePrivateLink{
		Principal:      firstPrincipal.Principal,
		PrincipalType:  firstPrincipal.Type,
		PrincipalAlias: firstPrincipal.Alias,
		ShareName:      redis.String(shareName),
	}

	err = api.Client.PrivateLink.CreatePrivateLink(ctx, subId, link)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(subId))

	err = waitForPrivateLinkToBeActive(ctx, api, subId)

	if err != nil {
		return diag.FromErr(err)
	}

	err = createOtherPrincipals(ctx, api, subId, principals[1:])

	if err != nil {
		return diag.FromErr(err)
	}

	// TODO: figure out if this is necessary and remove/uncomment
	//err = waitForAllPrincipalsToBeAssociated(ctx, api, subId, principals)
	//if err != nil {
	//	return diag.FromErr(err)
	//}

	err = utils.WaitForSubscriptionToBeActive(ctx, subId, api)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceRedisCloudPrivateLinkRead(ctx, d, meta)
}

func resourceRedisCloudPrivateLinkRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*client.ApiClient)

	subId, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	privateLink, err := api.Client.PrivateLink.GetPrivateLink(ctx, subId)

	if err != nil {
		var notFound *pl.NotFound
		if errors.As(err, &notFound) {
			d.SetId("")
			return diags
		}
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(subId))

	err = d.Set("subscription_id", strconv.Itoa(subId))
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("share_name", privateLink.ShareName)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("principal", flattenPrincipals(privateLink.Principals)); err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("resource_configuration_id", privateLink.ResourceConfigurationId)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("resource_configuration_arn", privateLink.ResourceConfigurationArn)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("share_arn", privateLink.ShareArn)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("connections", flattenConnections(privateLink.Connections)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("databases", flattenDatabases(privateLink.Databases)); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceRedisCloudPrivateLinkUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*client.ApiClient)

	if d.HasChange("principal") {

		subId, err := strconv.Atoi(d.Get("subscription_id").(string))

		if err != nil {
			return diag.FromErr(err)
		}

		utils.SubscriptionMutex.Lock(subId)
		defer utils.SubscriptionMutex.Unlock(subId)

		privateLink, err := api.Client.PrivateLink.GetPrivateLink(ctx, subId)

		if err != nil {
			var notFound *pl.NotFound
			if errors.As(err, &notFound) {
				d.SetId("")
				return diags
			}
			return diag.FromErr(err)
		}

		apiPrincipals := privateLink.Principals
		tfPrincipals := principalsFromSet(d.Get("principal").(*schema.Set))

		principalsToCreate := findPrincipalsToCreate(apiPrincipals, tfPrincipals)
		err = createPrincipals(ctx, api, subId, principalsToCreate)

		if err != nil {
			return diag.FromErr(err)
		}

		principals := findPrincipalsToDelete(apiPrincipals, tfPrincipals)
		err = deletePrincipals(ctx, api, subId, principals)

		if err != nil {
			return diag.FromErr(err)
		}

	}

	return resourceRedisCloudPrivateLinkRead(ctx, d, meta)
}

func resourceRedisCloudPrivateLinkDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*client.ApiClient)

	subId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	utils.SubscriptionMutex.Lock(subId)
	defer utils.SubscriptionMutex.Unlock(subId)

	// direct delete doesn't exist on the API so delete each principal one by one
	privateLink, err := api.Client.PrivateLink.GetPrivateLink(ctx, subId)

	if err != nil {
		var notFound *pl.NotFound
		if errors.As(err, &notFound) {
			d.SetId("")
			return diags
		}
		return diag.FromErr(err)
	}

	for _, principal := range privateLink.Principals {
		err := api.Client.PrivateLink.DeletePrincipal(ctx, subId, *principal.Principal)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId("")

	err = utils.WaitForSubscriptionToBeActive(ctx, subId, api)
	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func createOtherPrincipals(ctx context.Context, api *client.ApiClient, subId int, otherPrincipals []pl.PrivateLinkPrincipal) error {
	if len(otherPrincipals) > 0 {
		for _, principal := range otherPrincipals {
			err := api.Client.PrivateLink.CreatePrincipal(ctx, subId, pl.CreatePrivateLinkPrincipal{
				Principal:      principal.Principal,
				PrincipalType:  principal.Type,
				PrincipalAlias: principal.Alias,
			})

			if err != nil {
				return err
			}
		}
	}
	return nil
}

func findPrincipalsToCreate(apiPrincipals []*pl.PrivateLinkPrincipal, tfPrincipals []pl.PrivateLinkPrincipal) []pl.CreatePrivateLinkPrincipal {
	var principals []pl.PrivateLinkPrincipal

	for _, tfPrincipal := range tfPrincipals {
		found := false
		for _, apiPrincipal := range apiPrincipals {
			if tfPrincipal.Principal == apiPrincipal.Principal {
				found = true
				break
			}
		}
		if !found {
			principals = append(principals, tfPrincipal)
		}
	}

	var createPrincipals []pl.CreatePrivateLinkPrincipal
	for _, principal := range principals {
		createPrincipal := pl.CreatePrivateLinkPrincipal{
			Principal:      principal.Principal,
			PrincipalType:  principal.Type,
			PrincipalAlias: principal.Alias,
		}
		createPrincipals = append(createPrincipals, createPrincipal)
	}

	return createPrincipals
}

func createPrincipals(ctx context.Context, api *client.ApiClient, subscriptionId int, principals []pl.CreatePrivateLinkPrincipal) error {
	for _, principal := range principals {
		err := api.Client.PrivateLink.CreatePrincipal(ctx, subscriptionId, principal)
		if err != nil {
			return err
		}
	}
	return nil
}

func findPrincipalsToDelete(apiPrincipals []*pl.PrivateLinkPrincipal, tfPrincipals []pl.PrivateLinkPrincipal) []pl.PrivateLinkPrincipal {
	var result []pl.PrivateLinkPrincipal

	for _, apiPrincipal := range apiPrincipals {
		found := false
		for _, tfPrincipal := range tfPrincipals {
			if apiPrincipal.Principal == tfPrincipal.Principal {
				found = true
				break
			}
		}
		if !found {
			result = append(result, *apiPrincipal)
		}
	}
	return result
}

func deletePrincipals(ctx context.Context, api *client.ApiClient, subscriptionId int, principals []pl.PrivateLinkPrincipal) error {
	for _, principal := range principals {
		err := api.Client.PrivateLink.DeletePrincipal(ctx, subscriptionId, *principal.Principal)
		if err != nil {
			return err
		}
	}
	return nil
}
