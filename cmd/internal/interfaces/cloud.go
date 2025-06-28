package interfaces

import (
	"context"

	"pkg.world.dev/world-cli/cmd/internal/models"
)

type CloudHandler interface {
	Deployment(ctx context.Context, organizationID string, project models.Project, deployType string) error
	Status(ctx context.Context, organization models.Organization, project models.Project) error
	TailLogs(ctx context.Context, region string, env string) error
}
