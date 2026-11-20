package iris

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

var (
	_ resource.Resource                = (*serviceResource)(nil)
	_ resource.ResourceWithConfigure   = (*serviceResource)(nil)
	_ resource.ResourceWithImportState = (*serviceResource)(nil)
)

type serviceResource struct {
	definition   definition
	client       *client.IrisServiceClient
	pollInterval time.Duration
}

// Resources supplies the direct-service resources separately from the CAPI family.
func Resources() []func() resource.Resource {
	out := make([]func() resource.Resource, 0, len(definitions))
	for _, d := range definitions {
		out = append(out, func() resource.Resource { return &serviceResource{definition: d, pollInterval: 5 * time.Second} })
	}
	return out
}
func (r *serviceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.definition.Name
}
func (r *serviceResource) schema(ctx context.Context) schema.Schema {
	s := generatedSchemas[r.definition.Name](ctx)
	s.Description = "Manages a direct Iris " + r.definition.Service + " resource, independently of Redis Cloud CAPI."
	for _, f := range r.definition.Fields {
		a, ok := s.Attributes[f.Name].(schema.StringAttribute)
		if !ok {
			continue
		}
		if f.Wire == "destinationNamespaceId" {
			a.Validators = []validator.String{namespaceValidator{}}
			a.Description = "Destination namespace identifier. Use an empty string to clear it."
			a.MarkdownDescription = a.Description
		}
		if f.JSON && (f.Create || f.Path) {
			a.Validators = append(a.Validators, jsonShapeValidator{shape: f.Shape})
		}
		if f.Path {
			a.Validators = append(a.Validators, pathComponentValidator{tenant: f.Wire == "tenant"})
		}
		s.Attributes[f.Name] = a
	}
	return s
}
func (r *serviceResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = r.schema(ctx)
}
func (r *serviceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.ApiClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider client", fmt.Sprintf("Expected *client.ApiClient, got %T", req.ProviderData))
		return
	}
	r.client = c.Iris[r.definition.Service]
	if r.client == nil {
		resp.Diagnostics.AddError("Missing Iris service configuration", "Configure iris_"+strings.ReplaceAll(r.definition.Service, "-", "_")+"_url and its bearer token for this resource.")
	}
}
func (r *serviceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	var plan types.Object
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	values := plan.Attributes()
	var configuration types.Object
	resp.Diagnostics.Append(req.Config.Get(ctx, &configuration)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body := r.body(configuration.Attributes(), false, nil, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	endpoint, err := r.endpoint(r.definition.Create, values)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Iris resource path", err.Error())
		return
	}
	created, err := r.client.Call(ctx, r.definition.Create.Method, endpoint, nil, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Iris resource", err.Error())
		return
	}
	id, ok := created[r.definition.IDField].(string)
	if !ok || id == "" {
		resp.Diagnostics.AddError("Missing Iris resource identifier", "The create response omitted its identifier. The resource may have been created; inspect the service before retrying.")
		return
	}
	values["id"] = types.StringValue(id)
	for _, f := range r.definition.Fields {
		if f.Wire == r.definition.IDField {
			values[f.Name] = types.StringValue(id)
		}
	}
	r.apply(ctx, values, created, false, &resp.Diagnostics)
	// Record ownership before the first read or readiness wait can fail.
	r.save(ctx, values, r.schema(ctx).Type().(types.ObjectType), &resp.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	current, err := r.refresh(ctx, values, true)
	if err != nil {
		if current != nil {
			r.apply(ctx, values, current, true, &resp.Diagnostics)
			r.save(ctx, values, r.schema(ctx).Type().(types.ObjectType), &resp.State, &resp.Diagnostics)
		}
		resp.Diagnostics.AddError("Failed to read created Iris resource", err.Error())
		return
	}
	r.apply(ctx, values, current, true, &resp.Diagnostics)
	r.save(ctx, values, r.schema(ctx).Type().(types.ObjectType), &resp.State, &resp.Diagnostics)
}
func (r *serviceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	var state types.Object
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	values := state.Attributes()
	current, err := r.refresh(ctx, values, false)
	if client.IrisNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Iris resource", err.Error())
		return
	}
	r.apply(ctx, values, current, true, &resp.Diagnostics)
	r.save(ctx, values, r.schema(ctx).Type().(types.ObjectType), &resp.State, &resp.Diagnostics)
}
func (r *serviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	var plan, state types.Object
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	values := plan.Attributes()
	old := state.Attributes()
	// Carry create-only secrets through an update and a subsequent read.
	for name, a := range r.schema(ctx).Attributes {
		if a.IsComputed() && !a.IsOptional() {
			if v := values[name]; v == nil || v.IsUnknown() {
				values[name] = old[name]
			}
		}
	}
	if r.definition.Update.Path == "" {
		resp.Diagnostics.AddError("Unsupported Iris update", "This resource requires replacement for changes.")
		return
	}
	// Read the revision at refresh time from Terraform state, never from the unknown plan.
	var configuration types.Object
	resp.Diagnostics.Append(req.Config.Get(ctx, &configuration)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body := r.body(configuration.Attributes(), true, old, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	endpoint, err := r.endpoint(r.definition.Update, values)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Iris resource path", err.Error())
		return
	}
	updated, err := r.client.Call(ctx, r.definition.Update.Method, endpoint, nil, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update Iris resource", err.Error())
		return
	}
	r.apply(ctx, values, updated, false, &resp.Diagnostics)
	r.save(ctx, values, r.schema(ctx).Type().(types.ObjectType), &resp.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	current, err := r.refresh(ctx, values, true)
	if err != nil {
		if current != nil {
			r.apply(ctx, values, current, true, &resp.Diagnostics)
			r.save(ctx, values, r.schema(ctx).Type().(types.ObjectType), &resp.State, &resp.Diagnostics)
		}
		resp.Diagnostics.AddError("Failed to read updated Iris resource", err.Error())
		return
	}
	r.apply(ctx, values, current, true, &resp.Diagnostics)
	r.save(ctx, values, r.schema(ctx).Type().(types.ObjectType), &resp.State, &resp.Diagnostics)
}
func (r *serviceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	var state types.Object
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	values := state.Attributes()
	endpoint, err := r.endpoint(r.definition.Delete, values)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Iris resource path", err.Error())
		return
	}
	query := url.Values{}
	if r.definition.Name == "iris_agent_memory" {
		flush, _ := values["flush_on_delete"].(types.Bool)
		query.Set("flush", fmt.Sprint(flush.ValueBool()))
	}
	if r.definition.DeleteRevision {
		revision, _ := values["revision"].(types.String)
		if revision.IsNull() || revision.IsUnknown() {
			resp.Diagnostics.AddError("Missing Iris revision", "Refresh the resource before deleting it.")
			return
		}
		query.Set("expectedRevision", revision.ValueString())
	}
	_, err = r.client.Call(ctx, r.definition.Delete.Method, endpoint, query, nil)
	if err != nil && !client.IrisNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete Iris resource", err.Error())
	}
}
func (r *serviceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parents := r.definition.parentFields()
	parts := strings.Split(req.ID, "/")
	if len(parts) != len(parents)+1 {
		resp.Diagnostics.AddError("Invalid Iris import identifier", fmt.Sprintf("Expected %d slash-separated path components (parent identifiers followed by the resource identifier).", len(parents)+1))
		return
	}
	for _, part := range parts {
		if err := validatePathComponent(part, false); err != nil {
			resp.Diagnostics.AddError("Invalid Iris import identifier", err.Error())
			return
		}
	}
	for i, f := range parents {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(f.Name), parts[i])...)
	}
	id := parts[len(parts)-1]
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
	for _, f := range r.definition.Fields {
		if f.Wire == r.definition.IDField {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(f.Name), id)...)
		}
	}
}
func (r *serviceResource) endpoint(op operation, values map[string]attr.Value) (string, error) {
	result := op.Path
	for strings.Contains(result, "{") {
		start := strings.Index(result, "{")
		end := strings.Index(result[start:], "}") + start
		if end < start {
			return "", fmt.Errorf("invalid generated path")
		}
		wire := result[start+1 : end]
		name := r.definition.pathField(wire)
		value, ok := values[name].(types.String)
		if !ok || value.IsNull() || value.IsUnknown() {
			return "", fmt.Errorf("%s must be known", name)
		}
		if err := validatePathComponent(value.ValueString(), wire == "tenant"); err != nil {
			return "", err
		}
		result = result[:start] + url.PathEscape(value.ValueString()) + result[end+1:]
	}
	return result, nil
}
func (r *serviceResource) body(values map[string]attr.Value, update bool, old map[string]attr.Value, diags *diag.Diagnostics) map[string]any {
	body := map[string]any{}
	for _, f := range r.definition.Fields {
		if (!update && !f.Create) || (update && !f.Update) {
			continue
		}
		value := values[f.Name]
		if value == nil || value.IsNull() {
			continue
		}
		if value.IsUnknown() {
			diags.AddError("Unknown Iris configuration", f.Name+" must be known before apply.")
			continue
		}
		if update && r.definition.Update.Method == "PATCH" {
			if prev := old[f.Name]; prev != nil && prev.Equal(value) {
				continue
			}
		}
		wire, err := toWire(value, f.JSON)
		if err != nil {
			diags.AddError("Invalid Iris configuration", f.Name+": "+err.Error())
			continue
		}
		if !update && f.Wire == "destinationNamespaceId" && wire == "" {
			continue
		}
		body[f.Wire] = wire
	}
	if update && r.definition.Revision {
		v, _ := old["revision"].(types.String)
		if v.IsNull() || v.IsUnknown() {
			diags.AddError("Missing Iris revision", "Refresh the resource before updating it.")
		} else {
			body["expectedRevision"] = v.ValueString()
		}
	}
	return body
}
func (r *serviceResource) refresh(ctx context.Context, values map[string]attr.Value, wait bool) (map[string]any, error) {
	endpoint, err := r.endpoint(r.definition.Read, values)
	if err != nil {
		return nil, err
	}
	fetch := func() (map[string]any, error) {
		current, e := r.client.Call(ctx, r.definition.Read.Method, endpoint, nil, nil)
		if e != nil {
			return nil, e
		}
		expected, _ := values["id"].(types.String)
		actual, _ := current[r.definition.IDField].(string)
		if actual == "" || actual != expected.ValueString() {
			return nil, fmt.Errorf("iris read response did not match the managed resource identifier")
		}
		return current, nil
	}
	if !wait || r.definition.Name != "iris_agent_memory" {
		return fetch()
	}
	var last map[string]any
	waiter := retry.StateChangeConf{Pending: []string{"PROVISIONING"}, Target: []string{"READY"}, Timeout: 30 * time.Minute, PollInterval: r.pollInterval, Refresh: func() (any, string, error) {
		current, e := fetch()
		if e != nil {
			return nil, "", e
		}
		last = current
		state, _ := current["state"].(string)
		return current, state, nil
	}}
	result, err := waiter.WaitForStateContext(ctx)
	if err != nil {
		return last, err
	}
	return result.(map[string]any), nil
}
func (r *serviceResource) apply(ctx context.Context, values map[string]attr.Value, current map[string]any, full bool, diags *diag.Diagnostics) {
	s := r.schema(ctx)
	for _, f := range r.definition.Fields {
		raw, exists := current[f.Wire]
		previous := values[f.Name]
		if exists && f.Path && f.Wire != r.definition.IDField {
			if expected, ok := previous.(types.String); ok && !expected.IsNull() && !expected.IsUnknown() && raw != expected.ValueString() {
				diags.AddError("Invalid Iris response identity", "The response did not match the managed parent identifier.")
				continue
			}
		}
		if !exists {
			if f.Read && !f.Path && (full || !f.Create) {
				values[f.Name] = nullValue(ctx, s.Attributes[f.Name].GetType())
				if f.Wire == "expiresAt" {
					values[f.Name] = types.Int64Value(0)
				}
				if f.Wire == "destinationNamespaceId" {
					values[f.Name] = types.StringValue("")
				}
				if f.JSON && f.Shape["type"] == "array" {
					values[f.Name] = types.StringValue("[]")
					if old, ok := previous.(types.String); ok && equalJSON(old.ValueString(), "[]") {
						values[f.Name] = old
					}
				}
			}
			continue
		}
		if f.JSON {
			template, _ := previous.(types.String)
			if !template.IsNull() && !template.IsUnknown() {
				configured, err := decodeJSON(template.ValueString())
				if err == nil {
					raw = projectJSON(configured, raw, f.ReadShape)
				}
			}
		}
		converted, e := fromWire(ctx, raw, s.Attributes[f.Name].GetType(), f.JSON)
		if e != nil {
			diags.AddError("Invalid Iris response", f.Name+": "+e.Error())
			continue
		}
		if f.JSON {
			before, bok := previous.(types.String)
			after, aok := converted.(types.String)
			if bok && aok && equalJSON(before.ValueString(), after.ValueString()) {
				converted = before
			}
		}
		values[f.Name] = converted
	}
}
