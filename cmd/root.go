package cmd

import (
	"fmt"
	"os"

	"github.com/ohader/gh-hookmon/internal/config"
	"github.com/ohader/gh-hookmon/internal/filter"
	"github.com/ohader/gh-hookmon/internal/github"
	"github.com/ohader/gh-hookmon/internal/output"
	"github.com/spf13/cobra"
)

var cfg config.Config

var rootCmd = &cobra.Command{
	Use:   "gh-hookmon",
	Short: "Monitor GitHub webhook deliveries",
	Long: `Retrieve and display webhook delivery history from GitHub organizations or repositories.

Examples:
  # List all webhook deliveries for an organization
  gh hookmon --org=myorg

  # List webhook deliveries for a specific repository
  gh hookmon --repo=owner/repo

  # Filter by URL pattern
  gh hookmon --org=myorg --filter="slack.com"

  # Filter by date range
  gh hookmon --org=myorg --since=2026-01-01 --until=2026-01-31

  # Show only failed deliveries
  gh hookmon --org=myorg --failed

  # Show only the 5 most recent deliveries per repository
  gh hookmon --org=myorg --head=5

  # Combine filters: failed deliveries from last week, top 3 per repo
  gh hookmon --org=myorg --failed --since=2026-01-13 --head=3

  # Sort by repository name alphabetically
  gh hookmon --org=myorg --sort=repository

  # Sort by status code ascending (success codes first)
  gh hookmon --org=myorg --sort=code:asc

  # Sort by event type descending
  gh hookmon --org=myorg --sort=event:desc

  # Combine with filters and sorting
  gh hookmon --org=myorg --failed --sort=repository:asc --head=5

  # Output as JSON
  gh hookmon --repo=owner/repo --json`,
	RunE: run,
}

func init() {
	rootCmd.Flags().StringVar(&cfg.Org, "org", "", "Process all repos in organization (required if --repo not set)")
	rootCmd.Flags().StringVar(&cfg.Repo, "repo", "", "Process specific repository OWNER/REPO (required if --org not set)")
	rootCmd.Flags().StringVar(&cfg.Filter, "filter", "", "Filter webhook URLs by pattern")
	rootCmd.Flags().String("since", "", "Start date YYYY-MM-DD (00:00:00)")
	rootCmd.Flags().String("until", "", "End date YYYY-MM-DD (23:59:59)")
	rootCmd.Flags().BoolVar(&cfg.JSONOutput, "json", false, "Output in JSON format")
	rootCmd.Flags().BoolVar(&cfg.Failed, "failed", false, "Filter for failed webhook deliveries (4xx, 5xx, or no response)")
	rootCmd.Flags().IntVar(&cfg.Head, "head", 0, "Show only N most recent deliveries per repository (default: all)")
	rootCmd.Flags().StringVar(&cfg.SortBy, "sort", "", "Sort by field (repository, timestamp, code, event) with optional order (:asc or :desc)")
}

func Execute() error {
	return rootCmd.Execute()
}

func run(cmd *cobra.Command, args []string) error {
	// Parse date range
	sinceStr, _ := cmd.Flags().GetString("since")
	untilStr, _ := cmd.Flags().GetString("until")

	since, until, err := config.ParseDateRange(sinceStr, untilStr)
	if err != nil {
		return err
	}

	cfg.Since = since
	cfg.Until = until

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	// Create GitHub client
	client, err := github.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w\nHint: Run 'gh auth login' to authenticate", err)
	}

	var allDeliveries []github.Delivery

	// Process organization or repository
	if cfg.Org != "" {
		allDeliveries, err = processOrganization(client, cfg.Org)
		if err != nil {
			return err
		}
	} else {
		allDeliveries, err = processRepository(client, cfg.Repo)
		if err != nil {
			return err
		}
	}

	// Apply date range filter
	filteredDeliveries := make([]github.Delivery, 0)
	for _, d := range allDeliveries {
		if filter.InRange(d.DeliveredAt, cfg.Since, cfg.Until) {
			filteredDeliveries = append(filteredDeliveries, d)
		}
	}

	// Apply status filter if --failed is specified
	if cfg.Failed {
		statusFilteredDeliveries := make([]github.Delivery, 0)
		for _, d := range filteredDeliveries {
			if filter.IsFailed(d.StatusCode) {
				statusFilteredDeliveries = append(statusFilteredDeliveries, d)
			}
		}
		filteredDeliveries = statusFilteredDeliveries
	}

	// If URL filter is specified, fetch detailed delivery info and filter
	if cfg.Filter != "" {
		detailedDeliveries, err := fetchDeliveryDetails(client, filteredDeliveries, cfg.Org != "")
		if err != nil {
			return err
		}

		// Filter by URL pattern
		finalDeliveries := make([]github.Delivery, 0)
		for _, d := range detailedDeliveries {
			if filter.MatchesPattern(d.URL, cfg.Filter) {
				finalDeliveries = append(finalDeliveries, d)
			}
		}
		filteredDeliveries = finalDeliveries
	}

	// Apply sorting based on configuration
	sortField, ascending := cfg.GetSortConfig()
	github.ApplySort(filteredDeliveries, sortField, ascending)

	// Apply per-repository head limit if specified
	if cfg.Head > 0 {
		sortField, ascending := cfg.GetSortConfig()
		filteredDeliveries = applyHeadLimit(filteredDeliveries, cfg.Head, sortField, ascending)
	}

	// Output results
	if cfg.JSONOutput {
		return output.FormatJSON(filteredDeliveries, os.Stdout)
	} else {
		output.FormatTable(filteredDeliveries, os.Stdout)
		return nil
	}
}

func processOrganization(client *github.Client, org string) ([]github.Delivery, error) {
	fmt.Fprintf(os.Stderr, "Fetching repositories for organization: %s\n", org)

	// Get all repositories in the organization
	repos, err := client.ListOrgRepos(org)
	if err != nil {
		return nil, fmt.Errorf("failed to list organization repositories: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Found %d repositories\n", len(repos))

	if len(repos) == 0 {
		return []github.Delivery{}, nil
	}

	// Use concurrent workers to speed up repository processing
	const maxConcurrent = 10
	numWorkers := maxConcurrent
	if len(repos) < numWorkers {
		numWorkers = len(repos)
	}

	// Channels for work distribution and results
	type repoResult struct {
		repo       string
		deliveries []github.Delivery
		err        error
	}

	jobs := make(chan string, len(repos))
	results := make(chan repoResult, len(repos))

	// Start workers
	for w := 0; w < numWorkers; w++ {
		go func() {
			for repo := range jobs {
				fmt.Fprintf(os.Stderr, "Processing repository: %s\n", repo)
				repoDeliveries, err := processRepository(client, repo)
				results <- repoResult{
					repo:       repo,
					deliveries: repoDeliveries,
					err:        err,
				}
			}
		}()
	}

	// Send jobs
	for _, repo := range repos {
		jobs <- repo
	}
	close(jobs)

	// Collect results
	var allDeliveries []github.Delivery
	for i := 0; i < len(repos); i++ {
		result := <-results
		if result.err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to process repository %s: %v\n", result.repo, result.err)
			continue
		}
		allDeliveries = append(allDeliveries, result.deliveries...)
	}

	return allDeliveries, nil
}

func processRepository(client *github.Client, repo string) ([]github.Delivery, error) {
	// Get webhooks for the repository
	hooks, err := client.ListRepoWebhooks(repo)
	if err != nil {
		return nil, fmt.Errorf("failed to list webhooks: %w", err)
	}

	if len(hooks) == 0 {
		return []github.Delivery{}, nil
	}

	var allDeliveries []github.Delivery

	// For each webhook, get deliveries
	for _, hook := range hooks {
		// If we have a URL filter, check if this hook matches before fetching deliveries
		if cfg.Filter != "" && !hook.MatchesFilter(cfg.Filter) {
			continue
		}

		deliveries, err := client.ListRepoHookDeliveries(repo, hook.ID, 100)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to list deliveries for hook %d: %v\n", hook.ID, err)
			continue
		}

		// Add the webhook target URL to each delivery
		targetURL := hook.GetTargetURL()
		for i := range deliveries {
			deliveries[i].URL = targetURL
		}

		allDeliveries = append(allDeliveries, deliveries...)
	}

	return allDeliveries, nil
}

func fetchDeliveryDetails(client *github.Client, deliveries []github.Delivery, isOrg bool) ([]github.Delivery, error) {
	if len(deliveries) == 0 {
		return deliveries, nil
	}

	// Use concurrent workers to speed up fetching
	const maxConcurrent = 5
	numWorkers := maxConcurrent
	if len(deliveries) < numWorkers {
		numWorkers = len(deliveries)
	}

	// Channels for work distribution and results
	jobs := make(chan github.Delivery, len(deliveries))
	results := make(chan github.Delivery, len(deliveries))
	errors := make(chan error, len(deliveries))

	// Start workers
	for w := 0; w < numWorkers; w++ {
		go func() {
			for d := range jobs {
				// Always use repository webhook endpoint since all webhooks are repository webhooks
				// Even when processing an org, we iterate through repos and fetch their webhooks
				detail, err := client.GetRepoHookDeliveryDetail(d.Repository, d.HookID, d.ID)

				if err != nil {
					errors <- fmt.Errorf("failed to get delivery detail for %d: %v", d.ID, err)
					continue
				}

				// Copy basic delivery info and add URL
				detailed := d
				detailed.URL = detail.URL
				results <- detailed
			}
		}()
	}

	// Send jobs
	for _, d := range deliveries {
		jobs <- d
	}
	close(jobs)

	// Collect results
	detailedDeliveries := make([]github.Delivery, 0, len(deliveries))
	for i := 0; i < len(deliveries); i++ {
		select {
		case detailed := <-results:
			detailedDeliveries = append(detailedDeliveries, detailed)
		case err := <-errors:
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
	}

	return detailedDeliveries, nil
}

// applyHeadLimit limits the results to the N most recent deliveries per repository
// Assumes deliveries are already sorted by the configured sort field and direction
func applyHeadLimit(deliveries []github.Delivery, limit int, sortField string, ascending bool) []github.Delivery {
	if limit <= 0 {
		return deliveries
	}

	// Group deliveries by repository
	repoGroups := make(map[string][]github.Delivery)
	for _, d := range deliveries {
		repoGroups[d.Repository] = append(repoGroups[d.Repository], d)
	}

	// Take only the first N from each repository (already sorted)
	result := make([]github.Delivery, 0)
	for _, group := range repoGroups {
		count := limit
		if count > len(group) {
			count = len(group)
		}
		result = append(result, group[:count]...)
	}

	// Re-sort the combined results to maintain global sort order
	github.ApplySort(result, sortField, ascending)

	return result
}
