package organization

import (
	"pkg.world.dev/world-cli/cmd/internal/interfaces"
)

// Interface guard.
var _ interfaces.OrganizationHandler = (*Handler)(nil)

type Handler struct {
	ProjectHandler interfaces.ProjectHandler
}
