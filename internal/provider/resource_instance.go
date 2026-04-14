package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &InstanceResource{}
	_ resource.ResourceWithImportState = &InstanceResource{}
)

type instanceResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	Region              types.String `tfsdk:"region"`
	InstanceType        types.String `tfsdk:"instance_type"`
	Username            types.String `tfsdk:"username"`
	TemplateName        types.String `tfsdk:"template_name"`
	ISOURL              types.String `tfsdk:"iso_url"`
	Metadata            types.Map    `tfsdk:"metadata"`
	SSHPublicKeyWO      types.String `tfsdk:"ssh_public_key_wo"`
	PasswordWO          types.String `tfsdk:"password_wo"`
	PowerState          types.String `tfsdk:"power_state"`
	HostID              types.String `tfsdk:"host_id"`
	IPAddress           types.String `tfsdk:"ip_address"`
	InternalIP          types.String `tfsdk:"internal_ip"`
	CPUCores            types.Int64  `tfsdk:"cpu_cores"`
	MemoryGB            types.Int64  `tfsdk:"memory"`
	StorageGB           types.Int64  `tfsdk:"storage"`
	GPUCount            types.Int64  `tfsdk:"gpu_count"`
	GPUModel            types.String `tfsdk:"gpu_model"`
	PowerStatus         types.String `tfsdk:"power_status"`
	CreatedAt           types.String `tfsdk:"created_at"`
	PriceCentsPerHour   types.Int64  `tfsdk:"price_cents_per_hour"`
	TeamID              types.String `tfsdk:"team_id"`
	TeamName            types.String `tfsdk:"team_name"`
	AccessType          types.String `tfsdk:"access_type"`
	TemplateDisplayName types.String `tfsdk:"template_display_name"`
	OSType              types.String `tfsdk:"os_type"`
	VMUsername          types.String `tfsdk:"vm_username"`
}

type InstanceResource struct {
	client *Client
}

func NewInstanceResource() resource.Resource {
	return &InstanceResource{}
}

func (r *InstanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance"
}

func (r *InstanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage a standard Blue Lobster instance launched from a region and instance type.",
		Attributes:          standardInstanceAttributes(),
	}
}

func standardInstanceAttributes() map[string]schema.Attribute {
	attrs := sharedInstanceAttributes()
	attrs["region"] = schema.StringAttribute{
		MarkdownDescription: "Deployment region.",
		Required:            true,
		PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	attrs["instance_type"] = schema.StringAttribute{
		MarkdownDescription: "Blue Lobster instance type identifier.",
		Required:            true,
		PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	return attrs
}

func sharedInstanceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"name": schema.StringAttribute{
			Optional: true,
			Computed: true,
		},
		"username": schema.StringAttribute{
			Required:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"template_name": schema.StringAttribute{
			Optional:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"iso_url": schema.StringAttribute{
			Optional:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"metadata": schema.MapAttribute{
			Optional:      true,
			ElementType:   types.StringType,
			PlanModifiers: []planmodifier.Map{mapplanmodifier.RequiresReplace()},
		},
		"ssh_public_key_wo": schema.StringAttribute{
			Optional:      true,
			Sensitive:     true,
			WriteOnly:     true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"password_wo": schema.StringAttribute{
			Optional:      true,
			Sensitive:     true,
			WriteOnly:     true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"power_state": schema.StringAttribute{
			Optional:      true,
			Computed:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"host_id":               schema.StringAttribute{Computed: true},
		"ip_address":            schema.StringAttribute{Computed: true},
		"internal_ip":           schema.StringAttribute{Computed: true},
		"cpu_cores":             schema.Int64Attribute{Computed: true},
		"memory":                schema.Int64Attribute{Computed: true},
		"storage":               schema.Int64Attribute{Computed: true},
		"gpu_count":             schema.Int64Attribute{Computed: true},
		"gpu_model":             schema.StringAttribute{Computed: true},
		"power_status":          schema.StringAttribute{Computed: true},
		"created_at":            schema.StringAttribute{Computed: true},
		"price_cents_per_hour":  schema.Int64Attribute{Computed: true},
		"team_id":               schema.StringAttribute{Computed: true},
		"team_name":             schema.StringAttribute{Computed: true},
		"access_type":           schema.StringAttribute{Computed: true},
		"template_display_name": schema.StringAttribute{Computed: true},
		"os_type":               schema.StringAttribute{Computed: true},
		"vm_username":           schema.StringAttribute{Computed: true},
	}
}

func (r *InstanceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureClient(req.ProviderData, &resp.Diagnostics, &r.client)
}

func (r *InstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan instanceResourceModel
	var sshKey, password types.String

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("ssh_public_key_wo"), &sshKey)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("password_wo"), &password)...)
	if resp.Diagnostics.HasError() {
		return
	}

	metadata, diags := buildStringMap(ctx, plan.Metadata)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(validateStandardInstancePlan(plan, sshKey.ValueString(), password.ValueString())...)
	if resp.Diagnostics.HasError() {
		return
	}

	launch, err := r.client.LaunchStandardInstance(ctx, LaunchStandardInstanceInput{
		Region:       strings.TrimSpace(plan.Region.ValueString()),
		InstanceType: strings.TrimSpace(plan.InstanceType.ValueString()),
		Name:         strings.TrimSpace(plan.Name.ValueString()),
		Username:     strings.TrimSpace(plan.Username.ValueString()),
		SSHPublicKey: strings.TrimSpace(sshKey.ValueString()),
		Password:     strings.TrimSpace(password.ValueString()),
		Metadata:     metadata,
		TemplateName: strings.TrimSpace(plan.TemplateName.ValueString()),
		ISOURL:       strings.TrimSpace(plan.ISOURL.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Blue Lobster instance", err.Error())
		return
	}

	remote, err := r.waitForCreatedInstance(ctx, launch)
	if err != nil {
		resp.Diagnostics.AddError("Unable to observe created Blue Lobster instance", err.Error())
		return
	}

	remote, err = reconcileDesiredPowerState(ctx, r.client, remote, normalizeDesiredPowerState(plan.PowerState))
	if err != nil {
		resp.Diagnostics.AddError("Unable to reconcile Blue Lobster instance power state", err.Error())
		return
	}

	syncStandardInstanceModel(&plan, remote)
	plan.SSHPublicKeyWO = types.StringNull()
	plan.PasswordWO = types.StringNull()

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *InstanceResource) resolveCreatedInstanceID(ctx context.Context, launch LaunchResponse) (string, error) {
	if launch.InstanceID != "" {
		return launch.InstanceID, nil
	}
	if launch.TaskID == "" {
		return "", fmt.Errorf("launch response did not include an instance identifier or task id")
	}
	task, err := r.client.WaitForTask(ctx, launch.TaskID)
	if err != nil {
		return "", err
	}
	return extractInstanceIDFromTask(task)
}

func (r *InstanceResource) waitForCreatedInstance(ctx context.Context, launch LaunchResponse) (VMInstance, error) {
	instanceID := launch.InstanceID
	if launch.TaskID != "" {
		tflog.Debug(ctx, "waiting for Blue Lobster launch task", map[string]any{
			"task_id":     launch.TaskID,
			"instance_id": instanceID,
			"assigned_ip": launch.AssignedIP,
		})
		task, err := r.client.WaitForTask(ctx, launch.TaskID)
		if err != nil {
			return VMInstance{}, err
		}
		if instanceID == "" {
			instanceID, err = extractInstanceIDFromTask(task)
			if err != nil {
				return VMInstance{}, err
			}
		}
	}
	if instanceID == "" {
		return VMInstance{}, fmt.Errorf("unable to resolve created Blue Lobster instance id")
	}

	tflog.Debug(ctx, "waiting for Blue Lobster instance to become visible", map[string]any{
		"instance_id": instanceID,
	})
	return r.client.WaitForInstanceVisible(ctx, instanceID)
}

func extractInstanceIDFromTask(task Task) (string, error) {
	if task.Params != nil {
		for _, key := range []string{"vm_uuid", "instance_id"} {
			if value := stringValue(task.Params[key]); value != "" {
				return value, nil
			}
		}
	}
	return "", fmt.Errorf("task %s completed without an instance identifier", task.ID)
}

func (r *InstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state instanceResourceModel
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
		resp.Diagnostics.AddError("Unable to read Blue Lobster instance", err.Error())
		return
	}

	syncStandardInstanceModel(&state, remote)
	state.SSHPublicKeyWO = types.StringNull()
	state.PasswordWO = types.StringNull()
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *InstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state instanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if strings.TrimSpace(plan.Name.ValueString()) != strings.TrimSpace(state.Name.ValueString()) && strings.TrimSpace(plan.Name.ValueString()) != "" {
		remote, err := applyInstanceAction(ctx, r.client, state.ID.ValueString(), "rename",
			func(ctx context.Context) (ActionResponse, error) {
				return r.client.RenameInstance(ctx, state.ID.ValueString(), strings.TrimSpace(plan.Name.ValueString()))
			},
			func(instance VMInstance) bool {
				return strings.TrimSpace(instance.Name) == strings.TrimSpace(plan.Name.ValueString())
			},
		)
		if err != nil {
			resp.Diagnostics.AddError("Unable to rename Blue Lobster instance", err.Error())
			return
		}
		syncStandardInstanceModel(&state, remote)
		state.SSHPublicKeyWO = types.StringNull()
		state.PasswordWO = types.StringNull()
	} else {
		remote, err := r.client.GetInstance(ctx, state.ID.ValueString())
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				resp.State.RemoveResource(ctx)
				return
			}
			resp.Diagnostics.AddError("Unable to read Blue Lobster instance before update", err.Error())
			return
		}
		syncStandardInstanceModel(&state, remote)
	}

	remote, err := reconcileDesiredPowerState(ctx, r.client, VMInstance{
		ID:          state.ID.ValueString(),
		Name:        state.Name.ValueString(),
		PowerStatus: state.PowerStatus.ValueString(),
	}, normalizeDesiredPowerState(plan.PowerState))
	if err != nil {
		resp.Diagnostics.AddError("Unable to reconcile Blue Lobster instance power state", err.Error())
		return
	}

	syncStandardInstanceModel(&plan, remote)
	plan.SSHPublicKeyWO = types.StringNull()
	plan.PasswordWO = types.StringNull()
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *InstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state instanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := deleteInstanceAndWait(ctx, r.client, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to delete Blue Lobster instance", err.Error())
		return
	}
}

func (r *InstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func validateStandardInstancePlan(plan instanceResourceModel, sshKey, password string) diag.Diagnostics {
	var diags diag.Diagnostics

	if strings.TrimSpace(plan.Region.ValueString()) == "" {
		diags.AddError("Missing region", "`region` must be set.")
	}
	if strings.TrimSpace(plan.InstanceType.ValueString()) == "" {
		diags.AddError("Missing instance type", "`instance_type` must be set.")
	}
	validateSharedInstancePlan(&diags, plan, sshKey, password)
	return diags
}

func validateSharedInstancePlan(diags *diag.Diagnostics, plan instanceResourceModel, sshKey, password string) {
	if strings.TrimSpace(plan.Username.ValueString()) == "" {
		diags.AddError("Missing username", "`username` must be set.")
	}
	if strings.TrimSpace(sshKey) == "" && strings.TrimSpace(password) == "" {
		diags.AddError("Missing credentials", "Set at least one of `ssh_public_key_wo` or `password_wo`.")
	}
	if strings.TrimSpace(plan.TemplateName.ValueString()) == "" && strings.TrimSpace(plan.ISOURL.ValueString()) == "" {
		diags.AddError("Missing image source", "Set one of `template_name` or `iso_url`.")
	}
	if normalizeDesiredPowerState(plan.PowerState) == "" {
		diags.AddError("Invalid power_state", "`power_state` must be `running` or `stopped` if set.")
	}
}

func normalizeDesiredPowerState(value types.String) string {
	if value.IsNull() || value.IsUnknown() || strings.TrimSpace(value.ValueString()) == "" {
		return "running"
	}
	switch normalizePowerState(value.ValueString()) {
	case "running", "stopped":
		return normalizePowerState(value.ValueString())
	default:
		return ""
	}
}

func reconcileDesiredPowerState(ctx context.Context, client *Client, current VMInstance, desired string) (VMInstance, error) {
	switch desired {
	case "", "running":
		if normalizePowerState(current.PowerStatus) == "running" {
			return current, nil
		}
		return applyInstanceAction(ctx, client, current.ID, "power-on",
			func(ctx context.Context) (ActionResponse, error) {
				return client.PowerOnInstance(ctx, current.ID)
			},
			func(instance VMInstance) bool {
				return normalizePowerState(instance.PowerStatus) == "running"
			},
		)
	case "stopped":
		if normalizePowerState(current.PowerStatus) == "stopped" {
			return current, nil
		}
		return applyInstanceAction(ctx, client, current.ID, "shutdown",
			func(ctx context.Context) (ActionResponse, error) {
				return client.ShutdownInstance(ctx, current.ID)
			},
			func(instance VMInstance) bool {
				return normalizePowerState(instance.PowerStatus) == "stopped"
			},
		)
	default:
		return VMInstance{}, fmt.Errorf("unsupported desired power state %q", desired)
	}
}

func applyInstanceAction(
	ctx context.Context,
	client *Client,
	instanceID, action string,
	submit func(context.Context) (ActionResponse, error),
	condition func(VMInstance) bool,
) (VMInstance, error) {
	ticker := time.NewTicker(taskPollInterval)
	defer ticker.Stop()

	for {
		response, err := submit(ctx)
		if err == nil {
			if response.TaskID != "" {
				tflog.Debug(ctx, "waiting for Blue Lobster instance action task", map[string]any{
					"instance_id": instanceID,
					"action":      action,
					"task_id":     response.TaskID,
				})
				if _, err := client.WaitForTask(ctx, response.TaskID); err != nil {
					return VMInstance{}, err
				}
			}
			return waitForInstanceCondition(ctx, client, instanceID, action, condition)
		}

		if !isAPIInvalidState(err) {
			return VMInstance{}, err
		}

		tflog.Debug(ctx, "Blue Lobster instance action not ready; retrying", map[string]any{
			"instance_id": instanceID,
			"action":      action,
			"error":       err.Error(),
		})

		current, readErr := client.GetInstance(ctx, instanceID)
		if readErr == nil && condition(current) {
			return current, nil
		}
		if readErr != nil && !errors.Is(readErr, ErrNotFound) {
			return VMInstance{}, readErr
		}

		select {
		case <-ctx.Done():
			return VMInstance{}, fmt.Errorf("wait to retry Blue Lobster instance %s action on %s: %w", action, instanceID, ctx.Err())
		case <-ticker.C:
		}
	}
}

func waitForInstanceCondition(ctx context.Context, client *Client, instanceID, conditionName string, condition func(VMInstance) bool) (VMInstance, error) {
	ticker := time.NewTicker(taskPollInterval)
	defer ticker.Stop()

	for {
		instance, err := client.GetInstance(ctx, instanceID)
		if err != nil {
			return VMInstance{}, err
		}
		if condition(instance) {
			return instance, nil
		}

		tflog.Trace(ctx, "waiting for Blue Lobster instance condition", map[string]any{
			"instance_id": instanceID,
			"condition":   conditionName,
			"power":       instance.PowerStatus,
			"name":        instance.Name,
		})

		select {
		case <-ctx.Done():
			return VMInstance{}, fmt.Errorf("wait for Blue Lobster instance %s condition %s: %w", instanceID, conditionName, ctx.Err())
		case <-ticker.C:
		}
	}
}

func deleteInstanceAndWait(ctx context.Context, client *Client, instanceID string) error {
	ticker := time.NewTicker(taskPollInterval)
	defer ticker.Stop()

	for {
		response, err := client.DeleteInstance(ctx, instanceID)
		if err == nil {
			if response.TaskID != "" {
				tflog.Debug(ctx, "waiting for Blue Lobster delete task", map[string]any{
					"instance_id": instanceID,
					"task_id":     response.TaskID,
				})
				if _, err := client.WaitForTask(ctx, response.TaskID); err != nil {
					return err
				}
			}
			return client.WaitForDeletion(ctx, instanceID)
		}
		if !isAPIInvalidState(err) {
			return err
		}

		tflog.Debug(ctx, "Blue Lobster instance delete not ready; retrying", map[string]any{
			"instance_id": instanceID,
			"error":       err.Error(),
		})

		select {
		case <-ctx.Done():
			return fmt.Errorf("wait to retry Blue Lobster instance deletion on %s: %w", instanceID, ctx.Err())
		case <-ticker.C:
		}
	}
}

func syncStandardInstanceModel(model *instanceResourceModel, remote VMInstance) {
	model.ID = types.StringValue(remote.ID)
	model.Name = nullableString(remote.Name)
	if model.Region.IsNull() || model.Region.IsUnknown() || strings.TrimSpace(model.Region.ValueString()) == "" {
		model.Region = nullableString(remote.Region)
	}
	model.InstanceType = nullableString(remote.InstanceType)
	model.HostID = nullableString(remote.HostID)
	model.IPAddress = nullableString(remote.IPAddress)
	model.InternalIP = nullableString(remote.InternalIP)
	model.CPUCores = types.Int64Value(remote.CPUCores)
	model.MemoryGB = types.Int64Value(remote.MemoryGB)
	model.StorageGB = types.Int64Value(remote.StorageGB)
	model.GPUCount = types.Int64Value(remote.GPUCount)
	model.GPUModel = nullableString(remote.GPUModel)
	model.PowerStatus = nullableString(remote.PowerStatus)
	model.PowerState = types.StringValue(firstNonEmptyString(normalizePowerState(remote.PowerStatus), "running"))
	model.CreatedAt = nullableString(remote.CreatedAt)
	if remote.Metadata != nil {
		model.Metadata = mapToTerraformStringMap(remote.Metadata)
	}
	if remote.PriceCentsPerHour != nil {
		model.PriceCentsPerHour = types.Int64Value(*remote.PriceCentsPerHour)
	} else {
		model.PriceCentsPerHour = types.Int64Null()
	}
	model.TeamID = nullableString(remote.TeamID)
	model.TeamName = nullableString(remote.TeamName)
	model.AccessType = nullableString(remote.AccessType)
	model.TemplateName = nullableString(remote.TemplateName)
	model.TemplateDisplayName = nullableString(remote.TemplateDisplayName)
	model.OSType = nullableString(remote.OSType)
	model.VMUsername = nullableString(remote.VMUsername)
}

func configureClient(providerData any, diags *diag.Diagnostics, target **Client) {
	if providerData == nil {
		return
	}
	client, ok := providerData.(*Client)
	if !ok {
		diags.AddError("Unexpected provider data type", fmt.Sprintf("Expected *provider.Client, got %T.", providerData))
		return
	}
	*target = client
}
