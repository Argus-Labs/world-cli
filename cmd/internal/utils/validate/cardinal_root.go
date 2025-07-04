package validate

import (
	"os"
	"path/filepath"

	"github.com/rotisserie/eris"
)

var ErrNotInWorldCardinalRoot = eris.New("Not in a World Cardinal root")

// IsInWorldCardinalRoot checks if the current working directory is a World project.
// It checks for the presence of world.toml and cardinal directory.
func IsInWorldCardinalRoot() (bool, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return false, eris.Wrap(err, "failed to get working directory")
	}

	worldTomlPath := filepath.Join(cwd, "world.toml")
	cardinalDirPath := filepath.Join(cwd, "cardinal")

	tomlInfo, err := os.Stat(worldTomlPath)
	if err != nil || tomlInfo.IsDir() {
		return false, nil //nolint:nilerr // false return gives all the information we need
	}

	cardinalInfo, err := os.Stat(cardinalDirPath)
	if err != nil || !cardinalInfo.IsDir() {
		return false, nil //nolint:nilerr // false return gives all the information we need
	}
	return true, nil
}
