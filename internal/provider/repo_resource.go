package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	space "terraform-provider-jetbrains-space/internal/api"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &repoResource{}
	_ resource.ResourceWithConfigure   = &repoResource{}
	_ resource.ResourceWithImportState = &repoResource{}
)

// NewRepoResource is a helper function to simplify the provider implementation.
func NewRepoResource() resource.Resource {
	return &repoResource{}
}

// repoResource is the resource implementation.
type repoResource struct {
	client *space.Client
}

func (r *repoResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {

	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
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
			"project_id": schema.StringAttribute{
				Required: true,
			},
			"default_branch": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("main"),
			},
			"description": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"protected": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"protected_branches": schema.ListAttribute{
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"pattern": types.ListType{ElemType: types.StringType},
						"quality_gate": types.ObjectType{
							AttrTypes: map[string]attr.Type{
								"approvals": types.ListType{ElemType: types.ObjectType{
									AttrTypes: map[string]attr.Type{
										"min_approvals": types.Int64Type,
										"approved_by":   types.ListType{ElemType: types.StringType},
									},
								}},
							},
						},
					},
				},
				Optional: true,
			},
		},
	}
}

// Metadata returns the resource type name.
func (r *repoResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository"
}

// Create a new resource.
func (r *repoResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan.
	var plan repoResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	repoName := plan.Name.ValueString()
	projectID := plan.ProjectID.ValueString()
	protected := plan.Protected.ValueBool()
	repoData := space.CreateRepositoryData{
		Description:   plan.Description.ValueString(),
		DefaultBranch: plan.DefaultBranch.ValueString(),
		Initialize:    true,
		DefaultSetup:  true,
	}

	repo, err := r.client.CreateRepository(repoName, projectID, repoData)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating repo - "+plan.Name.String()+" ",
			err.Error(),
		)
		return
	}

	plan.Name = types.StringValue(repo.Name)
	plan.ID = types.StringValue(repo.ID)
	plan.ProjectID = types.StringValue(projectID)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	plan.Protected = types.BoolValue(protected)
	plan, err = r.UpdateRepositoryProtectedBranches(ctx, plan.ProjectID.ValueString(), plan.Name.ValueString(), plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Could not update protected branches for repository; "+plan.Name.String()+" ",
			err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *repoResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state.
	var state repoResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed repo data.
	repo, err := r.client.GetRepository(state.Name.ValueString(), state.ProjectID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Jetbrains Space repo"+state.Name.ValueString(),
			err.Error(),
		)
		return
	}

	branch, err := r.client.GetRepoProtectedBranches(state.ProjectID.ValueString(), state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating protected branches - "+state.Name.ValueString()+" ",
			err.Error(),
		)
		return
	}

	// Overwrite items with refreshed state.
	state.ID = types.StringValue(repo.ID)
	state.Name = types.StringValue(repo.Name)

	var protectedBranchesState []repoSettingsBranchModel

	for _, v := range branch.ProtectedBranches {

		var branchApprovals []repoSettingsBranchModelApprovals
		for _, va := range v.QualityGate.Approvals {

			var approvedByTF []types.String
			for _, value := range va.ApprovedBy {
				approvedByTF = append(approvedByTF, types.StringValue(value))
			}
			branchApprovals = append(branchApprovals, repoSettingsBranchModelApprovals{
				ApprovedBy:   approvedByTF,
				MinApprovals: types.Int64Value(int64(va.MinApprovals)),
			})
		}

		var patternsTF []types.String
		for _, value := range v.Pattern {
			patternsTF = append(patternsTF, types.StringValue(value))
		}

		protectedBranchesState = append(protectedBranchesState, repoSettingsBranchModel{
			Pattern: patternsTF,
			QualityGate: repoSettingsBranchModelQualityGate{
				Approvals: branchApprovals,
			},
		})

	}
	state.ProtectedBranches = protectedBranchesState

	// Set refreshed state.
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// CompareValues - Compare the values of state and plan to determine if they differ.
func CompareValues(ctx context.Context, path path.Path, state tfsdk.State, plan tfsdk.Plan) (bool, string, error) {
	var stateVal types.String
	var planVal types.String
	diag := state.GetAttribute(ctx, path, &stateVal)
	if diag.HasError() {
		return false, "", fmt.Errorf("Problem getting state value")
	}
	diag = plan.GetAttribute(ctx, path, &planVal)
	if diag.HasError() {
		return false, "", fmt.Errorf("Problem getting plan value")
	}
	if !stateVal.Equal(planVal) {
		return true, planVal.ValueString(), nil
	}
	return false, "", nil

}

// Update updates the resource and sets the updated Terraform state on success.
func (r *repoResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan.
	var plan repoResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var repo space.Repository
	name := plan.Name.ValueString()
	projectID := plan.ProjectID.ValueString()
	different, value, err := CompareValues(ctx, path.Root("description"), resp.State, req.Plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Problem comparing state and plan",
			err.Error(),
		)
	}
	if different {
		_, err := r.client.UpdateRepositoryDescription(projectID, name, value)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Updating Space repo; "+repo.ID+" is the value...",
				err.Error(),
			)
			return
		}
	}

	different, value, err = CompareValues(ctx, path.Root("default_branch"), resp.State, req.Plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Problem comparing state and plan",
			err.Error(),
		)
	}
	if different {
		_, err := r.client.UpdateRepositoryDefaultBranch(projectID, name, value)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Updating Space repo;"+repo.ID,
				err.Error(),
			)
			return
		}
	}

	err = r.client.DeleteRepositoryProtectedBranches(projectID, name)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Space repo;"+repo.ID,
			err.Error(),
		)
		return
	}

	plan, err = r.UpdateRepositoryProtectedBranches(ctx, plan.ProjectID.ValueString(), plan.Name.ValueString(), plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Problem updating protected branches.",
			err.Error(),
		)
		return
	}
	p, err := r.client.GetRepository(name, projectID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Space project"+projectID,
			err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(p.ID)
	plan.Name = types.StringValue(p.Name)
	plan.Description = types.StringValue(p.Description)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	plan.Protected = types.BoolValue(plan.Protected.ValueBool())

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *repoResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state.
	var state repoResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.Protected.ValueBool() {
		resp.Diagnostics.AddError(
			"Repo is protected, not deleting!",
			"",
		)
	} else {
		err := r.client.DeleteRepository(state.ProjectID.ValueString(), state.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Deleting Space Repo"+state.Name.ValueString(),
				err.Error(),
			)
			return
		}
	}
}

func (r *repoResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, ",")

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: attr_one,attr_two. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[1])...)
}

func (r *repoResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *repoResource) UpdateRepositoryProtectedBranches(ctx context.Context, ProjectID string, Repository string, plan repoResourceModel) (repoResourceModel, error) {

	var planBranches []space.ProtectedBranchesReq
	var planBranchesApprovals []space.ProtectedBranchesResultApprovals
	var members = []string{"@Members"}
	var admins = []string{"@Admins"}
	var data = space.ProtectedBranchesPost{}
	for k := range plan.ProtectedBranches {
		var patterns []string
		for _, va := range plan.ProtectedBranches[k].Pattern {
			patterns = append(patterns, va.ValueString())
		}
		for _, va := range plan.ProtectedBranches[k].QualityGate.Approvals {
			var appr []string
			for _, val := range va.ApprovedBy {
				appr = append(appr, val.ValueString())
			}

			planBranchesApprovals = append(planBranchesApprovals, space.ProtectedBranchesResultApprovals{
				ApprovedBy:   appr,
				MinApprovals: int(va.MinApprovals.ValueInt64()),
			})
		}
		planBranches = append(planBranches, space.ProtectedBranchesReq{
			Pattern:        patterns,
			AllowPush:      admins,
			AllowCreate:    members,
			AllowDelete:    admins,
			AllowForcePush: admins,
			QualityGate: space.ProtectedBranchesQualityGate{
				Approvals: planBranchesApprovals,
			},
		})

		data = space.ProtectedBranchesPost{
			Settings: space.ProtectedBranchesSettings{
				Version:           "1.0",
				ProtectedBranches: planBranches,
			},
		}

	}

	branch, err := r.client.UpdateRepoProtectedBranches(data, ProjectID, Repository)
	if err != nil {
		return repoResourceModel{}, fmt.Errorf("Could not update repos protected branches, unexpected error: " + err.Error())
	}

	for k, v := range branch.ProtectedBranches {
		var branchApprovals []repoSettingsBranchModelApprovals
		for _, va := range v.QualityGate.Approvals {

			var approvedByTF []types.String
			for _, value := range va.ApprovedBy {
				approvedByTF = append(approvedByTF, types.StringValue(value))
			}
			branchApprovals = append(branchApprovals, repoSettingsBranchModelApprovals{
				ApprovedBy:   approvedByTF,
				MinApprovals: types.Int64Value(int64(va.MinApprovals)),
			})
		}
		var patternsTF []types.String
		for _, value := range v.Pattern {
			patternsTF = append(patternsTF, types.StringValue(value))
		}
		plan.ProtectedBranches[k] = repoSettingsBranchModel{
			Pattern: patternsTF,
			QualityGate: repoSettingsBranchModelQualityGate{
				Approvals: branchApprovals,
			},
		}
	}
	return plan, nil
}
