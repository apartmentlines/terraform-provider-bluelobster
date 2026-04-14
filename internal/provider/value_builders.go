package provider

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type firewallRuleModel struct {
	Pos     types.Int64  `tfsdk:"pos"`
	Type    types.String `tfsdk:"type"`
	Action  types.String `tfsdk:"action"`
	Source  types.String `tfsdk:"source"`
	Dest    types.String `tfsdk:"dest"`
	Proto   types.String `tfsdk:"proto"`
	DPort   types.String `tfsdk:"dport"`
	SPort   types.String `tfsdk:"sport"`
	Comment types.String `tfsdk:"comment"`
	Enabled types.Bool   `tfsdk:"enabled"`
}

func buildStringMap(ctx context.Context, value types.Map) (map[string]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	if value.IsNull() || value.IsUnknown() {
		return nil, diags
	}

	items := map[string]string{}
	elements := value.Elements()
	for key, elem := range elements {
		strValue, ok := elem.(basetypes.StringValue)
		if !ok {
			diags.AddError("Invalid metadata value", fmt.Sprintf("Expected string map value for key %q.", key))
			continue
		}
		items[key] = strValue.ValueString()
	}
	return items, diags
}

func mapToTerraformStringMap(input map[string]string) types.Map {
	if len(input) == 0 {
		return types.MapNull(types.StringType)
	}
	result, _ := types.MapValueFrom(context.Background(), types.StringType, input)
	return result
}

func buildFirewallRuleList(rules []FirewallRule) types.List {
	if len(rules) == 0 {
		return types.ListNull(firewallRuleObjectType())
	}
	models := make([]firewallRuleModel, 0, len(rules))
	for _, rule := range rules {
		models = append(models, firewallRuleModel{
			Pos:     types.Int64Value(rule.Pos),
			Type:    types.StringValue(rule.Type),
			Action:  types.StringValue(rule.Action),
			Source:  nullableString(rule.Source),
			Dest:    nullableString(rule.Dest),
			Proto:   nullableString(rule.Proto),
			DPort:   nullableString(rule.DPort),
			SPort:   nullableString(rule.SPort),
			Comment: nullableString(rule.Comment),
			Enabled: types.BoolValue(rule.Enable != 0),
		})
	}
	value, _ := types.ListValueFrom(context.Background(), firewallRuleObjectType(), models)
	return value
}

func firewallRuleObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"pos":     types.Int64Type,
		"type":    types.StringType,
		"action":  types.StringType,
		"source":  types.StringType,
		"dest":    types.StringType,
		"proto":   types.StringType,
		"dport":   types.StringType,
		"sport":   types.StringType,
		"comment": types.StringType,
		"enabled": types.BoolType,
	}}
}

func expandFirewallRules(ctx context.Context, list types.List) ([]FirewallRule, diag.Diagnostics) {
	var diags diag.Diagnostics
	if list.IsNull() || list.IsUnknown() {
		return nil, diags
	}

	var models []firewallRuleModel
	diags.Append(list.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return nil, diags
	}

	rules := make([]FirewallRule, 0, len(models))
	for _, model := range models {
		rules = append(rules, FirewallRule{
			Type:    strings.TrimSpace(model.Type.ValueString()),
			Action:  strings.TrimSpace(model.Action.ValueString()),
			Source:  strings.TrimSpace(model.Source.ValueString()),
			Dest:    strings.TrimSpace(model.Dest.ValueString()),
			Proto:   strings.TrimSpace(model.Proto.ValueString()),
			DPort:   strings.TrimSpace(model.DPort.ValueString()),
			SPort:   strings.TrimSpace(model.SPort.ValueString()),
			Comment: strings.TrimSpace(model.Comment.ValueString()),
			Enable:  boolToEnable(model.Enabled.ValueBool()),
		})
	}
	return rules, diags
}

func boolToEnable(v bool) int64 {
	if v {
		return 1
	}
	return 0
}

func normalizeIPList(ips []string) []string {
	out := append([]string{}, ips...)
	sort.Strings(out)
	return out
}

func buildStringList(values []string) types.List {
	if len(values) == 0 {
		return types.ListNull(types.StringType)
	}
	elems := make([]types.String, 0, len(values))
	for _, value := range values {
		elems = append(elems, types.StringValue(value))
	}
	result, _ := types.ListValueFrom(context.Background(), types.StringType, elems)
	return result
}

func nullableString(value string) types.String {
	if strings.TrimSpace(value) == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}
