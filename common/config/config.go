package config

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/pelletier/go-toml"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
	"pkg.world.dev/world-cli/common/logger"
)

const (
	WorldCLIConfigFileEnvVariable = "WORLD_CLI_CONFIG_FILE"
	WorldCLIConfigFilename        = "world.toml"

	flagForConfigFile = "config"
)

var (
	// Items under these toml headers will be included in the environment variables when
	// running docker. An error will be generated if a duplicate key is found across
	// these sections.
	dockerEnvHeaders = []string{"cardinal", "evm", "nakama", "common"}

	configFile string // Config file flag value
)

type Config struct {
	RootDir   string
	GameDir   string
	Detach    bool
	Build     bool
	Debug     bool
	DevDA     bool
	Telemetry bool
	Timeout   int
	DockerEnv map[string]string
}

func AddConfigFlag(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&configFile, flagForConfigFile, "c", "", "A TOML config file")
}

func GetConfig() (*Config, error) {
	cfg, err := findAndLoadConfigFile()
	if err != nil {
		return nil, err
	}

	// Set any default values.
	cfg.Build = true
	return cfg, nil
}

// findAndLoadConfigFile searches for a config file based on the following priorities:
// 1. A config file set via a flag
// 2. A config file set via an environment variable
// 3. A config file named "world.toml" in the current directory
// 4. A config file found in a parent directory.
func findAndLoadConfigFile() (*Config, error) {
	// First look for the config file in the config file flag.
	if configFile != "" {
		return loadConfigFromFile(configFile)
	}

	// Next check the environment variable for a config flag.
	if filename := os.Getenv(WorldCLIConfigFileEnvVariable); filename != "" {
		return loadConfigFromFile(filename)
	}

	// Next, check the current directory, followed by all the parent directories for a "world.toml" file.
	currDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	for {
		filename := path.Join(currDir, WorldCLIConfigFilename)
		if cfg, err := loadConfigFromFile(filename); err == nil {
			return cfg, nil
		} else if !os.IsNotExist(err) {
			return nil, err
		}
		before := currDir
		currDir = path.Join(currDir, "..")
		if currDir == before {
			// We can't move up any more directories, so no config file can be found.
			break
		}
	}

	return nil, eris.New("No config file found")
}

//nolint:cyclop // slightly over limit, but not too complex
func loadConfigFromFile(filename string) (*Config, error) {
	cfg := Config{
		DockerEnv: map[string]string{},
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data := map[string]any{}
	if err = toml.NewDecoder(file).Decode(&data); err != nil {
		return nil, err
	}
	if rootDir, ok := data["root_dir"]; ok {
		cfg.RootDir, ok = rootDir.(string)
		if !ok {
			return nil, eris.New("root_dir must be a string")
		}
	} else {
		cfg.RootDir, _ = filepath.Split(filename)
	}

	// Load the game directory.
	if gameDir, ok := data["game_dir"]; ok {
		cfg.GameDir, ok = gameDir.(string)
		if !ok {
			return nil, eris.New("game_dir must be a string")
		}
	} else {
		cfg.GameDir = "cardinal"
	}

	for _, header := range dockerEnvHeaders {
		m, ok := data[header]
		if !ok {
			continue
		}
		for key, val := range m.(map[string]any) {
			if _, okay := cfg.DockerEnv[key]; okay {
				return nil, eris.Errorf("duplicate env variable %q", key)
			}
			cfg.DockerEnv[key] = fmt.Sprintf("%v", val)
		}
	}

	logger.Debugf("successfully loaded config from %q", filename)

	return &cfg, nil
}
