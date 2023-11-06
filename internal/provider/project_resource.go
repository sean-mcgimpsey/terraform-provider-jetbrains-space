package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	space "terraform-provider-jetbrains-space/internal/api"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &projectResource{}
	_ resource.ResourceWithConfigure   = &projectResource{}
	_ resource.ResourceWithImportState = &projectResource{}
)

// NewProjectResource is a helper function to simplify the provider implementation.
func NewProjectResource() resource.Resource {
	return &projectResource{}
}

// ProjectResource is the resource implementation.
type projectResource struct {
	client *space.Client
}

// Metadata returns the resource type name.
func (r *projectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

// Create a new resource.
func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan projectResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create new project
	projectName := plan.Name.ValueString()
	project, err := r.client.CreateProject(projectName)
	protected := plan.Protected.ValueBool()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project - "+plan.Name.String()+" ",
			"Could not create project, unexpected error: "+err.Error(),
		)
		return
	}

	var toRemove []string // we dont remove on create

	err = r.UpdateProjectRoles(ctx, plan, project.ID, toRemove, true, "admin")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error mapping teams to admin role in project"+project.ID,
			err.Error(),
		)
		return
	}
	err = r.UpdateProjectRoles(ctx, plan, project.ID, toRemove, true, "member")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error mapping teams to member role in project"+project.ID,
			err.Error(),
		)
		return
	}
	err = r.UpdateProjectRoles(ctx, plan, project.ID, toRemove, false, "admin")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error mapping user to admin role in project"+project.ID,
			err.Error(),
		)
		return
	}
	err = r.UpdateProjectRoles(ctx, plan, project.ID, toRemove, false, "member")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error mapping user to member role in project"+project.ID,
			err.Error(),
		)
		return
	}
	// Call get project again to get updated project values

	p, err := r.client.GetProject(project.ID)

	plan, err = FetchUpdatedAccessForProject(plan, p)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error setting project access profiles to state",
			err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.Name = types.StringValue(project.Name)
	plan.ID = types.StringValue(project.ID)
	plan.Key = types.StringValue(project.Key.Key)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	plan.Protected = types.BoolValue(protected)

	//plan.Key = types.StringValue(project.Key.Key)

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *projectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state projectResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed project values
	project, err := r.client.GetProject(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Jetbrains Space project",
			"Could not read project3 ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Overwrite items with refreshed state
	state.ID = types.StringValue(project.ID)
	state.Name = types.StringValue(project.Name)
	state.Key = types.StringValue(project.Key.Key)

	state, err = FetchUpdatedAccessForProject(state, project)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error setting project access profiles to state",
			err.Error(),
		)
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *projectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan projectResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var project space.Project
	project.Name = plan.Name.ValueString()
	project.Key.Key = plan.Key.ValueString()
	project.ID = plan.ID.ValueString()
	// Update project with plan values
	_, err := r.client.UpdateProject(project.ID, project)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Space project; "+project.ID+" is the value...",
			"Could not update project, unexpected error: "+err.Error(),
		)
		return
	}

	type MembersMap struct {
		Remove    []string
		Different bool
		Team      bool
		TeamType  string
	}
	membersMap := []MembersMap{}

	for _, v := range []string{"members", "member_teams", "admins", "admin_teams"} {
		// Compare plan and state of project roles, so those not in plan can be removed.
		diff, toRemove, err := CompareProjectRoles(ctx, path.Root(v), resp.State, req.Plan)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error comparing state and plan values for Project Roles;"+v+". Project ID; "+project.ID,
				err.Error(),
			)
			return
		}

		var isTeam bool

		var typeOfTeam string
		if strings.HasSuffix(v, "_teams") {
			isTeam = true
		} else {
			isTeam = false
		}
		if strings.HasPrefix(v, "member") {
			typeOfTeam = "member"
		}
		if strings.HasPrefix(v, "admin") {
			typeOfTeam = "admin"
		}

		m := MembersMap{
			Different: diff,
			Remove:    toRemove,
			Team:      isTeam,
			TeamType:  typeOfTeam,
		}

		membersMap = append(membersMap, m)
	}

	for _, v := range membersMap {
		if v.Different {
			if len(v.Remove) > 0 {
				err = r.RemoveProjectMembers(ctx, plan, project.ID, v.Remove, v.Team, v.TeamType)
				if err != nil {
					resp.Diagnostics.AddError(
						"Error removing members from project; "+project.ID,
						err.Error(),
					)
					return
				}
			}
			err = r.UpdateProjectRoles(ctx, plan, project.ID, v.Remove, v.Team, v.TeamType)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error updates members within project; "+project.ID,
					err.Error(),
				)
				return
			}
		}
	}

	// Fetch updated items from Project
	p, err := r.client.GetProject(project.ID)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Space project",
			"Could not read Space project ID "+project.ID+": "+err.Error(),
		)
		return
	}

	plan, err = FetchUpdatedAccessForProject(plan, p)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error setting project access profiles to state",
			err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(p.ID)
	plan.Key = types.StringValue(p.Key.Key)
	plan.Name = types.StringValue(p.Name)

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	plan.Protected = types.BoolValue(plan.Protected.ValueBool())
	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Delete deletes the resource and removes the Terraform state on success.
func (r *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state projectResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing order
	if state.Protected.ValueBool() {
		resp.Diagnostics.AddError(
			"Project is protected, not deleting!",
			"",
		)
	} else {
		// Delete existing order
		err := r.client.DeleteProject(state.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Deleting Space project",
				err.Error(),
			)
			return
		}
	}
}

func (r *projectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *projectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*space.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *space.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *projectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},
			"key": schema.StringAttribute{
				Computed: true,
			},
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				Computed: true,
			},
			"protected": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"member_teams": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			"members": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			"admin_teams": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			"admins": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
		},
	}
}

func (r *projectResource) UpdateProjectRoles(ctx context.Context, plan projectResourceModel, projectID string, toRemove []string, isTeam bool, memberType string) error {

	// Prepare request to map team to project role (Members)
	var members []interface{}

	members = append(members, memberType)
	var empty []interface{}
	empty = append(empty, "")
	if isTeam {
		data := space.ProjectRoles{
			AddRoles:    members,
			RemoveRoles: empty,
		}
		if memberType == "member" {
			for _, v := range plan.MemberTeams {
				data.Team = "name:" + v.ValueString()
				err := r.client.MapTeamToProjectRole(data, projectID)
				if err != nil {
					return err
				}
			}
		} else {
			for _, v := range plan.AdminTeams {

				data.Team = "name:" + v.ValueString()
				err := r.client.MapTeamToProjectRole(data, projectID)
				if err != nil {
					return err
				}
			}
		}

	} else {
		data := space.ProjectMembers{
			AddRoles:    members,
			RemoveRoles: empty,
		}
		if memberType == "member" {

			for _, v := range plan.Members {
				data.Profile = "username:" + v.ValueString()
				err := r.client.SetProjectMembers(data, projectID)
				if err != nil {
					return err
				}
			}
		} else {
			if memberType == "admin" {
				for _, v := range plan.Admins {
					data.Profile = "username:" + v.ValueString()
					err := r.client.SetProjectMembers(data, projectID)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func (r *projectResource) RemoveProjectMembers(ctx context.Context, plan projectResourceModel, projectID string, toRemove []string, isTeam bool, memberType string) error {

	// Prepare request to map team to project role (Members)
	var members []interface{}

	members = append(members, memberType)
	var empty []interface{}
	empty = append(empty, "")

	if len(toRemove) > 0 {
		var data space.ProjectRoles
		if isTeam {
			data = space.ProjectRoles{
				AddRoles:    empty,
				RemoveRoles: members,
			}

			for _, v := range toRemove {

				data.Team = "name:" + v

				err := r.client.MapTeamToProjectRole(data, projectID)
				if err != nil {
					return err
				}
			}
		} else {
			data := space.ProjectMembers{
				AddRoles:    empty,
				RemoveRoles: members,
			}
			for _, v := range toRemove {

				data.Profile = "username:" + v

				err := r.client.SetProjectMembers(data, projectID)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func CompareProjectRoles(ctx context.Context, path path.Path, state tfsdk.State, plan tfsdk.Plan) (bool, []string, error) {
	var stateVal []types.String
	var planVal []types.String
	var different = false

	diag := state.GetAttribute(ctx, path, &stateVal)
	if diag.HasError() {
		return false, []string{""}, fmt.Errorf("Problem getting state value for roles")
	}
	diag = plan.GetAttribute(ctx, path, &planVal)
	if diag.HasError() {
		return false, []string{""}, fmt.Errorf("Problem getting plan value for roles")
	}
	if len(stateVal) != len(planVal) {
		different = true
	} else {
		for i, v := range stateVal {
			if v != planVal[i] {
				different = true
			}
		}
	}
	if different {
		var toRemove []string
		for _, v := range stateVal {
			toRemove = append(toRemove, v.ValueString())
		}
		return true, toRemove, nil
	}
	var nothing []string
	return true, nothing, nil

}

func FetchUpdatedAccessForProject(state projectResourceModel, project space.Project) (projectResourceModel, error) {

	var memberTeamsState []types.String
	for _, value := range project.MemberTeams {
		memberTeamsState = append(memberTeamsState, types.StringValue(value.Name))
	}

	state.MemberTeams = memberTeamsState

	var memberState []types.String
	for _, value := range project.Members {
		memberState = append(memberState, types.StringValue(value.Profile.Username))
	}
	state.Members = memberState

	var adminTeamsState []types.String
	for _, value := range project.AdminTeams {
		adminTeamsState = append(adminTeamsState, types.StringValue(value.Name))
	}

	state.AdminTeams = adminTeamsState

	var adminState []types.String
	for _, value := range project.Admins {
		adminState = append(adminState, types.StringValue(value.Username))
	}
	state.Admins = adminState

	return state, nil

}
