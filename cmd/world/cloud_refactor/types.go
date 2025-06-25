package cloud

import "pkg.world.dev/world-cli/cmd/internal/interfaces"

// Interface guard.
var _ interfaces.CloudHandler = (*Handler)(nil)

type Handler struct {
}
