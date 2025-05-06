package forge

import (
	"github.com/spf13/cobra"
)

type Credential struct {
	Token string `json:"token"`
	ID    string `json:"id"`
	Name  string `json:"name"`
}

type ForgeCommandState struct {
	Command      *cobra.Command
	User         *User
	Organization *organization
	Project      *project
}

type KnownProject struct {
	RepoURL        string `json:"repo_url"`
	RepoPath       string `json:"repo_path"`
	OrganizationID string `json:"organization_id"`
	ProjectID      string `json:"project_id"`
	ProjectName    string `json:"project_name"`
}
