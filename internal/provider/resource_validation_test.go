package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

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
