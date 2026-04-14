package provider

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const defaultBaseURL = "https://api.bluelobster.ai/api/v1"

var _ provider.Provider = &BlueLobsterProvider{}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &BlueLobsterProvider{version: version}
	}
}

type BlueLobsterProvider struct {
	version string
}

type BlueLobsterProviderModel struct {
	APIKey  types.String `tfsdk:"api_key"`
	BaseURL types.String `tfsdk:"base_url"`
}

func (p *BlueLobsterProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "bluelobster"
	resp.Version = p.version
}

func (p *BlueLobsterProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Terraform provider for provisioning Blue Lobster instances through the public API.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				MarkdownDescription: "Blue Lobster API key. Can also be supplied with the `BLUELOBSTER_API_KEY` or `BLUELOBSTER_API_TOKEN` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"base_url": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Blue Lobster API base URL. Defaults to `%s`. Can also be supplied with `BLUELOBSTER_BASE_URL`.", defaultBaseURL),
				Optional:            true,
			},
		},
	}
}

func (p *BlueLobsterProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data BlueLobsterProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiKey := strings.TrimSpace(os.Getenv("BLUELOBSTER_API_KEY"))
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("BLUELOBSTER_API_TOKEN"))
	}
	if !data.APIKey.IsNull() && !data.APIKey.IsUnknown() {
		apiKey = strings.TrimSpace(data.APIKey.ValueString())
	}
	if apiKey == "" {
		resp.Diagnostics.AddError(
			"Missing Blue Lobster API key",
			"Set `api_key` in provider configuration or export `BLUELOBSTER_API_KEY`.",
		)
		return
	}

	baseURL := strings.TrimSpace(os.Getenv("BLUELOBSTER_BASE_URL"))
	if !data.BaseURL.IsNull() && !data.BaseURL.IsUnknown() {
		baseURL = strings.TrimSpace(data.BaseURL.ValueString())
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	client, err := NewClient(baseURL, apiKey, p.version)
	if err != nil {
		resp.Diagnostics.AddError("Invalid provider configuration", err.Error())
		return
	}

	resp.ResourceData = client
	resp.DataSourceData = client
}

func (p *BlueLobsterProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewInstanceResource,
		NewCustomInstanceResource,
		NewInstanceFirewallResource,
		NewBackupScheduleResource,
		NewInstanceIPResource,
	}
}

func (p *BlueLobsterProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewAvailableInstancesDataSource,
		NewTemplatesDataSource,
		NewInstancesDataSource,
		NewInstanceDataSource,
		NewInstanceBackupsDataSource,
	}
}
