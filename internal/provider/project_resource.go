package provider

import (
	"context"
	"fmt"
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
	_ resource.Resource              = &projectResource{}
	_ resource.ResourceWithConfigure = &projectResource{}
)

// NewProjectResource is a helper function to simplify the provider implementation.
func NewProjectResource() resource.Resource {
	return &projectResource{}
}

// ProjectResource is the resource implementation.
type projectResource struct {
	client *space.Client
}

type projectResourceModel struct {
	Name        types.String   `tfsdk:"name"`
	Key         types.String   `tfsdk:"key"`
	ID          types.String   `tfsdk:"id"`
	LastUpdated types.String   `tfsdk:"last_updated"`
	Protected   types.Bool     `tfsdk:"protected"`
	MemberTeams []types.String `tfsdk:"member_teams"`
}

type projectRolesResourceModel struct {
	Team types.String   `tfsdk:"team"`
	Role []types.String `tfsdk:"role"`
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

	// Create new order
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

	err = r.UpdateProjectRoles(ctx, plan, project.ID, []string{""})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error mapping teams to role in project"+project.ID,
			err.Error(),
		)
		return
	}

	memberTeams, err := r.client.GetTeamToProjectRole(project.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting project teams in project; "+project.ID,
			err.Error(),
		)
		return
	}
	for k, v := range memberTeams {
		plan.MemberTeams[k] = types.StringValue(v.Name)
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

	// Get refreshed order value from HashiCups
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

	memberTeams, err := r.client.GetTeamToProjectRole(project.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting teams with project roles in project; "+project.ID,
			err.Error(),
		)
		return
	}
	for k, value := range memberTeams {
		state.MemberTeams[k] = types.StringValue(value.Name)
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
	// Update existing order
	_, err := r.client.UpdateProject(project.ID, project)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Space project; "+project.ID+" is the value...",
			"Could not update project, unexpected error: "+err.Error(),
		)
		return
	}

	_, toRemove, err := CompareProjectRoles(ctx, path.Root("member_teams"), resp.State, req.Plan)

	err = r.UpdateProjectRoles(ctx, plan, project.ID, toRemove)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error mapping teams to role in project"+project.ID,
			err.Error(),
		)
		return
	}

	// Fetch updated items from GetOrder as UpdateOrder items are not
	// populated.
	p, err := r.client.GetProject(project.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Space project",
			"Could not read Space project ID "+project.ID+": "+err.Error(),
		)
		return
	}
	memberTeams, err := r.client.GetTeamToProjectRole(project.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting teams with project roles in project; "+project.ID,
			err.Error(),
		)
		return
	}
	plan.ID = types.StringValue(p.ID)
	plan.Key = types.StringValue(p.Key.Key)
	plan.Name = types.StringValue(p.Name)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	plan.Protected = types.BoolValue(plan.Protected.ValueBool())
	for k, v := range memberTeams {
		plan.MemberTeams[k] = types.StringValue(v.Name)
	}

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
		},
	}
}

func (r *projectResource) UpdateProjectRoles(ctx context.Context, plan projectResourceModel, projectID string, teamsToRemove []string) error {

	// Prepare request to map team to project role (Members)
	var members []interface{}

	members = append(members, "member")
	var empty []interface{}
	empty = append(empty, "")

	if len(teamsToRemove) > 0 {
		for _, team := range teamsToRemove {
			data := space.ProjectRoles{
				Team:        "name:" + team,
				AddRoles:    empty,
				RemoveRoles: members,
			}
			err := r.client.MapTeamToProjectRole(data, projectID)
			if err != nil {
				return err
			}
		}
	}

	for _, team := range plan.MemberTeams {

		data := space.ProjectRoles{
			Team:        "name:" + team.ValueString(),
			AddRoles:    members,
			RemoveRoles: empty,
		}

		err := r.client.MapTeamToProjectRole(data, projectID)
		if err != nil {
			return err
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
		return false, toRemove, nil
	}
	return true, []string{""}, nil

}
