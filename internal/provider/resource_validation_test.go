package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestValidateCustomInstancePlanRequiresGPUModel(t *testing.T) {
	diags := validateCustomInstancePlan(customInstanceResourceModel{
		instanceResourceModel: instanceResourceModel{
			Name:         types.StringValue("custom"),
			InstanceType: types.StringValue("gpu_custom"),
			Username:     types.StringValue("ubuntu"),
			TemplateName: types.StringValue("UBUNTU-22-04-NV"),
			PowerState:   types.StringValue("running"),
		},
		Host:       types.StringValue("phl-gpu-01"),
		Cores:      types.Int64Value(8),
		MemorySize: types.Int64Value(32),
		DiskSize:   types.Int64Value(100),
		GPUCountIn: types.Int64Value(1),
		GPUModelIn: types.StringNull(),
	}, "ssh-ed25519 AAAA", "")

	if !diags.HasError() {
		t.Fatal("expected validation error for missing gpu_model_input")
	}
}

func TestValidateBackupSchedulePlanSupportsDocumentedDayFields(t *testing.T) {
	diags := validateBackupSchedulePlan(backupScheduleResourceModel{
		InstanceID: types.StringValue("vm-1"),
		Frequency:  types.StringValue("weekly"),
		HourUTC:    types.Int64Value(3),
		DayOfWeek:  types.Int64Value(1),
	})

	if diags.HasError() {
		t.Fatalf("expected weekly schedule with day_of_week to validate, got %v", diags)
	}
}
