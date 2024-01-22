package root

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"go.uber.org/mock/gomock"
	"pkg.world.dev/world-cli/internal/teacmd"
	mock_terminal "pkg.world.dev/world-cli/utils/terminal/mock"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

// outputFromCmd runs the rootCmd with the given cmd arguments and returns the output of the command along with
// any errors.
func outputFromRootCmd(strcmd string) (lines []string, err error) {
	rootCmd := New()

	stdOut := &bytes.Buffer{}
	stdErr := &bytes.Buffer{}
	rootCmd.SetOut(stdOut)
	rootCmd.SetErr(stdErr)
	rootCmd.SetArgs(strings.Split(strcmd, " "))
	if err = rootCmd.Execute(); err != nil {
		return nil, fmt.Errorf("root command failed with: %w", err)
	}
	lines = strings.Split(stdOut.String(), "\n")
	errorStr := stdErr.String()
	if len(errorStr) > 0 {
		err = errors.New(errorStr)
	}
	return lines, err
}

// outputFromCmd runs the rootCmd with the given cmd arguments and returns the output of the command along with
// any errors.
func outputFromCmd(cmd *cobra.Command, strcmd string) (lines []string, err error) {
	stdOut := &bytes.Buffer{}
	stdErr := &bytes.Buffer{}
	cmd.SetOut(stdOut)
	cmd.SetErr(stdErr)
	cmd.SetArgs(strings.Split(strcmd, " "))
	if err = cmd.Execute(); err != nil {
		return nil, fmt.Errorf("root command failed with: %w", err)
	}
	lines = strings.Split(stdOut.String(), "\n")
	errorStr := stdErr.String()
	if len(errorStr) > 0 {
		err = errors.New(errorStr)
	}
	return lines, err
}

func TestRoot(t *testing.T) {
	// Test Success
	t.Run("success", func(t *testing.T) {
		lines, err := outputFromRootCmd("")
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
	})

	// Test Error
	t.Run("error", func(t *testing.T) {
		_, err := outputFromRootCmd("error")
		assert.ErrorContains(t, err, "root command failed with: unknown command \"error\" for \"world\"")
	})
}

func TestDoctor(t *testing.T) {
	// Init Mock
	terminalMock := mock_terminal.NewMockTerminal(gomock.NewController(t))

	// Test Success
	t.Run("success", func(t *testing.T) {
		teaCmd := teacmd.New(terminalMock)
		cmd := doctorCmd(teaCmd)

		terminalMock.EXPECT().ExecCmd(gomock.Any()).Return([]byte(""), nil).AnyTimes()

		err := cmd.Execute()
		assert.NilError(t, err)

	})
}

func TestVersion(t *testing.T) {
	// Test Success
	t.Run("success", func(t *testing.T) {
		lines, err := outputFromRootCmd("version")
		assert.NilError(t, err)
		assert.Equal(t, len(lines), 1) // have 1 line for version
	})
}
