package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &AvailableInstancesDataSource{}

type AvailableInstancesDataSource struct {
	client *Client
}

type availableInstancesDataSourceModel struct {
	Items types.List `tfsdk:"items"`
}

func NewAvailableInstancesDataSource() datasource.DataSource {
	return &AvailableInstancesDataSource{}
}

func (d *AvailableInstancesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_available_instances"
}

func (d *AvailableInstancesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List Blue Lobster instance types and current region capacity.",
		Attributes: map[string]schema.Attribute{
			"items": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{Computed: true},
						"instance_type": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"name":                 schema.StringAttribute{Computed: true},
								"description":          schema.StringAttribute{Computed: true},
								"gpu_description":      schema.StringAttribute{Computed: true},
								"price_cents_per_hour": schema.Int64Attribute{Computed: true},
								"specs": schema.SingleNestedAttribute{
									Computed: true,
									Attributes: map[string]schema.Attribute{
										"vcpus":       schema.Int64Attribute{Computed: true},
										"memory_gib":  schema.Int64Attribute{Computed: true},
										"storage_gib": schema.Int64Attribute{Computed: true},
										"gpus":        schema.Int64Attribute{Computed: true},
										"gpu_model":   schema.StringAttribute{Computed: true},
									},
								},
							},
						},
						"regions_with_capacity_available": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"name":        schema.StringAttribute{Computed: true},
									"description": schema.StringAttribute{Computed: true},
									"location": schema.SingleNestedAttribute{
										Computed: true,
										Attributes: map[string]schema.Attribute{
											"city":    schema.StringAttribute{Computed: true},
											"state":   schema.StringAttribute{Computed: true},
											"country": schema.StringAttribute{Computed: true},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *AvailableInstancesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AvailableInstancesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	items, err := d.client.ListAvailableInstances(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Blue Lobster available instances", err.Error())
		return
	}
	value, diags := types.ListValueFrom(ctx, availableInstanceObjectType(), items)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &availableInstancesDataSourceModel{Items: value})...)
}

func availableInstanceObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"id": types.StringType,
		"instance_type": types.ObjectType{AttrTypes: map[string]attr.Type{
			"name":                 types.StringType,
			"description":          types.StringType,
			"gpu_description":      types.StringType,
			"price_cents_per_hour": types.Int64Type,
			"specs": types.ObjectType{AttrTypes: map[string]attr.Type{
				"vcpus":       types.Int64Type,
				"memory_gib":  types.Int64Type,
				"storage_gib": types.Int64Type,
				"gpus":        types.Int64Type,
				"gpu_model":   types.StringType,
			}},
		}},
		"regions_with_capacity_available": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
			"name":        types.StringType,
			"description": types.StringType,
			"location": types.ObjectType{AttrTypes: map[string]attr.Type{
				"city":    types.StringType,
				"state":   types.StringType,
				"country": types.StringType,
			}},
		}}},
	}}
}
