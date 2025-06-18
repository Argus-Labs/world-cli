package root

import "pkg.world.dev/world-cli/cmd/internal/interfaces"

// Interface guard.
var _ interfaces.RootHandler = (*Handler)(nil)

type Handler struct {
}
