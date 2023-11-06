package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Provider
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

// Repo Resources
type repoResourceModel struct {
	Name              types.String              `tfsdk:"name"`
	ID                types.String              `tfsdk:"id"`
	ProjectID         types.String              `tfsdk:"project_id"`
	LastUpdated       types.String              `tfsdk:"last_updated"`
	Description       types.String              `tfsdk:"description"`
	DefaultBranch     types.String              `tfsdk:"default_branch"`
	Protected         types.Bool                `tfsdk:"protected"`
	ProtectedBranches []repoSettingsBranchModel `tfsdk:"protected_branches"`
}

type repoSettingsBranchModel struct {
	Pattern     []types.String                     `tfsdk:"pattern"`
	QualityGate repoSettingsBranchModelQualityGate `tfsdk:"quality_gate"`
}

type repoSettingsBranchModelQualityGate struct {
	Approvals []repoSettingsBranchModelApprovals `tfsdk:"approvals"`
}

type repoSettingsBranchModelApprovals struct {
	MinApprovals types.Int64    `tfsdk:"min_approvals"`
	ApprovedBy   []types.String `tfsdk:"approved_by"`
}

// Project Resource
type projectResourceModel struct {
	Name        types.String   `tfsdk:"name"`
	Key         types.String   `tfsdk:"key"`
	ID          types.String   `tfsdk:"id"`
	LastUpdated types.String   `tfsdk:"last_updated"`
	Protected   types.Bool     `tfsdk:"protected"`
	MemberTeams []types.String `tfsdk:"member_teams"`
	Members     []types.String `tfsdk:"members"`
	AdminTeams  []types.String `tfsdk:"admin_teams"`
	Admins      []types.String `tfsdk:"admins"`
}

type projectRolesResourceModel struct {
	Team types.String   `tfsdk:"team"`
	Role []types.String `tfsdk:"role"`
}

// Data sources

type ProjectDataSourceModel struct {
	Projects []ProjectsModel `tfsdk:"projects"`
}

type ProjectsModel struct {
	Key  types.String `tfsdk:"key"`
	Id   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}
