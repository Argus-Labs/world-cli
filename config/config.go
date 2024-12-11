package config

import (
	"os"
	"path/filepath"

	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	logger "pkg.world.dev/world-cli/logging"
)

// Config represents the configuration for the World CLI
type Config struct {
	RootDir   string            `mapstructure:"root_dir"`
	GameDir   string            `mapstructure:"game_dir"`
	Build     bool              `mapstructure:"build"`
	Debug     bool              `mapstructure:"debug"`
	Detach    bool              `mapstructure:"detach"`
	Timeout   int               `mapstructure:"timeout"`
	Telemetry bool              `mapstructure:"telemetry"`
	DevDA     bool              `mapstructure:"dev_da"`
	DockerEnv map[string]string `mapstructure:"docker_env"`
}

// GetConfig returns the configuration from the config file
func GetConfig() (*Config, error) {
	if err := initConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, eris.Wrap(err, "failed to unmarshal config")
	}

	return &cfg, nil
}

// AddConfigFlag adds the config flag to the given command
func AddConfigFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().String("config", "", "config file (default is $HOME/.world/config.toml)")
}

// SetupConfigDir creates the config directory if it doesn't exist
func SetupConfigDir() error {
	configDir := filepath.Join(os.Getenv("HOME"), ".world")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return eris.Wrap(err, "failed to create config directory")
	}
	return nil
}

// initConfig reads in config file and ENV variables if set
func initConfig() error {
	if err := SetupConfigDir(); err != nil {
		return err
	}

	configFile := viper.GetString("config")
	if configFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(configFile)
	} else {
		// Search config in home directory
		home := os.Getenv("HOME")
		if home == "" {
			return eris.New("$HOME not set")
		}

		viper.AddConfigPath(filepath.Join(home, ".world"))
		viper.SetConfigType("toml")
		viper.SetConfigName("config")
	}

	// Read the config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return eris.Wrap(err, "failed to read config file")
		}
		logger.Warn("No config file found, using defaults")
	}

	return nil
}
