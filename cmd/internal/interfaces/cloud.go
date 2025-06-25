package interfaces

type CloudHandler interface {
	Deploy(force bool) error
	Status() error
	Promote() error
	Destroy() error
	Reset() error
	Logs(region string, env string) error
}
