package models

type StartCardinalFlags struct {
	Config     string
	Detach     bool
	LogLevel   string
	Debug      bool
	Telemetry  bool
	Editor     bool
	EditorPort string
}

type StopCardinalFlags struct {
	Config string
}

type RestartCardinalFlags struct {
	Config string
	Detach bool
	Debug  bool
}

type DevCardinalFlags struct {
	Config    string
	Editor    bool
	PrettyLog bool
}

type PurgeCardinalFlags struct {
	Config string
}

type BuildCardinalFlags struct {
	Config    string
	LogLevel  string
	Debug     bool
	Telemetry bool
	Push      string
	Auth      string
	User      string
	Pass      string
	RegToken  string
}
