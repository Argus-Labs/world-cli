package models

type DeploymentPreview struct {
	OrgName        string   `json:"org_name"`
	OrgSlug        string   `json:"org_slug"`
	ProjectName    string   `json:"project_name"`
	ProjectSlug    string   `json:"project_slug"`
	ExecutorName   string   `json:"executor_name"`
	DeploymentType string   `json:"deployment_type"`
	TickRate       int      `json:"tick_rate"`
	Regions        []string `json:"regions"`
}

// Parse the response into a map of environment names to their status.
type DeploymentHealthCheckResult struct {
	OK      bool `json:"ok"`
	Offline bool `json:"offline"`
}

type TemporaryCredential struct {
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	SessionToken    string `json:"session_token"`
	Region          string `json:"region"`
	RepoURI         string `json:"repo_uri"`
}

const (
	DeploymentTypeDeploy      = "deploy"
	DeploymentTypeForceDeploy = "forceDeploy"
	DeploymentTypeDestroy     = "destroy"
	DeploymentTypeReset       = "reset"
	DeploymentTypePromote     = "promote"
)
