package provider

import (
	"context"
	"sort"
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
	_ resource.Resource                = &InstanceFirewallResource{}
	_ resource.ResourceWithImportState = &InstanceFirewallResource{}
)

type instanceFirewallResourceModel struct {
	ID         types.String `tfsdk:"id"`
	InstanceID types.String `tfsdk:"instance_id"`
	Enabled    types.Bool   `tfsdk:"enabled"`
	PolicyIn   types.String `tfsdk:"policy_in"`
	PolicyOut  types.String `tfsdk:"policy_out"`
	Rules      types.List   `tfsdk:"rules"`
}

type InstanceFirewallResource struct {
	client *Client
}

func NewInstanceFirewallResource() resource.Resource {
	return &InstanceFirewallResource{}
}

func (r *InstanceFirewallResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_firewall"
}

func (r *InstanceFirewallResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage the full firewall configuration for a Blue Lobster instance.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"instance_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"enabled":    schema.BoolAttribute{Required: true},
			"policy_in":  schema.StringAttribute{Required: true},
			"policy_out": schema.StringAttribute{Required: true},
			"rules": schema.ListNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"pos":     schema.Int64Attribute{Computed: true},
						"type":    schema.StringAttribute{Required: true},
						"action":  schema.StringAttribute{Required: true},
						"source":  schema.StringAttribute{Optional: true},
						"dest":    schema.StringAttribute{Optional: true},
						"proto":   schema.StringAttribute{Optional: true},
						"dport":   schema.StringAttribute{Optional: true},
						"sport":   schema.StringAttribute{Optional: true},
						"comment": schema.StringAttribute{Optional: true},
						"enabled": schema.BoolAttribute{Optional: true, Computed: true},
					},
				},
			},
		},
	}
}

func (r *InstanceFirewallResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureClient(req.ProviderData, &resp.Diagnostics, &r.client)
}

func (r *InstanceFirewallResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan instanceFirewallResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.applyFirewallPlan(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *InstanceFirewallResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state instanceFirewallResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	firewall, err := r.client.GetFirewall(ctx, state.InstanceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Blue Lobster firewall", err.Error())
		return
	}
	syncFirewallModel(&state, firewall)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *InstanceFirewallResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan instanceFirewallResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.applyFirewallPlan(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *InstanceFirewallResource) applyFirewallPlan(ctx context.Context, plan *instanceFirewallResourceModel, diags *diag.Diagnostics) {
	rules, ruleDiags := expandFirewallRules(ctx, plan.Rules)
	diags.Append(ruleDiags...)
	diags.Append(validateFirewallPlan(*plan, rules)...)
	if diags.HasError() {
		return
	}

	instanceID := strings.TrimSpace(plan.InstanceID.ValueString())
	if err := r.client.UpdateFirewall(ctx, instanceID, FirewallOptions{
		Enabled:   plan.Enabled.ValueBool(),
		PolicyIn:  strings.ToUpper(strings.TrimSpace(plan.PolicyIn.ValueString())),
		PolicyOut: strings.ToUpper(strings.TrimSpace(plan.PolicyOut.ValueString())),
	}); err != nil {
		diags.AddError("Unable to update Blue Lobster firewall", err.Error())
		return
	}

	current, err := r.client.GetFirewall(ctx, instanceID)
	if err != nil {
		diags.AddError("Unable to read Blue Lobster firewall before rule reconciliation", err.Error())
		return
	}
	sort.Slice(current.Rules, func(i, j int) bool { return current.Rules[i].Pos > current.Rules[j].Pos })
	for _, rule := range current.Rules {
		if err := r.client.DeleteFirewallRule(ctx, instanceID, rule.Pos); err != nil {
			diags.AddError("Unable to delete existing Blue Lobster firewall rule", err.Error())
			return
		}
	}

	for _, rule := range rules {
		if rule.Enable == 0 {
			rule.Enable = 1
		}
		if err := r.client.AddFirewallRule(ctx, instanceID, rule); err != nil {
			diags.AddError("Unable to create Blue Lobster firewall rule", err.Error())
			return
		}
	}

	firewall, err := r.client.GetFirewall(ctx, instanceID)
	if err != nil {
		diags.AddError("Unable to read Blue Lobster firewall after reconciliation", err.Error())
		return
	}
	syncFirewallModel(plan, firewall)
}

func (r *InstanceFirewallResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state instanceFirewallResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	firewall, err := r.client.GetFirewall(ctx, state.InstanceID.ValueString())
	if err == nil {
		sort.Slice(firewall.Rules, func(i, j int) bool { return firewall.Rules[i].Pos > firewall.Rules[j].Pos })
		for _, rule := range firewall.Rules {
			if deleteErr := r.client.DeleteFirewallRule(ctx, state.InstanceID.ValueString(), rule.Pos); deleteErr != nil {
				resp.Diagnostics.AddError("Unable to delete Blue Lobster firewall rule", deleteErr.Error())
				return
			}
		}
	}
	if err := r.client.UpdateFirewall(ctx, state.InstanceID.ValueString(), FirewallOptions{Enabled: false, PolicyIn: "ACCEPT", PolicyOut: "ACCEPT"}); err != nil {
		resp.Diagnostics.AddError("Unable to reset Blue Lobster firewall", err.Error())
	}
}

func (r *InstanceFirewallResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("instance_id"), req, resp)
}

func validateFirewallPlan(plan instanceFirewallResourceModel, rules []FirewallRule) diag.Diagnostics {
	var diags diag.Diagnostics
	if strings.TrimSpace(plan.InstanceID.ValueString()) == "" {
		diags.AddError("Missing instance_id", "`instance_id` must be set.")
	}
	if !validPolicy(plan.PolicyIn.ValueString()) || !validPolicy(plan.PolicyOut.ValueString()) {
		diags.AddError("Invalid firewall policy", "`policy_in` and `policy_out` must be `ACCEPT` or `DROP`.")
	}
	for _, rule := range rules {
		if rule.Type != "in" && rule.Type != "out" {
			diags.AddError("Invalid firewall rule type", "`rules.type` must be `in` or `out`.")
			break
		}
		action := strings.ToUpper(rule.Action)
		if action != "ACCEPT" && action != "DROP" && action != "REJECT" {
			diags.AddError("Invalid firewall rule action", "`rules.action` must be `ACCEPT`, `DROP`, or `REJECT`.")
			break
		}
	}
	return diags
}

func validPolicy(v string) bool {
	value := strings.ToUpper(strings.TrimSpace(v))
	return value == "ACCEPT" || value == "DROP"
}

func syncFirewallModel(model *instanceFirewallResourceModel, firewall FirewallStatus) {
	model.ID = types.StringValue(model.InstanceID.ValueString())
	model.Enabled = types.BoolValue(firewall.Enabled)
	model.PolicyIn = types.StringValue(firewall.PolicyIn)
	model.PolicyOut = types.StringValue(firewall.PolicyOut)
	model.Rules = buildFirewallRuleList(firewall.Rules)
}
