package psc

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	pscapi "github.com/RedisLabs/rediscloud-go-api/service/psc"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
)

// ServiceAttachmentModel represents a GCP service attachment created for a Private Service Connect endpoint.
type ServiceAttachmentModel struct {
	Name               types.String `tfsdk:"name"`
	DNSRecord          types.String `tfsdk:"dns_record"`
	IPAddressName      types.String `tfsdk:"ip_address_name"`
	ForwardingRuleName types.String `tfsdk:"forwarding_rule_name"`
}

var serviceAttachmentElemType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(ServiceAttachmentModel{})}

// ServiceAttachmentsFromAPI maps service attachments as the SDKv2 implementation did, writing "" for missing fields.
func ServiceAttachmentsFromAPI(ctx context.Context, attachments []pscapi.TerraformGCPServiceAttachment) (types.List, diag.Diagnostics) {
	models := make([]ServiceAttachmentModel, 0, len(attachments))
	for _, attachment := range attachments {
		//TODO(TF3.0) make service attachment fields nullable
		models = append(models, ServiceAttachmentModel{
			Name:               types.StringValue(redis.StringValue(attachment.Name)),
			DNSRecord:          types.StringValue(redis.StringValue(attachment.DNSRecord)),
			IPAddressName:      types.StringValue(redis.StringValue(attachment.IPAddressName)),
			ForwardingRuleName: types.StringValue(redis.StringValue(attachment.ForwardingRuleName)),
		})
	}
	return types.ListValueFrom(ctx, serviceAttachmentElemType, models)
}

func buildPrivateServiceConnectId(subId int, pscServiceId int) string {
	return fmt.Sprintf("%d/%d", subId, pscServiceId)
}

const placeholderStatusDisappear = "disappeared"

type endpointsGetter func(context.Context) (*pscapi.PrivateServiceConnectEndpoints, error)

func findEndpoint(id int, endpoints []*pscapi.PrivateServiceConnectEndpoint) *pscapi.PrivateServiceConnectEndpoint {
	for _, endpoint := range endpoints {
		if redis.IntValue(endpoint.ID) == id {
			return endpoint
		}
	}
	return nil
}

func hasServiceAttachments(endpoint *pscapi.PrivateServiceConnectEndpoint) bool {
	status := redis.StringValue(endpoint.Status)
	return status != pscapi.EndpointStatusRejected && status != pscapi.EndpointStatusDeleted
}

// waitForEndpointToDisappear waits for an endpoint that the API cannot delete
// directly: it goes away once the related GCP resources are removed, which this
// provider cannot check.
func waitForEndpointToDisappear(ctx context.Context, api *client.ApiClient, getEndpoints endpointsGetter, endpointId int, timeout time.Duration) error {
	wait := &retry.StateChangeConf{
		Pending: []string{
			pscapi.EndpointStatusProcessing,
			pscapi.EndpointStatusPending,
			pscapi.EndpointStatusAcceptPending,
			pscapi.EndpointStatusActive,
			pscapi.EndpointStatusDeleted,
			pscapi.EndpointStatusRejected,
			pscapi.EndpointStatusRejectPending,
			pscapi.EndpointStatusFailed,
		},
		Target:       []string{placeholderStatusDisappear},
		Timeout:      timeout,
		Delay:        api.WaitDelay(10 * time.Second),
		PollInterval: api.WaitPollInterval(30 * time.Second),
		Refresh: func() (interface{}, string, error) {
			endpoints, err := getEndpoints(ctx)
			if err != nil {
				return nil, "", err
			}
			endpoint := findEndpoint(endpointId, endpoints.Endpoints)
			if endpoint == nil {
				return placeholderStatusDisappear, placeholderStatusDisappear, nil
			}
			return redis.StringValue(endpoint.Status), redis.StringValue(endpoint.Status), nil
		},
	}
	_, err := wait.WaitForStateContext(ctx)
	return err
}

func parseIntIdParts(id string, count int) ([]int, error) {
	parts := strings.Split(id, "/")
	if len(parts) != count {
		return nil, fmt.Errorf("invalid id: %s", id)
	}
	values := make([]int, count)
	for i, part := range parts {
		value, err := strconv.Atoi(part)
		if err != nil {
			return nil, err
		}
		values[i] = value
	}
	return values, nil
}

type creationScriptGetter func(ctx context.Context, endpointId int) (*pscapi.CreationScript, error)

// readEndpoint returns the endpoint and its service attachments, or nil when
// the endpoint or its Private Service Connect service no longer exists.
func readEndpoint(ctx context.Context, getEndpoints endpointsGetter, getCreationScript creationScriptGetter, endpointId int) (*pscapi.PrivateServiceConnectEndpoint, types.List, error) {
	noAttachments := types.ListNull(serviceAttachmentElemType)

	endpoints, err := getEndpoints(ctx)
	if err != nil {
		if isNotFound(err) {
			return nil, noAttachments, nil
		}
		return nil, noAttachments, err
	}

	endpoint := findEndpoint(endpointId, endpoints.Endpoints)
	if endpoint == nil {
		return nil, noAttachments, nil
	}

	var apiAttachments []pscapi.TerraformGCPServiceAttachment
	if hasServiceAttachments(endpoint) {
		script, err := getCreationScript(ctx, redis.IntValue(endpoint.ID))
		if err != nil {
			if isNotFound(err) {
				return nil, noAttachments, nil
			}
			return nil, noAttachments, err
		}
		apiAttachments = script.Script.TerraformGcp.ServiceAttachments
	}

	attachments, diags := ServiceAttachmentsFromAPI(ctx, apiAttachments)
	if diags.HasError() {
		return nil, noAttachments, fmt.Errorf("mapping service attachments: %v", diags)
	}
	return endpoint, attachments, nil
}

func isNotFound(err error) bool {
	var notFound *pscapi.NotFound
	var notFoundActiveActive *pscapi.NotFoundActiveActive
	return errors.As(err, &notFound) || errors.As(err, &notFoundActiveActive)
}
