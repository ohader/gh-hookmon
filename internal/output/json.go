package output

import (
	"encoding/json"
	"io"

	"github.com/ohader/gh-hookmon/internal/github"
)

// FormatJSON outputs deliveries in JSON format
func FormatJSON(deliveries []github.Delivery, w io.Writer) error {
	// Transform deliveries for display
	displayDeliveries := make([]github.Delivery, len(deliveries))
	for i, d := range deliveries {
		displayDeliveries[i] = d
		// Handle status code 0 specially
		if d.StatusCode == 0 && d.Status == "" {
			displayDeliveries[i].Status = "delivery failed"
		}
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(displayDeliveries)
}
