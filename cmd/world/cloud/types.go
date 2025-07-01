package cloud

import (
	"pkg.world.dev/world-cli/cmd/internal/clients/api"
	"pkg.world.dev/world-cli/cmd/internal/interfaces"
	"pkg.world.dev/world-cli/cmd/internal/models"
	"pkg.world.dev/world-cli/cmd/internal/services/config"
	"pkg.world.dev/world-cli/cmd/internal/services/input"
)

// Interface guard.
var _ interfaces.CloudHandler = (*Handler)(nil)

type Handler struct {
	apiClient      api.ClientInterface
	configService  config.ServiceInterface
	projectHandler interfaces.ProjectHandler
	inputHandler   input.ServiceInterface
}

const (
	DeploymentTypeDeploy      = "deploy"
	DeploymentTypeForceDeploy = "forceDeploy"
	DeploymentTypeDestroy     = "destroy"
	DeploymentTypeReset       = "reset"
	DeploymentTypePromote     = "promote"

	DeployStatusFailed  DeployStatus = "failed"
	DeployStatusCreated DeployStatus = "created"
	DeployStatusRemoved DeployStatus = "removed"

	DeployEnvPreview = "dev"
	DeployEnvLive    = "prod"
)

type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusOffline   HealthStatus = "offline"
)

type DeployStatus string

type DeployInfo struct {
	DeployType    string
	DeployStatus  DeployStatus
	DeployDisplay string
}

type logParams struct {
	organization models.Organization
	project      models.Project
	region       string
	env          string
}

func NewHandler(
	apiClient api.ClientInterface,
	configService config.ServiceInterface,
	projectHandler interfaces.ProjectHandler,
	inputHandler input.ServiceInterface,
) *Handler {
	return &Handler{
		apiClient:      apiClient,
		configService:  configService,
		projectHandler: projectHandler,
		inputHandler:   inputHandler,
	}
}
