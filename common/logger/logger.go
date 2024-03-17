package logger

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

// Debug function
func Debug(args ...interface{}) {
	log.Debug().Timestamp().Msg(fmt.Sprint(args...))
}

// Debugln function
func Debugln(args ...interface{}) {
	log.Debug().Timestamp().Msg(fmt.Sprintln(args...))
}

// Debugf function
func Debugf(format string, v ...interface{}) {
	log.Debug().Timestamp().Msgf(format, v...)
}

// DebugWithFields function
func DebugWithFields(msg string, kv map[string]interface{}) {
	log.Debug().Timestamp().Fields(kv).Msg(msg)
}

// Info function
func Info(args ...interface{}) {
	log.Info().Timestamp().Msg(fmt.Sprint(args...))
}

// Infoln function
func Infoln(args ...interface{}) {
	log.Info().Timestamp().Msg(fmt.Sprintln(args...))
}

// Infof function
func Infof(format string, v ...interface{}) {
	log.Info().Timestamp().Msgf(format, v...)
}

// InfoWithFields function
func InfoWithFields(msg string, kv map[string]interface{}) {
	log.Info().Timestamp().Fields(kv).Msg(msg)
}

// Warn function
func Warn(args ...interface{}) {
	log.Warn().Timestamp().Msg(fmt.Sprint(args...))
}

// Warnln function
func Warnln(args ...interface{}) {
	log.Warn().Timestamp().Msg(fmt.Sprintln(args...))
}

// Warnf function
func Warnf(format string, v ...interface{}) {
	log.Warn().Timestamp().Msgf(format, v...)
}

// WarnWithFields function
func WarnWithFields(msg string, kv map[string]interface{}) {
	log.Warn().Timestamp().Fields(kv).Msg(msg)
}

// Error function
func Error(args ...interface{}) {
	log.Error().Timestamp().Msg(fmt.Sprint(args...))
}

// Errorln function
func Errorln(args ...interface{}) {
	log.Error().Timestamp().Msg(fmt.Sprintln(args...))
}

// Errorf function
func Errorf(format string, v ...interface{}) {
	log.Error().Timestamp().Msgf(format, v...)
}

// ErrorWithFields function
func ErrorWithFields(msg string, kv map[string]interface{}) {
	log.Error().Timestamp().Fields(kv).Msg(msg)
}

// Errors function to log errors package
func Errors(err error) {
	log.Error().Timestamp().Msg(err.Error())
}

// Fatal function
func Fatal(args ...interface{}) {
	log.Fatal().Timestamp().Msg(fmt.Sprint(args...))
}

// Fatalln function
func Fatalln(args ...interface{}) {
	log.Fatal().Timestamp().Msg(fmt.Sprintln(args...))
}

// Fatalf function
func Fatalf(format string, v ...interface{}) {
	log.Fatal().Timestamp().Msgf(format, v...)
}

// FatalWithFields function
func FatalWithFields(msg string, kv map[string]interface{}) {
	log.Fatal().Timestamp().Fields(kv).Msg(msg)
}

// Printf standard printf with debug mode validation
func Printf(format string, v ...interface{}) {
	if DebugMode {
		fmt.Printf(format, v...)
	}
}

// Println standard println with debug mode validation
func Println(v ...interface{}) {
	if DebugMode {
		fmt.Println(v...)
	}
}

// Print standard print with debug mode validation
func Print(v ...interface{}) {
	if DebugMode {
		fmt.Print(v...)
	}
}
