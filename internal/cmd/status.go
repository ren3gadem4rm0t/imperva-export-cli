package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check the status of an export process",
	Long: `Check the status of an export process using the provided handler and CAID.
This command polls the API to determine if the export has completed and downloads the file once ready.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		handler, _ := cmd.Flags().GetString("handler")
		if err := ValidateHandler(handler); err != nil {
			return err
		}

		caid, _ := cmd.Flags().GetInt64("caid")
		if err := ValidateCAID(caid); err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		if err := checkExportStatusWithContext(ctx, caid, handler); err != nil {
			return fmt.Errorf("error checking export status: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().String("handler", "", "The handler received in the export response")
	statusCmd.Flags().Int64("caid", 0, "The account ID to work on")
	if err := statusCmd.MarkFlagRequired("handler"); err != nil {
		log.Error().Err(err).Msg("Failed to mark flag as required")
	}
	if err := statusCmd.MarkFlagRequired("caid"); err != nil {
		log.Error().Err(err).Msg("Failed to mark flag as required")
	}
}

func checkExportStatusWithContext(ctx context.Context, caid int64, handler string) error {
	url := fmt.Sprintf("%s/v3/export/download/%s?caid=%d", apiBaseURL, handler, caid)

	initialDelay := 1 * time.Second
	maxDelay := 30 * time.Second
	currentDelay := initialDelay
	attempts := 0
	maxAttempts := 60

	log.Info().Msg("Waiting for export to complete...")
	if zerolog.GlobalLevel() == zerolog.Disabled {
		fmt.Print("Waiting for export to complete")
	}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out while waiting for export to complete")
		default:
			log.Debug().Msgf("Checking export status at URL: %s", url)
			resp, err := makeAPIRequest(ctx, http.MethodGet, url, nil)
			if err != nil {
				return err
			}

			switch resp.StatusCode {
			case http.StatusOK:
				if zerolog.GlobalLevel() == zerolog.Disabled {
					fmt.Println("\nExport completed. Saving file...")
				}
				err := SaveExportFile(caid, handler, resp)
				closeErr := resp.Body.Close()
				if err != nil {
					return fmt.Errorf("failed to save export file: %w", err)
				}
				if closeErr != nil {
					return fmt.Errorf("failed to close response body: %w", closeErr)
				}
				return nil
			case http.StatusAccepted:
				if zerolog.GlobalLevel() == zerolog.Disabled {
					fmt.Print(".")
				}
				log.Info().Msg("Export still in progress...")
				closeErr := resp.Body.Close()
				if closeErr != nil {
					return fmt.Errorf("failed to close response body: %w", closeErr)
				}
			default:
				body, readErr := io.ReadAll(resp.Body)
				closeErr := resp.Body.Close()
				if readErr != nil {
					return fmt.Errorf("failed to read response body: %w", readErr)
				}
				if closeErr != nil {
					return fmt.Errorf("failed to close response body: %w", closeErr)
				}
				apiErr := HandleHTTPError(resp)
				if apiErr != nil {
					return fmt.Errorf("API error: %w", apiErr)
				}
				return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
			}

			select {
			case <-ctx.Done():
				return fmt.Errorf("timed out while waiting for export to complete")
			case <-time.After(currentDelay):
				if currentDelay < maxDelay {
					currentDelay *= 2
					if currentDelay > maxDelay {
						currentDelay = maxDelay
					}
				}
				attempts++
				if attempts >= maxAttempts {
					return fmt.Errorf("maximum number of attempts (%d) reached while waiting for export to complete", maxAttempts)
				}
			}
		}
	}
}
