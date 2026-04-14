package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &InstanceBackupsDataSource{}

type InstanceBackupsDataSource struct {
	client *Client
}

type instanceBackupsDataSourceModel struct {
	InstanceID types.String `tfsdk:"instance_id"`
	Storage    types.String `tfsdk:"storage"`
	Backups    types.List   `tfsdk:"backups"`
	Total      types.Int64  `tfsdk:"total"`
}

func NewInstanceBackupsDataSource() datasource.DataSource {
	return &InstanceBackupsDataSource{}
}

func (d *InstanceBackupsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_backups"
}

func (d *InstanceBackupsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List backups for a Blue Lobster instance.",
		Attributes: map[string]schema.Attribute{
			"instance_id": schema.StringAttribute{Required: true},
			"storage":     schema.StringAttribute{Computed: true},
			"total":       schema.Int64Attribute{Computed: true},
			"backups": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"volid":      schema.StringAttribute{Computed: true},
						"size_bytes": schema.Int64Attribute{Computed: true},
						"size_human": schema.StringAttribute{Computed: true},
						"created_at": schema.StringAttribute{Computed: true},
						"format":     schema.StringAttribute{Computed: true},
						"notes":      schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *InstanceBackupsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *InstanceBackupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state instanceBackupsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	backups, err := d.client.ListInstanceBackups(ctx, state.InstanceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Blue Lobster instance backups", err.Error())
		return
	}
	state.Storage = nullableString(backups.Storage)
	state.Total = types.Int64Value(backups.Total)
	list, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: map[string]attr.Type{
		"volid":      types.StringType,
		"size_bytes": types.Int64Type,
		"size_human": types.StringType,
		"created_at": types.StringType,
		"format":     types.StringType,
		"notes":      types.StringType,
	}}, backups.Backups)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Backups = list
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
