package config

import "time"

type Config struct {
	OrganizationID string         `json:"organization_id"`
	ProjectID      string         `json:"project_id"`
	Credential     Credential     `json:"credential"`
	KnownProjects  []KnownProject `json:"known_projects"`
	// the following are not saved in json
	// TODO: get rid of these since they will be handled by the init flow state
	CurrRepoKnown   bool   `json:"-"` // when true, the current repo and path are already in known_projects
	CurrRepoURL     string `json:"-"`
	CurrRepoPath    string `json:"-"`
	CurrProjectName string `json:"-"`
}

type Credential struct {
	Token          string    `json:"token"`
	TokenExpiresAt time.Time `json:"token_expires_at,omitempty"`
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Email          string    `json:"email"`
}

type KnownProject struct {
	RepoURL        string `json:"repo_url"`
	RepoPath       string `json:"repo_path"`
	OrganizationID string `json:"organization_id"`
	ProjectID      string `json:"project_id"`
	ProjectName    string `json:"project_name"`
}

var _ ServiceInterface = (*Service)(nil)

type Service struct {
	Env    string
	Config Config
}

type ServiceInterface interface {
	// GetConfig returns a copy of the config
	GetConfig() *Config
	// Save saves the config to the file system
	Save() error
}
