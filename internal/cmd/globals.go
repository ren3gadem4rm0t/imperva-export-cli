package cmd

import (
	"fmt"
)

const (
	apiIDHeaderName  string = "x-API-Id"  // #nosec G101 -- False positive. This is the name of the header, not the value.
	apiKeyHeaderName string = "x-API-Key" // #nosec G101 -- False positive. This is the name of the header, not the value.
	version          string = "1.0.0"
)

var (
	apiBaseURL     string = "https://api.imperva.com/account-export-import"
	userAgentValue string = fmt.Sprintf("imperva-export-cli/%s", version)
	cfgFile        string
)
