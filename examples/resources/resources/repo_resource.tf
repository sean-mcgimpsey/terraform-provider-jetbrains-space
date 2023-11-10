resource "jetbrainsspace_repository" "example" {
  name = "Example"
  project_id = jetbrainsspace_project.exampleproj.id
  protected_branches = [
    {
      pattern = [
          "+:refs/heads/main",
        ],
      quality_gate = {
        approvals = [
          {
            min_approvals = 1
            approved_by = [
              "@Admins"
            ]
          }
        ]
      }
    }
  ]
}