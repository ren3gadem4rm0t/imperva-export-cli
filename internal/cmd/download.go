package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download the exported zip file after successful export",
	Long: `Download the exported zip file after a successful export process.
The download command retrieves the export file using the provided handler and CAID.`,
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

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		if err := downloadExportFile(ctx, caid, handler); err != nil {
			return fmt.Errorf("error downloading export file: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(downloadCmd)
	downloadCmd.Flags().String("handler", "", "The handler received in the export response")
	downloadCmd.Flags().Int64("caid", 0, "The account ID to work on")
	err := downloadCmd.MarkFlagRequired("handler")
	if err != nil {
		log.Error().Err(err).Msg("Failed to mark flag as required")
	}
	err = downloadCmd.MarkFlagRequired("caid")
	if err != nil {
		log.Error().Err(err).Msg("Failed to mark flag as required")
	}
}

func downloadExportFile(ctx context.Context, caid int64, handler string) error {
	if err := ValidateHandler(handler); err != nil {
		return err
	}

	outputDir := viper.GetString("output-dir")
	if outputDir == "" {
		outputDir = "."
	}
	if err := ValidateOutputDir(outputDir); err != nil {
		return err
	}

	url := fmt.Sprintf("%s/v3/export/download/%s?caid=%d", apiBaseURL, handler, caid)

	resp, err := makeAPIRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return HandleHTTPError(resp)
	}

	if err := SaveExportFile(caid, handler, resp); err != nil {
		return fmt.Errorf("failed to save export file: %w", err)
	}
	return nil
}

func SaveExportFile(caid int64, handler string, resp *http.Response) error {
	filename := fmt.Sprintf("export_%d_%s.zip", caid, handler)
	outputDir := viper.GetString("output-dir")
	if outputDir == "" {
		outputDir = "."
	}
	filePath := filepath.Join(outputDir, filename)
	if err := ValidateOutputDir(outputDir); err != nil {
		return fmt.Errorf("invalid output dir: %w", err)
	}
	tempFilePath := filePath + ".tmp"

	if err := ValidateFilePath(tempFilePath); err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	if err := os.MkdirAll(outputDir, 0750); err != nil {
		log.Error().Err(err).Msgf("Failed to create output directory: %s", outputDir)
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	outFile, err := os.OpenFile(tempFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600) // #nosec G304 -- Path validated above
	if err != nil {
		log.Error().Err(err).Msgf("Failed to create temp file: %s", tempFilePath)
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		if err := outFile.Close(); err != nil {
			log.Error().Err(err).Msgf("Failed to close temp file: %s", tempFilePath)
		}
		if err := os.Remove(tempFilePath); err != nil {
			if !os.IsNotExist(err) {
				log.Error().Err(err).Msgf("Failed to remove temp file: %s", tempFilePath)
			}
		}
	}()

	buffer := make([]byte, 32*1024)
	var totalBytes int64 = 0
	for {
		n, readErr := resp.Body.Read(buffer)
		if n > 0 {
			if _, writeErr := outFile.Write(buffer[:n]); writeErr != nil {
				log.Error().Err(writeErr).Msgf("Failed to write to temp file: %s", tempFilePath)
				return fmt.Errorf("failed to write to temp file: %w", writeErr)
			}
			totalBytes += int64(n)
		}
		if readErr != nil {
			if readErr != io.EOF {
				log.Error().Err(readErr).Msg("Error reading response body")
				return fmt.Errorf("error reading response body: %w", readErr)
			}
			break
		}
	}

	if err := os.Rename(tempFilePath, filePath); err != nil {
		log.Error().Err(err).Msgf("Failed to rename temp file to final file: %s", filePath)
		return fmt.Errorf("failed to rename temp file to final file: %w", err)
	}

	log.Info().Msgf("Export file downloaded successfully to %s (%d bytes)", filePath, totalBytes)
	if zerolog.GlobalLevel() == zerolog.Disabled {
		fmt.Printf("Export file downloaded successfully to %s (%d bytes)\n", filePath, totalBytes)
	}
	return nil
}
