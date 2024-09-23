package main

import (
	"os"

	"github.com/ren3gadem4rm0t/imperva-export-cli/internal/cmd"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	if err := cmd.Execute(); err != nil {
		if zerolog.GlobalLevel() == zerolog.Disabled {
			_, _ = os.Stderr.WriteString(err.Error() + "\n")
		} else {
			log.Error().Err(err).Msg("Error executing command")
		}
		os.Exit(1)
	}
}
