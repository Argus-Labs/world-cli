package cloud

var CmdPlugin struct {
	Cloud *Cmd `cmd:""`
}

//nolint:lll // needed to put all the help text in the same line
type Cmd struct {
	Deploy  *DeployCmd  `cmd:"" group:"Cloud Management Commands:" help:"Deploy your World Forge project to a TEST environment in the cloud"`
	Status  *StatusCmd  `cmd:"" group:"Cloud Management Commands:" help:"Check the status of your deployed World Forge project"`
	Promote *PromoteCmd `cmd:"" group:"Cloud Management Commands:" help:"Deploy your game project to a LIVE environment in the cloud"`
	Destroy *DestroyCmd `cmd:"" group:"Cloud Management Commands:" help:"Remove your game project's deployed infrastructure from the cloud"`
	Reset   *ResetCmd   `cmd:"" group:"Cloud Management Commands:" help:"Restart your game project with a clean state"`
	Logs    *LogsCmd    `cmd:"" group:"Cloud Management Commands:" help:"Tail logs for your game project"`
}

type DeployCmd struct {
	Force bool `flag:"" help:"Force the deployment"`
}

func (c *DeployCmd) Run() error {
	// TODO: implement
	return nil
}

type StatusCmd struct {
}

func (c *StatusCmd) Run() error {
	// TODO: implement
	return nil
}

type PromoteCmd struct {
}

func (c *PromoteCmd) Run() error {
	// TODO: implement
	return nil
}

type DestroyCmd struct {
}

func (c *DestroyCmd) Run() error {
	// TODO: implement
	return nil
}

type ResetCmd struct {
}

func (c *ResetCmd) Run() error {
	// TODO: implement
	return nil
}

//nolint:lll // needed to put all the help text in the same line
type LogsCmd struct {
	Region string `arg:"" enum:"ap-southeast-1,eu-central-1,us-east-1,us-west-2" default:"us-west-2" optional:"" help:"The region to tail logs for"`
	Env    string `arg:"" enum:"test,live"                                       default:"test"      optional:"" help:"The environment to tail logs for"`
}

func (c *LogsCmd) Run() error {
	// TODO: implement
	return nil
}
