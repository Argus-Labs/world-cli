//nolint:reassign,forbidigo // customizing zerolog globals is safe and intentional for this CLI
package logger

import (
	"bytes"
	"fmt"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	DefaultTimeFormat           = "15:04:05.000"
	DefaultCallerSkipFrameCount = 3 // set to 3 because logger wrapped in logger.go

	NoColor   = true
	UseCaller = false // for developer, if you want to expose line of code of caller
)

var (
	logBuffer bytes.Buffer

	// VerboseMode flag for determining verbose logging.
	VerboseMode = false
)

//nolint:gochecknoinits // Common package init, should self init as it shouldn't have dependencies..
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

// PrintLogs print all stacked log.
func PrintLogs() {
	if VerboseMode {
		// Extract the logs from the buffer and print them
		logs := logBuffer.String()
		if len(logs) > 0 {
			fmt.Println()
			fmt.Println("----- Log -----")
			fmt.Println(logs)
		}
	}
}
