package models

type Project struct {
	ID           string        `json:"id"`
	OrgID        string        `json:"org_id"`
	OwnerID      string        `json:"owner_id"`
	Name         string        `json:"name"`
	Slug         string        `json:"slug"`
	CreatedTime  string        `json:"created_time"`
	UpdatedTime  string        `json:"updated_time"`
	Deleted      bool          `json:"deleted"`
	DeletedTime  string        `json:"deleted_time"`
	RepoURL      string        `json:"repo_url"`
	RepoToken    string        `json:"repo_token"`
	RepoPath     string        `json:"repo_path"`
	DeploySecret string        `json:"deploy_secret,omitempty"`
	Config       ProjectConfig `json:"config"`
	AvatarURL    string        `json:"avatar_url"`

	Update bool `json:"-"`
}

type ProjectConfig struct {
	Region  []string             `json:"region"`
	Discord ProjectConfigDiscord `json:"discord"`
	Slack   ProjectConfigSlack   `json:"slack"`
}

type ProjectConfigDiscord struct {
	Enabled bool   `json:"enabled"`
	Token   string `json:"token"`
	Channel string `json:"channel"`
}

type ProjectConfigSlack struct {
	Enabled bool   `json:"enabled"`
	Token   string `json:"token"`
	Channel string `json:"channel"`
}

type CreateProjectFlags struct {
	Name      string
	Slug      string
	AvatarURL string
}

type SwitchProjectFlags struct {
	Slug string
}

type UpdateProjectFlags struct {
	Name      string
	Slug      string
	AvatarURL string
}
