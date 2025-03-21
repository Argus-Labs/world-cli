package logger

import (
	"bytes"
	"fmt"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const (
	DefaultTimeFormat           = "15:04:05.000"
	DefaultCallerSkipFrameCount = 3 // set to 3 because logger wrapped in logger.go

	NoColor   = true
	UseCaller = false // for developer, if you want to expose line of code of caller
)

var (
	logBuffer bytes.Buffer

	// VerboseMode flag for determining verbose logging
	verboseMode = false
)

func init() {
	var (
		lgr zerolog.Logger
	)

	zerolog.TimeFieldFormat = DefaultTimeFormat
	zerolog.CallerSkipFrameCount = DefaultCallerSkipFrameCount

	var writers zerolog.LevelWriter
	consoleWriter := zerolog.ConsoleWriter{
		Out:        &logBuffer,
		NoColor:    NoColor,
		TimeFormat: DefaultTimeFormat,
	}
	writers = zerolog.MultiLevelWriter(consoleWriter)

	lgr = zerolog.New(writers)

	if UseCaller {
		lgr = lgr.With().Caller().Logger()
	}

	log.Logger = lgr
}

// PrintLogs print all stacked log
func PrintLogs() {
	if verboseMode {
		// Extract the logs from the buffer and print them
		logs := logBuffer.String()
		if len(logs) > 0 {
			fmt.Println("\n----- Log -----")
			fmt.Println(logs)
		}
	}
}

// AddVerboseFlag set flag --log-debug
func AddVerboseFlag(cmd ...*cobra.Command) {
	for _, c := range cmd {
		c.Flags().BoolVarP(&verboseMode, "verbose", "v", false, "Enable World CLI debug logs")
	}
}
