package provider

import (
	"context"
	"errors"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &BackupScheduleResource{}
	_ resource.ResourceWithImportState = &BackupScheduleResource{}
)

type backupScheduleResourceModel struct {
	ID         types.String `tfsdk:"id"`
	InstanceID types.String `tfsdk:"instance_id"`
	Frequency  types.String `tfsdk:"frequency"`
	HourUTC    types.Int64  `tfsdk:"hour_utc"`
	DayOfWeek  types.Int64  `tfsdk:"day_of_week"`
	DayOfMonth types.Int64  `tfsdk:"day_of_month"`
}

type BackupScheduleResource struct {
	client *Client
}

func NewBackupScheduleResource() resource.Resource {
	return &BackupScheduleResource{}
}

func (r *BackupScheduleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_backup_schedule"
}

func (r *BackupScheduleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage the backup schedule for a Blue Lobster instance.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"instance_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"frequency":    schema.StringAttribute{Required: true},
			"hour_utc":     schema.Int64Attribute{Required: true},
			"day_of_week":  schema.Int64Attribute{Optional: true},
			"day_of_month": schema.Int64Attribute{Optional: true},
		},
	}
}

func (r *BackupScheduleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureClient(req.ProviderData, &resp.Diagnostics, &r.client)
}

func (r *BackupScheduleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan backupScheduleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	applyBackupSchedule(ctx, r.client, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BackupScheduleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state backupScheduleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	schedule, err := r.client.GetBackupSchedule(ctx, state.InstanceID.ValueString())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read Blue Lobster backup schedule", err.Error())
		return
	}
	syncBackupScheduleModel(&state, schedule)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *BackupScheduleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan backupScheduleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	applyBackupSchedule(ctx, r.client, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func applyBackupSchedule(ctx context.Context, client *Client, model *backupScheduleResourceModel, diags *diag.Diagnostics) {
	diags.Append(validateBackupSchedulePlan(*model)...)
	if diags.HasError() {
		return
	}

	schedule := BackupSchedule{
		Frequency: strings.TrimSpace(model.Frequency.ValueString()),
		HourUTC:   model.HourUTC.ValueInt64(),
	}
	if !model.DayOfWeek.IsNull() && !model.DayOfWeek.IsUnknown() {
		value := model.DayOfWeek.ValueInt64()
		schedule.DayOfWeek = &value
	}
	if !model.DayOfMonth.IsNull() && !model.DayOfMonth.IsUnknown() {
		value := model.DayOfMonth.ValueInt64()
		schedule.DayOfMonth = &value
	}

	if err := client.UpsertBackupSchedule(ctx, model.InstanceID.ValueString(), schedule); err != nil {
		diags.AddError("Unable to update Blue Lobster backup schedule", err.Error())
		return
	}

	remote, err := client.GetBackupSchedule(ctx, model.InstanceID.ValueString())
	if err != nil {
		diags.AddError("Unable to read Blue Lobster backup schedule", err.Error())
		return
	}
	syncBackupScheduleModel(model, remote)
}

func (r *BackupScheduleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state backupScheduleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteBackupSchedule(ctx, state.InstanceID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to delete Blue Lobster backup schedule", err.Error())
	}
}

func (r *BackupScheduleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("instance_id"), req, resp)
}

func validateBackupSchedulePlan(plan backupScheduleResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	if strings.TrimSpace(plan.InstanceID.ValueString()) == "" {
		diags.AddError("Missing instance_id", "`instance_id` must be set.")
	}
	if plan.HourUTC.ValueInt64() < 0 || plan.HourUTC.ValueInt64() > 23 {
		diags.AddError("Invalid hour_utc", "`hour_utc` must be between 0 and 23.")
	}
	switch strings.ToLower(strings.TrimSpace(plan.Frequency.ValueString())) {
	case "daily":
	case "weekly":
		if plan.DayOfWeek.IsNull() || plan.DayOfWeek.IsUnknown() {
			diags.AddError("Missing day_of_week", "`day_of_week` is required for weekly schedules.")
		}
	case "monthly":
		if plan.DayOfMonth.IsNull() || plan.DayOfMonth.IsUnknown() {
			diags.AddError("Missing day_of_month", "`day_of_month` is required for monthly schedules.")
		}
	default:
		diags.AddError("Invalid frequency", "`frequency` must be `daily`, `weekly`, or `monthly`.")
	}
	return diags
}

func syncBackupScheduleModel(model *backupScheduleResourceModel, schedule BackupSchedule) {
	model.ID = types.StringValue(model.InstanceID.ValueString())
	model.Frequency = types.StringValue(schedule.Frequency)
	model.HourUTC = types.Int64Value(schedule.HourUTC)
	if schedule.DayOfWeek != nil {
		model.DayOfWeek = types.Int64Value(*schedule.DayOfWeek)
	} else {
		model.DayOfWeek = types.Int64Null()
	}
	if schedule.DayOfMonth != nil {
		model.DayOfMonth = types.Int64Value(*schedule.DayOfMonth)
	} else {
		model.DayOfMonth = types.Int64Null()
	}
}
