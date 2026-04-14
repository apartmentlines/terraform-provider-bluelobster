package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &InstancesDataSource{}

type InstancesDataSource struct {
	client *Client
}

type instancesDataSourceModel struct {
	Instances types.List `tfsdk:"instances"`
}

func NewInstancesDataSource() datasource.DataSource {
	return &InstancesDataSource{}
}

func (d *InstancesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instances"
}

func (d *InstancesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List Blue Lobster instances visible to the current API key.",
		Attributes: map[string]schema.Attribute{
			"instances": schema.ListNestedAttribute{
				Computed:     true,
				NestedObject: instanceNestedObject(),
			},
		},
	}
}

func (d *InstancesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *InstancesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	instances, err := d.client.ListInstances(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Blue Lobster instances", err.Error())
		return
	}
	value, diags := types.ListValueFrom(ctx, instanceDataObjectType(), instances)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &instancesDataSourceModel{Instances: value})...)
}

func instanceNestedObject() schema.NestedAttributeObject {
	return schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"id":                    schema.StringAttribute{Computed: true},
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

func instanceDataObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"id":                    types.StringType,
		"name":                  types.StringType,
		"host_id":               types.StringType,
		"region":                types.StringType,
		"ip_address":            types.StringType,
		"internal_ip":           types.StringType,
		"cpu_cores":             types.Int64Type,
		"memory":                types.Int64Type,
		"storage":               types.Int64Type,
		"gpu_count":             types.Int64Type,
		"gpu_model":             types.StringType,
		"power_status":          types.StringType,
		"created_at":            types.StringType,
		"metadata":              types.MapType{ElemType: types.StringType},
		"instance_type":         types.StringType,
		"price_cents_per_hour":  types.Int64Type,
		"team_id":               types.StringType,
		"team_name":             types.StringType,
		"access_type":           types.StringType,
		"template_name":         types.StringType,
		"template_display_name": types.StringType,
		"os_type":               types.StringType,
		"vm_username":           types.StringType,
	}}
}
