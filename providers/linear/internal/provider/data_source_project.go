package provider

import (
	"context"
	"errors"

	"github.com/benfdking/sdsf/providers/linear/internal/linear"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &projectDataSource{}
var _ datasource.DataSourceWithConfigure = &projectDataSource{}

type projectDataSource struct {
	client *linear.Client
}

type projectDataSourceModel struct {
	ID          types.String  `tfsdk:"id"`
	ProjectID   types.String  `tfsdk:"project_id"`
	SlugID      types.String  `tfsdk:"slug_id"`
	Name        types.String  `tfsdk:"name"`
	Description types.String  `tfsdk:"description"`
	URL         types.String  `tfsdk:"url"`
	Progress    types.Float64 `tfsdk:"progress"`
	StartDate   types.String  `tfsdk:"start_date"`
	TargetDate  types.String  `tfsdk:"target_date"`
	StatusID    types.String  `tfsdk:"status_id"`
	StatusName  types.String  `tfsdk:"status_name"`
	StatusType  types.String  `tfsdk:"status_type"`
	StatusColor types.String  `tfsdk:"status_color"`
	LeadID      types.String  `tfsdk:"lead_id"`
	TeamIDs     types.Set     `tfsdk:"team_ids"`
}

func NewProjectDataSource() datasource.DataSource { return &projectDataSource{} }

func (d *projectDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (d *projectDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a Linear project by UUID or URL slug ID.",
		Attributes: map[string]schema.Attribute{
			"id":           schema.StringAttribute{Computed: true, Description: "The project's UUID."},
			"project_id":   schema.StringAttribute{Optional: true, Description: "UUID of the project to look up. Exactly one of project_id or slug_id must be set."},
			"slug_id":      schema.StringAttribute{Optional: true, Computed: true, Description: "The project's unique URL slug. Exactly one of project_id or slug_id must be set."},
			"name":         schema.StringAttribute{Computed: true, Description: "Project name."},
			"description":  schema.StringAttribute{Computed: true, Description: "Project summary."},
			"url":          schema.StringAttribute{Computed: true, Description: "Project URL."},
			"progress":     schema.Float64Attribute{Computed: true, Description: "Project completion progress from zero to one."},
			"start_date":   schema.StringAttribute{Computed: true, Description: "Estimated project start date."},
			"target_date":  schema.StringAttribute{Computed: true, Description: "Estimated project target date."},
			"status_id":    schema.StringAttribute{Computed: true, Description: "Project status UUID."},
			"status_name":  schema.StringAttribute{Computed: true, Description: "Project status name."},
			"status_type":  schema.StringAttribute{Computed: true, Description: "Project status type."},
			"status_color": schema.StringAttribute{Computed: true, Description: "Project status color."},
			"lead_id":      schema.StringAttribute{Computed: true, Description: "UUID of the project lead."},
			"team_ids": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "UUIDs of the teams associated with the project.",
			},
		},
	}
}

func (d *projectDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*clientData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", "The Linear provider returned an unexpected data source configuration type.")
		return
	}
	d.client = data.client
}

func nullableString(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

func projectState(ctx context.Context, project *linear.Project) (projectDataSourceModel, error) {
	teamIDs := make([]string, 0, len(project.Teams.Nodes))
	for _, team := range project.Teams.Nodes {
		teamIDs = append(teamIDs, team.ID)
	}
	teamIDSet, diagnostics := types.SetValueFrom(ctx, types.StringType, teamIDs)
	if diagnostics.HasError() {
		return projectDataSourceModel{}, errors.New("unable to convert Linear project team IDs to Terraform values")
	}
	state := projectDataSourceModel{
		ID:          types.StringValue(project.ID),
		SlugID:      types.StringValue(project.SlugID),
		Name:        types.StringValue(project.Name),
		Description: nullableString(project.Description),
		URL:         types.StringValue(project.URL),
		Progress:    types.Float64Value(project.Progress),
		StartDate:   nullableString(project.StartDate),
		TargetDate:  nullableString(project.TargetDate),
		TeamIDs:     teamIDSet,
	}
	if project.Status == nil {
		state.StatusID = types.StringNull()
		state.StatusName = types.StringNull()
		state.StatusType = types.StringNull()
		state.StatusColor = types.StringNull()
	} else {
		state.StatusID = types.StringValue(project.Status.ID)
		state.StatusName = types.StringValue(project.Status.Name)
		state.StatusType = types.StringValue(project.Status.Type)
		state.StatusColor = types.StringValue(project.Status.Color)
	}
	if project.Lead == nil {
		state.LeadID = types.StringNull()
	} else {
		state.LeadID = types.StringValue(project.Lead.ID)
	}
	return state, nil
}

func (d *projectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config projectDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	hasID := !config.ProjectID.IsNull() && config.ProjectID.ValueString() != ""
	hasSlug := !config.SlugID.IsNull() && !config.SlugID.IsUnknown() && config.SlugID.ValueString() != ""
	if hasID == hasSlug {
		resp.Diagnostics.AddError("Invalid project lookup", "Configure exactly one of project_id or slug_id.")
		return
	}

	var project *linear.Project
	var err error
	if hasID {
		project, err = d.client.Project(ctx, config.ProjectID.ValueString())
	} else {
		project, err = d.client.ProjectBySlugID(ctx, config.SlugID.ValueString())
	}
	if err != nil {
		if errors.Is(err, linear.ErrNotFound) {
			resp.Diagnostics.AddError("Linear project not found", "No Linear project matched the configured project_id or slug_id.")
		} else {
			resp.Diagnostics.AddError("Unable to read Linear project", err.Error())
		}
		return
	}

	state, err := projectState(ctx, project)
	if err != nil {
		resp.Diagnostics.AddError("Unable to store Linear project", err.Error())
		return
	}
	state.ProjectID = config.ProjectID
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
