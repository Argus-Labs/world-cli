package root

import "pkg.world.dev/world-cli/cmd/world/pkg/models"

// Interface guard.
var _ HandlerInterface = (*Handler)(nil)

type Handler struct {
}

type HandlerInterface interface {
	Create(directory string) error
	Doctor() error
	Version(check bool) error
	Login(ctx models.CommandContext) error
}
