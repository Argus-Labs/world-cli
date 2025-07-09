package config

import "pkg.world.dev/world-cli/cmd/world/internal/models"

type Config struct {
	OrganizationID string            `json:"organization_id"`
	ProjectID      string            `json:"project_id"`
	Credential     models.Credential `json:"credential"`
	KnownProjects  []KnownProject    `json:"known_projects"`
	// the following are not saved in json
	// TODO: get rid of these since they will be handled by the init flow state
	CurrRepoKnown   bool   `json:"-"` // when true, the current repo and path are already in known_projects
	CurrRepoURL     string `json:"-"`
	CurrRepoPath    string `json:"-"`
	CurrProjectName string `json:"-"`
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
	// AddKnownProject adds a known project to the config
	AddKnownProject(projectID, projectName, organizationID, repoURL, repoPath string)
	// RemoveKnownProject removes a known project from the config
	RemoveKnownProject(projectID, orgID string) error
}
