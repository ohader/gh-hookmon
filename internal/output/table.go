package output

import (
	"fmt"
	"io"
	"time"

	"github.com/ohader/gh-hookmon/internal/github"
	"github.com/olekukonko/tablewriter"
)

// FormatTable outputs deliveries as an ASCII table
func FormatTable(deliveries []github.Delivery, w io.Writer) {
	if len(deliveries) == 0 {
		fmt.Fprintln(w, "No webhook deliveries found")
		return
	}

	table := tablewriter.NewTable(w,
		tablewriter.WithHeader([]string{
			"Delivery ID",
			"Repository",
			"Hook ID",
			"Timestamp",
			"Status",
			"Code",
			"Event",
			"Action",
			"URL",
		}),
	)

	for _, d := range deliveries {
		// Color code status based on HTTP status code
		// Handle status code 0 specially
		status := d.Status
		if d.StatusCode == 0 {
			// Status code 0 means delivery failed (no response)
			status = "delivery failed"
			status = fmt.Sprintf("\033[31m%s\033[0m", status) // Red
		} else if d.Status == "" {
			// Fallback if status is empty but status code exists
			status = "-"
		} else if d.StatusCode >= 200 && d.StatusCode < 300 {
			status = fmt.Sprintf("\033[32m%s\033[0m", status) // Green
		} else if d.StatusCode >= 400 {
			status = fmt.Sprintf("\033[31m%s\033[0m", status) // Red
		} else if d.StatusCode >= 300 && d.StatusCode < 400 {
			status = fmt.Sprintf("\033[33m%s\033[0m", status) // Yellow
		}

		// Truncate long URLs for display
		urlDisplay := d.URL
		if urlDisplay == "" {
			urlDisplay = "-"
		} else if len(urlDisplay) > 50 {
			urlDisplay = urlDisplay[:47] + "..."
		}

		// Format timestamp
		timestamp := d.DeliveredAt.Format(time.RFC3339)

		// Format action (may be empty)
		action := d.Action
		if action == "" {
			action = "-"
		}

		table.Append([]string{
			fmt.Sprintf("%d", d.ID),
			d.Repository,
			fmt.Sprintf("%d", d.HookID),
			timestamp,
			status,
			fmt.Sprintf("%d", d.StatusCode),
			d.Event,
			action,
			urlDisplay,
		})
	}

	table.Render()
	table.Close()
}
