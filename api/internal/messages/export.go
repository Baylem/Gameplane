package messages

import (
	"encoding/json"
	"fmt"
	"io"
)

// ExportJSON writes the registry as a JSON array sorted by id, with
// two-space indentation and a trailing newline. The i18n tooling reads this
// output to sync the server.* entries of the English reference pack.
func ExportJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(All()); err != nil {
		return fmt.Errorf("messages: export json: %w", err)
	}
	return nil
}
