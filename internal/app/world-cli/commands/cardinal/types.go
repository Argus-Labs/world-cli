package cardinal

import "pkg.world.dev/world-cli/internal/app/world-cli/interfaces"

var _ interfaces.CardinalHandler = &Handler{}

type Handler struct {
}

func NewHandler() interfaces.CardinalHandler {
	return &Handler{}
}
