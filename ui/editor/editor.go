package editor

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/config"
)

const (
	EditorDir = "cardinal-editor"
)

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

func SetupCardinalEditor(rootDir, gameDir string) error {
	editorPath := filepath.Join(rootDir, EditorDir)
	if err := os.MkdirAll(editorPath, config.EditorDirPerm); err != nil {
		return eris.Wrapf(err, "failed to create editor directory at %s", editorPath)
	}

	gamePath := filepath.Join(rootDir, gameDir)
	if err := os.Symlink(gamePath, filepath.Join(editorPath, "game")); err != nil && !os.IsExist(err) {
		return eris.Wrapf(err, "failed to create symlink from %s to %s", gamePath, editorPath)
	}

	return nil
}
