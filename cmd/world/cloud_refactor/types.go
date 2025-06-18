package cloud

// Interface guard.
var _ HandlerInterface = (*Handler)(nil)

type Handler struct {
}

type HandlerInterface interface {
	Deploy(force bool) error
	Status() error
	Promote() error
	Destroy() error
	Reset() error
	Logs(region string, env string) error
}
