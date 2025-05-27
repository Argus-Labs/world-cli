package forge

import (
	"time"
)

type Credential struct {
	Token          string    `json:"token"`
	TokenExpiresAt time.Time `json:"token_expires_at,omitempty"`
	ID             string    `json:"id"`
	Name           string    `json:"name"`
}

type CommandState struct {
	LoggedIn      bool
	CurrRepoKnown bool
	User          *User
	Organization  *organization
	Project       *project
}

type KnownProject struct {
	RepoURL        string `json:"repo_url"`
	RepoPath       string `json:"repo_path"`
	OrganizationID string `json:"organization_id"`
	ProjectID      string `json:"project_id"`
	ProjectName    string `json:"project_name"`
}
