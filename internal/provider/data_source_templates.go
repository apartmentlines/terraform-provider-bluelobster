package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &TemplatesDataSource{}

type TemplatesDataSource struct {
	client *Client
}

type templatesDataSourceModel struct {
	Templates types.List `tfsdk:"templates"`
}

func NewTemplatesDataSource() datasource.DataSource {
	return &TemplatesDataSource{}
}

func (d *TemplatesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_templates"
}

func (d *TemplatesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List available Blue Lobster VM templates.",
		Attributes: map[string]schema.Attribute{
			"templates": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":         schema.StringAttribute{Computed: true},
						"display_name": schema.StringAttribute{Computed: true},
						"os_type":      schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *TemplatesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *TemplatesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	templates, err := d.client.ListTemplates(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Blue Lobster templates", err.Error())
		return
	}
	value, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: map[string]attr.Type{
		"name":         types.StringType,
		"display_name": types.StringType,
		"os_type":      types.StringType,
	}}, templates)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &templatesDataSourceModel{Templates: value})...)
}
