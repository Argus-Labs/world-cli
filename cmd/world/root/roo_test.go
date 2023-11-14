package root

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

// outputFromCmd runs the rootCmd with the given cmd arguments and returns the output of the command along with
// any errors.
func outputFromCmd(t *testing.T, cmd string) (lines []string, err error) {
	stdOut := &bytes.Buffer{}
	stdErr := &bytes.Buffer{}
	rootCmd.SetOut(stdOut)
	defer func() {
		rootCmd.SetOut(nil)
	}()
	rootCmd.SetErr(stdErr)
	defer func() {
		rootCmd.SetErr(nil)
	}()
	rootCmd.SetArgs(strings.Split(cmd, " "))
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	assert.NilError(t, rootCmd.Execute())
	lines = strings.Split(stdOut.String(), "\n")
	errorStr := stdErr.String()
	if len(errorStr) > 0 {
		err = errors.New(errorStr)
	}
	return lines, err
}

func TestSubcommandsHaveHelpText(t *testing.T) {
	lines, err := outputFromCmd(t, "help")
	assert.NilError(t, err)
	seenSubcommands := map[string]int{
		"cardinal":   0,
		"completion": 0,
		"doctor":     0,
		"help":       0,
		"version":    0,
	}

	for _, line := range lines {
		for subcommand := range seenSubcommands {
			if strings.HasPrefix(line, "  "+subcommand) {
				seenSubcommands[subcommand]++
			}
		}
	}

	for subcommand, count := range seenSubcommands {
		assert.Check(t, count > 0, "subcommand %q is not listed in the help command", subcommand)
	}
}
