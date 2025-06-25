package project

import "pkg.world.dev/world-cli/cmd/internal/interfaces"

// Interface guard.
var _ interfaces.ProjectHandler = (*Handler)(nil)

type Handler struct {
}
