package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &InstanceIPResource{}
	_ resource.ResourceWithImportState = &InstanceIPResource{}
)

type instanceIPResourceModel struct {
	ID         types.String `tfsdk:"id"`
	InstanceID types.String `tfsdk:"instance_id"`
	IPAddress  types.String `tfsdk:"ip_address"`
}

type InstanceIPResource struct {
	client *Client
}

func NewInstanceIPResource() resource.Resource {
	return &InstanceIPResource{}
}

func (r *InstanceIPResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_ip"
}

func (r *InstanceIPResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage an additional IP attached to a Blue Lobster instance.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"instance_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"ip_address": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *InstanceIPResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureClient(req.ProviderData, &resp.Diagnostics, &r.client)
}

func (r *InstanceIPResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan instanceIPResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	before, err := r.client.ListInstanceIPs(ctx, plan.InstanceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Blue Lobster instance IPs before create", err.Error())
		return
	}
	if err := r.client.AssignInstanceIP(ctx, plan.InstanceID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to assign Blue Lobster instance IP", err.Error())
		return
	}
	ip, err := r.client.WaitForNewIP(ctx, plan.InstanceID.ValueString(), before)
	if err != nil {
		resp.Diagnostics.AddError("Unable to observe new Blue Lobster instance IP", err.Error())
		return
	}

	plan.IPAddress = types.StringValue(ip)
	plan.ID = types.StringValue(fmt.Sprintf("%s,%s", plan.InstanceID.ValueString(), ip))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *InstanceIPResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state instanceIPResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ips, err := r.client.ListInstanceIPs(ctx, state.InstanceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Blue Lobster instance IPs", err.Error())
		return
	}
	currentIP := strings.TrimSpace(state.IPAddress.ValueString())
	for _, ip := range ips {
		if ip == currentIP {
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *InstanceIPResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {
}

func (r *InstanceIPResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state instanceIPResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.ReleaseInstanceIP(ctx, state.InstanceID.ValueString(), state.IPAddress.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to release Blue Lobster instance IP", err.Error())
	}
}

func (r *InstanceIPResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ",", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "Use `<instance_id>,<ip_address>`.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("ip_address"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
