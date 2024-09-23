package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var autoCmd = &cobra.Command{
	Use:   "auto",
	Short: "Initiate the export process and download the exported zip file after successful export",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		caid, _ := cmd.Flags().GetInt64("caid")
		if err := ValidateCAID(caid); err != nil {
			return err
		}

		handler, err := initiateAuto(caid)
		if err != nil {
			return fmt.Errorf("error during auto export: %w", err)
		}
		log.Info().Msgf("Export completed successfully. Handler ID: %s", handler)
		if zerolog.GlobalLevel() == zerolog.Disabled {
			fmt.Printf("Export completed successfully. Handler ID: %s\n", handler)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(autoCmd)
	autoCmd.Flags().Int64("caid", 0, "The account ID to work on")
	if err := autoCmd.MarkFlagRequired("caid"); err != nil {
		log.Error().Err(err).Msg("Failed to mark flag as required")
	}
}

func initiateAuto(caid int64) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	log.Info().Msgf("Initiating export for CAID: %d", caid)

	handler, err := initiateExport(ctx, caid)
	if err != nil {
		log.Error().Err(err).Msg("Failed to initiate export")
		return "", err
	}

	log.Info().Msgf("Export initiated. Handler ID: %s", handler)
	if zerolog.GlobalLevel() == zerolog.Disabled {
		fmt.Printf("Export initiated. Handler ID: %s\n", handler)
	}

	err = checkExportStatusWithContext(ctx, caid, handler)
	if err != nil {
		log.Error().Err(err).Msg("Error during status check")
		return "", err
	}

	log.Debug().Msg("Export completed successfully")
	return handler, nil
}
