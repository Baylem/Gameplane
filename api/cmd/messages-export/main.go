// Command messages-export writes the API message registry as JSON to stdout.
// The i18n-sync Makefile target feeds the output to i18n-check.
package main

import (
	"fmt"
	"os"

	"github.com/GameplanePanel/gameplane/api/internal/messages"
)

func main() {
	if err := messages.ExportJSON(os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
