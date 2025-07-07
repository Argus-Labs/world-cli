package evm

import "pkg.world.dev/world-cli/cmd/internal/interfaces"

var _ interfaces.EVMHandler = (*Handler)(nil)

type Handler struct {
}

func NewHandler() interfaces.EVMHandler {
	return &Handler{}
}
