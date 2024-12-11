package editor

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// OpenEditor opens the default system editor for the given file
func OpenEditor(filename string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		switch runtime.GOOS {
		case "windows":
			editor = "notepad"
		default:
			editor = "nano"
		}
	}

	cmd := exec.Command(editor, filename)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
