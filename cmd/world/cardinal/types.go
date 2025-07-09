package cardinal

import "pkg.world.dev/world-cli/cmd/world/internal/interfaces"

var _ interfaces.CardinalHandler = &Handler{}

type Handler struct {
}

func NewHandler() interfaces.CardinalHandler {
	return &Handler{}
}
