package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &InstanceDataSource{}

type InstanceDataSource struct {
	client *Client
}

type instanceDataSourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	HostID              types.String `tfsdk:"host_id"`
	Region              types.String `tfsdk:"region"`
	IPAddress           types.String `tfsdk:"ip_address"`
	InternalIP          types.String `tfsdk:"internal_ip"`
	CPUCores            types.Int64  `tfsdk:"cpu_cores"`
	Memory              types.Int64  `tfsdk:"memory"`
	Storage             types.Int64  `tfsdk:"storage"`
	GPUCount            types.Int64  `tfsdk:"gpu_count"`
	GPUModel            types.String `tfsdk:"gpu_model"`
	PowerStatus         types.String `tfsdk:"power_status"`
	CreatedAt           types.String `tfsdk:"created_at"`
	Metadata            types.Map    `tfsdk:"metadata"`
	InstanceType        types.String `tfsdk:"instance_type"`
	PriceCentsPerHour   types.Int64  `tfsdk:"price_cents_per_hour"`
	TeamID              types.String `tfsdk:"team_id"`
	TeamName            types.String `tfsdk:"team_name"`
	AccessType          types.String `tfsdk:"access_type"`
	TemplateName        types.String `tfsdk:"template_name"`
	TemplateDisplayName types.String `tfsdk:"template_display_name"`
	OSType              types.String `tfsdk:"os_type"`
	VMUsername          types.String `tfsdk:"vm_username"`
}

func NewInstanceDataSource() datasource.DataSource {
	return &InstanceDataSource{}
}

func (d *InstanceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance"
}

func (d *InstanceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Read a single Blue Lobster instance by ID.",
		Attributes: map[string]schema.Attribute{
			"id":                    schema.StringAttribute{Required: true},
			"name":                  schema.StringAttribute{Computed: true},
			"host_id":               schema.StringAttribute{Computed: true},
			"region":                schema.StringAttribute{Computed: true},
			"ip_address":            schema.StringAttribute{Computed: true},
			"internal_ip":           schema.StringAttribute{Computed: true},
			"cpu_cores":             schema.Int64Attribute{Computed: true},
			"memory":                schema.Int64Attribute{Computed: true},
			"storage":               schema.Int64Attribute{Computed: true},
			"gpu_count":             schema.Int64Attribute{Computed: true},
			"gpu_model":             schema.StringAttribute{Computed: true},
			"power_status":          schema.StringAttribute{Computed: true},
			"created_at":            schema.StringAttribute{Computed: true},
			"metadata":              schema.MapAttribute{Computed: true, ElementType: types.StringType},
			"instance_type":         schema.StringAttribute{Computed: true},
			"price_cents_per_hour":  schema.Int64Attribute{Computed: true},
			"team_id":               schema.StringAttribute{Computed: true},
			"team_name":             schema.StringAttribute{Computed: true},
			"access_type":           schema.StringAttribute{Computed: true},
			"template_name":         schema.StringAttribute{Computed: true},
			"template_display_name": schema.StringAttribute{Computed: true},
			"os_type":               schema.StringAttribute{Computed: true},
			"vm_username":           schema.StringAttribute{Computed: true},
		},
	}
}

func (d *InstanceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("Expected *provider.Client, got %T.", req.ProviderData))
		return
	}
	d.client = client
}

func (d *InstanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state instanceDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	instance, err := d.client.GetInstance(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Blue Lobster instance", err.Error())
		return
	}
	syncInstanceDataSourceModel(&state, instance)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func syncInstanceDataSourceModel(model *instanceDataSourceModel, remote VMInstance) {
	model.ID = types.StringValue(remote.ID)
	model.Name = nullableString(remote.Name)
	model.HostID = nullableString(remote.HostID)
	model.Region = nullableString(remote.Region)
	model.IPAddress = nullableString(remote.IPAddress)
	model.InternalIP = nullableString(remote.InternalIP)
	model.CPUCores = types.Int64Value(remote.CPUCores)
	model.Memory = types.Int64Value(remote.MemoryGB)
	model.Storage = types.Int64Value(remote.StorageGB)
	model.GPUCount = types.Int64Value(remote.GPUCount)
	model.GPUModel = nullableString(remote.GPUModel)
	model.PowerStatus = nullableString(remote.PowerStatus)
	model.CreatedAt = nullableString(remote.CreatedAt)
	model.Metadata = mapToTerraformStringMap(remote.Metadata)
	model.InstanceType = nullableString(remote.InstanceType)
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
