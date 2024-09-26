// ./internal/cmd/export.go

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Initiate the export process",
	Long: `Initiate the export process for an account. This command starts the asynchronous
export operation and returns a handler ID to track the export status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		caid, _ := cmd.Flags().GetInt64("caid")
		if err := ValidateCAID(caid); err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		handler, err := initiateExport(ctx, caid)
		if err != nil {
			return fmt.Errorf("error initiating export: %w", err)
		}
		log.Info().Msgf("Export initiated. Handler: %s", handler)
		if zerolog.GlobalLevel() == zerolog.Disabled {
			fmt.Printf("Export initiated. Handler: %s\n", handler)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)
	exportCmd.Flags().Int64("caid", 0, "The account ID to work on")
	if err := exportCmd.MarkFlagRequired("caid"); err != nil {
		log.Error().Err(err).Msg("Failed to mark flag as required")
	}
}

// initiateExport starts the export process and returns the handler ID
func initiateExport(ctx context.Context, caid int64) (string, error) {
	url := fmt.Sprintf("%s/v3/export?caid=%d", apiBaseURL, caid)

	log.Debug().Msgf("Initiating export for CAID: %d", caid)
	log.Debug().Msgf("Export URL: %s", url)

	resp, err := makeAPIRequest(ctx, http.MethodPost, url, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to initiate export")
		return "", fmt.Errorf("failed to initiate export: %w", err)
	}
	defer resp.Body.Close()

	if err := HandleHTTPError(resp); err != nil {
		return "", fmt.Errorf("error response from export initiation: %w", err)
	}

	var asyncResp AsyncResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&asyncResp); err != nil {
		return "", fmt.Errorf("failed to decode export initiation response: %w", err)
	}

	if asyncResp.Handler == "" {
		return "", fmt.Errorf("received empty handler in response")
	}

	return asyncResp.Handler, nil
}
