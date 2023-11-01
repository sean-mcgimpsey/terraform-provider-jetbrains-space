package provider

import (
	"context"
	"os"

	space "terraform-provider-jetbrains-space/internal/api"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider = &jetbrainsSpaceProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &jetbrainsSpaceProvider{
			version: version,
		}
	}
}

// jetbrainsSpaceProvider is the provider implementation.
type jetbrainsSpaceProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

type jetbrainsSpaceProviderModel struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	Host  types.String `tfsdk:"host"`
	Token types.String `tfsdk:"token"`
}

// Metadata returns the provider type name.
func (p *jetbrainsSpaceProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "jetbrainsspace"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *jetbrainsSpaceProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Optional: true,
			},
			"token": schema.StringAttribute{
				Optional: true,
			},
		},
	}
}

// Configure prepares a jetbrainsSpace API client for data sources and resources.
func (p *jetbrainsSpaceProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Retrieve provider data from configuration
	tflog.Info(ctx, "Configurating Jetbrains Space provider")
	tflog.Info(ctx, "MADE IT HERE")
	var config jetbrainsSpaceProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If practitioner provided a configuration value for any of the
	// attributes, it must be a known value.

	if config.Host.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Unknown jetbrainsSpace API Host",
			"The provider cannot create the jetbrainsSpace API client as there is an unknown configuration value for the jetbrainsSpace API host. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the jetbrainsSpace_HOST environment variable.",
		)
	}

	if config.Token.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Unknown jetbrainsSpace API Username",
			"The provider cannot create the jetbrainsSpace API client as there is an unknown configuration value for the jetbrainsSpace API username. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the jetbrainsSpace_USERNAME environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.

	host := os.Getenv("jetbrainsSpace_HOST")
	token := os.Getenv("jetbrainsSpace_TOKEN")

	if !config.Host.IsNull() {
		host = config.Host.ValueString()
	}

	if !config.Token.IsNull() {
		token = config.Token.ValueString()
	}

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.

	if host == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Missing jetbrainsSpace API Host",
			"The provider cannot create the jetbrainsSpace API client as there is a missing or empty value for the jetbrainsSpace API host. "+
				"Set the host value in the configuration or use the jetbrainsSpace_HOST environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if token == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Missing jetbrainsSpace API Username",
			"The provider cannot create the jetbrainsSpace API client as there is a missing or empty value for the jetbrainsSpace API username. "+
				"Set the username value in the configuration or use the jetbrainsSpace_USERNAME environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "jetbrainspsace_host", host)
	ctx = tflog.SetField(ctx, "jetbrainspsace_token", token)
	// Create a new jetbrainsSpace client using the configuration values
	client, err := space.NewClient(host, token)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create jetbrainsSpace API Client",
			"An unexpected error occurred when creating the jetbrainsSpace API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"jetbrainsSpace Client Error: "+err.Error(),
		)
		return
	}

	// Make the jetbrainsSpace client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client
	tflog.Info(ctx, "Configured Jetbrains Space client this is a test", map[string]any{"success": true})
}

// DataSources defines the data sources implemented in the provider.
func (p *jetbrainsSpaceProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		projectsDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *jetbrainsSpaceProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewProjectResource,
		NewRepoResource,
	}
}
