package cardinal

import (
	"context"

	cmdsetup "pkg.world.dev/world-cli/cmd/world/internal/controllers/cmd_setup"
	"pkg.world.dev/world-cli/cmd/world/internal/models"
	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/dependency"
	"pkg.world.dev/world-cli/common/docker/service"
)

var CardinalCmdPlugin struct {
	Cardinal *CardinalCmd `cmd:"" group:"Cardinal Commands:" help:"Manage your Cardinal game shard"`
}

//nolint:lll, revive // needed to put all the help text in the same line
type CardinalCmd struct {
	Config       string                `flag:"" type:"existingfile" help:"A TOML config file"`
	Context      context.Context       `                                                      kong:"-"`
	Dependencies cmdsetup.Dependencies `                                                      kong:"-"`

	Start   *StartCmd   `cmd:"" group:"Cardinal Commands:" help:"Launch your Cardinal game environment"`
	Stop    *StopCmd    `cmd:"" group:"Cardinal Commands:" help:"Gracefully shut down your Cardinal game environment"`
	Restart *RestartCmd `cmd:"" group:"Cardinal Commands:" help:"Restart your Cardinal game environment"`
	Dev     *DevCmd     `cmd:"" group:"Cardinal Commands:" help:"Run Cardinal in fast development mode with hot reloading"`
	Purge   *PurgeCmd   `cmd:"" group:"Cardinal Commands:" help:"Reset your Cardinal game shard to a clean state by removing all data and containers"`
	Build   *BuildCmd   `cmd:"" group:"Cardinal Commands:" help:"Build and package your Cardinal game into production-ready Docker images"`
}

func (c *CardinalCmd) Run() error {
	return dependency.Check(
		dependency.Go,
		dependency.Git,
		dependency.Docker,
		dependency.DockerDaemon,
	)
}

//nolint:lll // needed to put all the help text in the same line
type StartCmd struct {
	Parent     *CardinalCmd `kong:"-"`
	Detach     bool         `         flag:"" help:"Run in detached mode"`
	LogLevel   string       `         flag:"" help:"Set the log level for Cardinal"`
	Debug      bool         `         flag:"" help:"Enable delve debugging"`
	Telemetry  bool         `         flag:"" help:"Enable tracing, metrics, and profiling"`
	Editor     bool         `         flag:"" help:"Run Cardinal Editor, useful for prototyping and debugging"`
	EditorPort string       `         flag:"" help:"Port for Cardinal Editor"                                  default:"auto"`
}

func (c *StartCmd) Run() error {
	flags := models.StartCardinalFlags{
		Config:     c.Parent.Config,
		Detach:     c.Detach,
		LogLevel:   c.LogLevel,
		Debug:      c.Debug,
		Telemetry:  c.Telemetry,
		Editor:     c.Editor,
		EditorPort: c.EditorPort,
	}
	return c.Parent.Dependencies.CardinalHandler.Start(c.Parent.Context, flags)
}

type StopCmd struct {
	Parent *CardinalCmd `kong:"-"`
}

func (c *StopCmd) Run() error {
	flags := models.StopCardinalFlags{
		Config: c.Parent.Config,
	}
	return c.Parent.Dependencies.CardinalHandler.Stop(c.Parent.Context, flags)
}

type RestartCmd struct {
	Parent *CardinalCmd `kong:"-"`
	Detach bool         `         flag:"" help:"Run in detached mode"`
	Debug  bool         `         flag:"" help:"Enable debugging"`
}

func (c *RestartCmd) Run() error {
	flags := models.RestartCardinalFlags{
		Config: c.Parent.Config,
		Detach: c.Detach,
		Debug:  c.Debug,
	}
	return c.Parent.Dependencies.CardinalHandler.Restart(c.Parent.Context, flags)
}

type DevCmd struct {
	Parent    *CardinalCmd `kong:"-"`
	Editor    bool         `         flag:"" help:"Enable Cardinal Editor"`
	PrettyLog bool         `         flag:"" help:"Run Cardinal with pretty logging" default:"true"`
}

func (c *DevCmd) Run() error {
	flags := models.DevCardinalFlags{
		Config:    c.Parent.Config,
		Editor:    c.Editor,
		PrettyLog: c.PrettyLog,
	}
	return c.Parent.Dependencies.CardinalHandler.Dev(c.Parent.Context, flags)
}

type PurgeCmd struct {
	Parent *CardinalCmd `kong:"-"`
}

func (c *PurgeCmd) Run() error {
	flags := models.PurgeCardinalFlags{
		Config: c.Parent.Config,
	}
	return c.Parent.Dependencies.CardinalHandler.Purge(c.Parent.Context, flags)
}

type BuildCmd struct {
	Parent    *CardinalCmd `kong:"-"`
	LogLevel  string       `         flag:"" help:"Set the log level for Cardinal"`
	Debug     bool         `         flag:"" help:"Enable debugging"`
	Telemetry bool         `         flag:"" help:"Enable tracing, metrics, and profiling"`
	Push      string       `         flag:"" help:"Push your cardinal image to a given image repository" hidden:"true"`
	Auth      string       `         flag:"" help:"Auth token for the given image repository"            hidden:"true"`
	User      string       `         flag:"" help:"User for the given image repository"                  hidden:"true"`
	Pass      string       `         flag:"" help:"Password for the given image repository"              hidden:"true"`
	RegToken  string       `         flag:"" help:"Registry token for the given image repository"        hidden:"true"`
}

func (c *BuildCmd) Run() error {
	flags := models.BuildCardinalFlags{
		Config:    c.Parent.Config,
		LogLevel:  c.LogLevel,
		Debug:     c.Debug,
		Telemetry: c.Telemetry,
		Push:      c.Push,
		Auth:      c.Auth,
		User:      c.User,
		Pass:      c.Pass,
		RegToken:  c.RegToken,
	}
	return c.Parent.Dependencies.CardinalHandler.Build(c.Parent.Context, flags)
}

func getServices(cfg *config.Config) []service.Builder {
	services := []service.Builder{service.NakamaDB, service.Redis, service.Cardinal, service.Nakama}
	if cfg.Telemetry && cfg.DockerEnv["NAKAMA_TRACE_ENABLED"] == "true" {
		services = append(services, service.Jaeger)
	}
	if cfg.Telemetry && cfg.DockerEnv["NAKAMA_METRICS_ENABLED"] == "true" {
		services = append(services, service.Prometheus)
	}
	return services
}

func getCardinalServices(_ *config.Config) []service.Builder {
	services := []service.Builder{service.Cardinal}
	return services
}
