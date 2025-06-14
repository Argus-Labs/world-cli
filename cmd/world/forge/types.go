package forge

import (
	"context"
	"time"

	"github.com/rotisserie/eris"
)

var ErrContextCanceled = eris.New("context canceled")

//nolint:revive // Name makes sense and is generally used within package
type ForgeContext struct {
	Context context.Context
	State   CommandState
	Config  *Config
}

type Credential struct {
	Token          string    `json:"token"`
	TokenExpiresAt time.Time `json:"token_expires_at,omitempty"`
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Email          string    `json:"email"`
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
