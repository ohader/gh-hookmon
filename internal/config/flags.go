package config

import (
	"fmt"
	"strings"
	"time"
)

// Config holds the application configuration
type Config struct {
	Org        string
	Repo       string
	Filter     string
	Since      *time.Time
	Until      *time.Time
	JSONOutput bool
	Failed     bool   // Filter for failed deliveries only
	Head       int    // Limit to N most recent deliveries per repo (0 = no limit)
	SortBy     string // Sort field and order: "field:order" (e.g., "repository:asc", "timestamp:desc")
}

// Validate checks that the configuration is valid
func (c *Config) Validate() error {
	// Exactly one of --org or --repo must be set
	if c.Org == "" && c.Repo == "" {
		return fmt.Errorf("either --org or --repo must be specified")
	}
	if c.Org != "" && c.Repo != "" {
		return fmt.Errorf("cannot specify both --org and --repo")
	}

	// If --repo, validate OWNER/REPO format
	if c.Repo != "" {
		parts := strings.Split(c.Repo, "/")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return fmt.Errorf("--repo must be in format OWNER/REPO")
		}
	}

	// Validate date range
	if c.Since != nil && c.Until != nil {
		if c.Since.After(*c.Until) {
			return fmt.Errorf("--since must be before --until")
		}
	}

	// Validate head flag
	if c.Head < 0 {
		return fmt.Errorf("--head must be a non-negative integer")
	}

	// Validate sort flag
	if c.SortBy != "" {
		parts := strings.Split(c.SortBy, ":")
		if len(parts) > 2 {
			return fmt.Errorf("--sort format should be 'field' or 'field:order'")
		}

		// Validate field
		field := parts[0]
		validFields := map[string]bool{
			"repository": true,
			"timestamp":  true,
			"code":       true,
			"event":      true,
		}
		if !validFields[field] {
			return fmt.Errorf("--sort field must be one of: repository, timestamp, code, event")
		}

		// Validate order if specified
		if len(parts) == 2 {
			order := parts[1]
			if order != "asc" && order != "desc" {
				return fmt.Errorf("--sort order must be 'asc' or 'desc'")
			}
		}
	}

	return nil
}

// ParseDateRange parses the since and until date strings
func ParseDateRange(sinceStr, untilStr string) (*time.Time, *time.Time, error) {
	var since, until *time.Time

	if sinceStr != "" {
		t, err := time.Parse("2006-01-02", sinceStr)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid --since format (expected YYYY-MM-DD): %w", err)
		}
		// Set to 00:00:00 UTC
		t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		since = &t
	}

	if untilStr != "" {
		t, err := time.Parse("2006-01-02", untilStr)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid --until format (expected YYYY-MM-DD): %w", err)
		}
		// Set to 23:59:59 UTC
		t = time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, time.UTC)
		until = &t
	}

	return since, until, nil
}

// GetSortConfig returns the sort field and whether it should be ascending
// Returns field name, ascending bool, and defaults based on field type
func (c *Config) GetSortConfig() (field string, ascending bool) {
	if c.SortBy == "" {
		return "timestamp", false // Default: timestamp descending
	}

	parts := strings.Split(c.SortBy, ":")
	field = parts[0]

	// If order explicitly specified, use it
	if len(parts) == 2 {
		ascending = parts[1] == "asc"
		return field, ascending
	}

	// Use field-specific defaults
	switch field {
	case "repository", "event":
		return field, true // Alphabetical fields default to ascending
	case "timestamp", "code":
		return field, false // Numeric/time fields default to descending
	default:
		return "timestamp", false
	}
}
