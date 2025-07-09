package interfaces

import (
	"context"

	"pkg.world.dev/world-cli/cmd/world/internal/models"
)

// CloudHandler defines the interface for cloud deployment and management operations.
type CloudHandler interface {
	// Deployment handles project deployment operations (deploy, destroy, reset, promote).
	Deployment(ctx context.Context, organizationID string, project models.Project, deployType string) error

	// Status displays the current deployment status for a project.
	Status(ctx context.Context, organization models.Organization, project models.Project) error

	// TailLogs streams logs from a specific deployment environment.
	TailLogs(ctx context.Context, region string, env string) error
}
