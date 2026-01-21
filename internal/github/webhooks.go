package github

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Hook represents a GitHub webhook
type Hook struct {
	ID     int    `json:"id"`
	URL    string `json:"url"`
	Active bool   `json:"active"`
	Config struct {
		URL string `json:"url"`
	} `json:"config"`
}

// ListOrgWebhooks retrieves all webhooks for an organization
func (c *Client) ListOrgWebhooks(org string) ([]Hook, error) {
	var hooks []Hook
	err := c.rest.Get(fmt.Sprintf("orgs/%s/hooks", org), &hooks)
	if err != nil {
		return nil, fmt.Errorf("failed to list organization webhooks: %w", err)
	}
	return hooks, nil
}

// ListRepoWebhooks retrieves all webhooks for a repository
func (c *Client) ListRepoWebhooks(repo string) ([]Hook, error) {
	var hooks []Hook
	err := c.rest.Get(fmt.Sprintf("repos/%s/hooks", repo), &hooks)
	if err != nil {
		return nil, fmt.Errorf("failed to list repository webhooks: %w", err)
	}
	return hooks, nil
}

// ListOrgRepos retrieves all repositories for an organization
func (c *Client) ListOrgRepos(org string) ([]string, error) {
	type repo struct {
		FullName string `json:"full_name"`
	}

	var repos []repo
	page := 1
	perPage := 100

	for {
		var pageRepos []repo
		path := fmt.Sprintf("orgs/%s/repos?per_page=%d&page=%d", org, perPage, page)

		response, err := c.rest.Request("GET", path, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to list organization repositories: %w", err)
		}
		defer response.Body.Close()

		body, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		if err := json.Unmarshal(body, &pageRepos); err != nil {
			return nil, fmt.Errorf("failed to parse repositories response: %w", err)
		}

		if len(pageRepos) == 0 {
			break
		}

		repos = append(repos, pageRepos...)

		if len(pageRepos) < perPage {
			break
		}

		page++
	}

	names := make([]string, len(repos))
	for i, r := range repos {
		names[i] = r.FullName
	}

	return names, nil
}

// GetWebhookTargetURL extracts the target URL from a webhook
func (h *Hook) GetTargetURL() string {
	if h.Config.URL != "" {
		return h.Config.URL
	}
	// Fallback to extracting from the hook URL
	// Hook URL is like: https://api.github.com/repos/owner/repo/hooks/123
	// We want the actual webhook target URL from config
	return ""
}

// MatchesFilter checks if the webhook's target URL matches the filter pattern
func (h *Hook) MatchesFilter(pattern string) bool {
	if pattern == "" {
		return true
	}
	targetURL := h.GetTargetURL()
	return strings.Contains(strings.ToLower(targetURL), strings.ToLower(pattern))
}
