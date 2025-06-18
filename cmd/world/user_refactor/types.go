package user

import "pkg.world.dev/world-cli/cmd/internal/interfaces"

// Interface guard.
var _ interfaces.UserHandler = (*Handler)(nil)

type Handler struct {
}
