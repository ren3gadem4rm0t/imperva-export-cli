package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "imperva-export-cli",
	Short: "A CLI tool for interacting with the Imperva Account-Export API",
	Long: `Imperva Export CLI is a command-line tool for exporting account configuration settings
from the Imperva platform to a zip file in standard Terraform format.

For more information, visit the documentation at:
https://docs.imperva.com/bundle/cloud-application-security/page/account-export.htm
`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initConfig()
	},
}

func Execute() error {
	rootCmd.Version = version
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/imperva-export-cli.yaml)")
	rootCmd.PersistentFlags().String("api-id", "", "API ID - prefer to use environment variable API_ID")
	rootCmd.PersistentFlags().String("api-key", "", "API Key - prefer to use environment variable API_KEY")
	rootCmd.PersistentFlags().String("log-level", "none", "Set the logging level (none, debug, info, warn, error)")
	rootCmd.PersistentFlags().String("output-dir", ".", "Directory to save exported files")

	if err := viper.BindPFlag("api-id", rootCmd.PersistentFlags().Lookup("api-id")); err != nil {
		log.Error().Err(err).Msg("Failed to bind flag api-id")
	}
	if err := viper.BindPFlag("api-key", rootCmd.PersistentFlags().Lookup("api-key")); err != nil {
		log.Error().Err(err).Msg("Failed to bind flag api-key")
	}
	if err := viper.BindPFlag("log-level", rootCmd.PersistentFlags().Lookup("log-level")); err != nil {
		log.Error().Err(err).Msg("Failed to bind flag log-level")
	}
	if err := viper.BindPFlag("output-dir", rootCmd.PersistentFlags().Lookup("output-dir")); err != nil {
		log.Error().Err(err).Msg("Failed to bind flag output-dir")
	}

	if err := viper.BindEnv("api-id", "API_ID"); err != nil {
		log.Error().Err(err).Msg("Failed to bind environment variable API_ID")
	}
	if err := viper.BindEnv("api-key", "API_KEY"); err != nil {
		log.Error().Err(err).Msg("Failed to bind environment variable API_KEY")
	}
	if err := viper.BindEnv("output-dir", "OUTPUT_DIR"); err != nil {
		log.Error().Err(err).Msg("Failed to bind environment variable OUTPUT_DIR")
	}

	rootCmd.SetVersionTemplate(fmt.Sprintf("imperva-export-cli version %s\n", version))
}

func initConfig() error {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("could not find home directory: %w", err)
		}
		configPath := filepath.Join(home, ".config")
		viper.AddConfigPath(configPath)
		viper.SetConfigName("imperva-export-cli")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("error reading config file: %w", err)
		}
	}

	logLevel := viper.GetString("log-level")
	setLogLevel(logLevel)

	if viper.GetString("api-id") == "" || viper.GetString("api-key") == "" {
		return fmt.Errorf("API ID and API Key must be provided via flags, config file, or environment variables")
	}

	if err := validateConfig(); err != nil {
		return err
	}

	return nil
}

func setLogLevel(level string) {
	zerolog.TimeFieldFormat = time.RFC3339

	switch level {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msg("Debug level logging enabled")
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "none":
		zerolog.SetGlobalLevel(zerolog.Disabled)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		log.Warn().Msgf("Unknown log level '%s', defaulting to 'info'", level)
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339,
		NoColor:    false,
	})
}

func validateConfig() error {
	if viper.GetString("api-id") == "" {
		return fmt.Errorf("API ID must be provided via flag, config file, or environment variable")
	}
	if viper.GetString("api-key") == "" {
		return fmt.Errorf("API Key must be provided via flag, config file, or environment variable")
	}
	return nil
}
