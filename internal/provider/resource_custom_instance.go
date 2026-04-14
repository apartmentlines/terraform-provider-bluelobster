package provider

import (
	"context"
	"errors"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &CustomInstanceResource{}
	_ resource.ResourceWithImportState = &CustomInstanceResource{}
)

type customInstanceResourceModel struct {
	instanceResourceModel
	Host       types.String `tfsdk:"host"`
	Cores      types.Int64  `tfsdk:"cores"`
	MemorySize types.Int64  `tfsdk:"memory_size"`
	DiskSize   types.Int64  `tfsdk:"disk_size"`
	GPUCountIn types.Int64  `tfsdk:"gpu_count_input"`
	GPUModelIn types.String `tfsdk:"gpu_model_input"`
}

type CustomInstanceResource struct {
	client *Client
}

func NewCustomInstanceResource() resource.Resource {
	return &CustomInstanceResource{}
}

func (r *CustomInstanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_instance"
}

func (r *CustomInstanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := sharedInstanceAttributes()
	attrs["host"] = schema.StringAttribute{
		Required:      true,
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	attrs["instance_type"] = schema.StringAttribute{
		Required:      true,
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	attrs["cores"] = schema.Int64Attribute{
		Required:      true,
		PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
	}
	attrs["memory_size"] = schema.Int64Attribute{
		Required:      true,
		PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
	}
	attrs["disk_size"] = schema.Int64Attribute{
		Required:      true,
		PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
	}
	attrs["gpu_count_input"] = schema.Int64Attribute{
		Required:      true,
		PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
	}
	attrs["gpu_model_input"] = schema.StringAttribute{
		Required:      true,
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage a custom Blue Lobster instance launched against a specific host.",
		Attributes:          attrs,
	}
}

func (r *CustomInstanceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureClient(req.ProviderData, &resp.Diagnostics, &r.client)
}

func (r *CustomInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan customInstanceResourceModel
	var sshKey, password types.String

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("ssh_public_key_wo"), &sshKey)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("password_wo"), &password)...)
	if resp.Diagnostics.HasError() {
		return
	}

	metadata, diags := buildStringMap(ctx, plan.Metadata)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(validateCustomInstancePlan(plan, sshKey.ValueString(), password.ValueString())...)
	if resp.Diagnostics.HasError() {
		return
	}

	launch, err := r.client.LaunchCustomInstance(ctx, LaunchCustomInstanceInput{
		Name:         strings.TrimSpace(plan.Name.ValueString()),
		InstanceType: strings.TrimSpace(plan.InstanceType.ValueString()),
		Host:         strings.TrimSpace(plan.Host.ValueString()),
		Cores:        plan.Cores.ValueInt64(),
		MemoryGB:     plan.MemorySize.ValueInt64(),
		DiskSizeGB:   plan.DiskSize.ValueInt64(),
		GPUCount:     plan.GPUCountIn.ValueInt64(),
		GPUModel:     strings.TrimSpace(plan.GPUModelIn.ValueString()),
		Username:     strings.TrimSpace(plan.Username.ValueString()),
		SSHPublicKey: strings.TrimSpace(sshKey.ValueString()),
		Password:     strings.TrimSpace(password.ValueString()),
		Metadata:     metadata,
		TemplateName: strings.TrimSpace(plan.TemplateName.ValueString()),
		ISOURL:       strings.TrimSpace(plan.ISOURL.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Blue Lobster custom instance", err.Error())
		return
	}

	instanceID := launch.InstanceID
	if instanceID == "" {
		task, err := r.client.WaitForTask(ctx, launch.TaskID)
		if err != nil {
			resp.Diagnostics.AddError("Unable to resolve created Blue Lobster custom instance", err.Error())
			return
		}
		instanceID, err = extractInstanceIDFromTask(task)
		if err != nil {
			resp.Diagnostics.AddError("Unable to resolve created Blue Lobster custom instance", err.Error())
			return
		}
	}

	remote, err := r.client.WaitForInstanceVisible(ctx, instanceID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to observe created Blue Lobster custom instance", err.Error())
		return
	}
	remote, err = reconcileDesiredPowerState(ctx, r.client, remote, normalizeDesiredPowerState(plan.PowerState))
	if err != nil {
		resp.Diagnostics.AddError("Unable to reconcile Blue Lobster custom instance power state", err.Error())
		return
	}

	syncCustomInstanceModel(&plan, remote)
	plan.SSHPublicKeyWO = types.StringNull()
	plan.PasswordWO = types.StringNull()
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CustomInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state customInstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	remote, err := r.client.GetInstance(ctx, state.ID.ValueString())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read Blue Lobster custom instance", err.Error())
		return
	}
	syncCustomInstanceModel(&state, remote)
	state.SSHPublicKeyWO = types.StringNull()
	state.PasswordWO = types.StringNull()
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *CustomInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state customInstanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if strings.TrimSpace(plan.Name.ValueString()) != strings.TrimSpace(state.Name.ValueString()) && strings.TrimSpace(plan.Name.ValueString()) != "" {
		if err := r.client.RenameInstance(ctx, state.ID.ValueString(), strings.TrimSpace(plan.Name.ValueString())); err != nil {
			resp.Diagnostics.AddError("Unable to rename Blue Lobster custom instance", err.Error())
			return
		}
	}
	remote, err := r.client.GetInstance(ctx, state.ID.ValueString())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read Blue Lobster custom instance before update", err.Error())
		return
	}
	remote, err = reconcileDesiredPowerState(ctx, r.client, remote, normalizeDesiredPowerState(plan.PowerState))
	if err != nil {
		resp.Diagnostics.AddError("Unable to reconcile Blue Lobster custom instance power state", err.Error())
		return
	}
	syncCustomInstanceModel(&plan, remote)
	plan.SSHPublicKeyWO = types.StringNull()
	plan.PasswordWO = types.StringNull()
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CustomInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state customInstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteInstance(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to delete Blue Lobster custom instance", err.Error())
		return
	}
	if err := r.client.WaitForDeletion(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to confirm Blue Lobster custom instance deletion", err.Error())
	}
}

func (r *CustomInstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func validateCustomInstancePlan(plan customInstanceResourceModel, sshKey, password string) diag.Diagnostics {
	var diags diag.Diagnostics
	validateSharedInstancePlan(&diags, plan.instanceResourceModel, sshKey, password)
	if strings.TrimSpace(plan.Name.ValueString()) == "" {
		diags.AddError("Missing name", "`name` must be set.")
	}
	if strings.TrimSpace(plan.Host.ValueString()) == "" {
		diags.AddError("Missing host", "`host` must be set.")
	}
	if strings.TrimSpace(plan.InstanceType.ValueString()) == "" {
		diags.AddError("Missing instance type", "`instance_type` must be set.")
	}
	if strings.TrimSpace(plan.GPUModelIn.ValueString()) == "" {
		diags.AddError("Missing gpu_model_input", "`gpu_model_input` must be set.")
	}
	if cores := plan.Cores.ValueInt64(); cores < 1 || cores > 64 || !(cores == 1 || isPowerOfTwo(cores)) {
		diags.AddError("Invalid cores", "`cores` must be 1 or a power of two up to 64.")
	}
	if memory := plan.MemorySize.ValueInt64(); memory < 8 || memory > 512 || memory%2 != 0 {
		diags.AddError("Invalid memory_size", "`memory_size` must be an even number between 8 and 512.")
	}
	if disk := plan.DiskSize.ValueInt64(); disk < 30 || disk > 4096 {
		diags.AddError("Invalid disk_size", "`disk_size` must be between 30 and 4096.")
	}
	if gpuCount := plan.GPUCountIn.ValueInt64(); gpuCount < 0 || gpuCount > 8 {
		diags.AddError("Invalid gpu_count_input", "`gpu_count_input` must be between 0 and 8.")
	}
	return diags
}

func isPowerOfTwo(v int64) bool {
	return v > 0 && v&(v-1) == 0
}

func syncCustomInstanceModel(model *customInstanceResourceModel, remote VMInstance) {
	syncStandardInstanceModel(&model.instanceResourceModel, remote)
	model.Host = nullableString(model.Host.ValueString())
	model.Cores = types.Int64Value(model.Cores.ValueInt64())
	model.MemorySize = types.Int64Value(model.MemorySize.ValueInt64())
	model.DiskSize = types.Int64Value(model.DiskSize.ValueInt64())
	model.GPUCountIn = types.Int64Value(model.GPUCountIn.ValueInt64())
	model.GPUModelIn = nullableString(model.GPUModelIn.ValueString())
}
