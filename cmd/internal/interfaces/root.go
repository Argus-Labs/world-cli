package interfaces

import "context"

type RootHandler interface {
	Create(directory string) error
	Doctor() error
	Version(check bool) error
	Login(ctx context.Context) error
}
