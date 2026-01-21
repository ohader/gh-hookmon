package github

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

// Delivery represents a webhook delivery
type Delivery struct {
	ID          int       `json:"id"`
	GUID        string    `json:"guid"`
	DeliveredAt time.Time `json:"delivered_at"`
	Redelivery  bool      `json:"redelivery"`
	Duration    float64   `json:"duration"`
	Status      string    `json:"status"`
	StatusCode  int       `json:"status_code"`
	Event       string    `json:"event"`
	Action      string    `json:"action"`
	URL         string    `json:"url,omitempty"` // Only available in detailed view
	Repository  string    `json:"-"`             // Added by us to track which repo
	HookID      int       `json:"-"`             // Added by us to track which hook
}

// DeliveryDetail represents a detailed webhook delivery with full information
type DeliveryDetail struct {
	Delivery
	Request struct {
		Headers map[string]string `json:"headers"`
		Payload interface{}       `json:"payload"`
	} `json:"request"`
	Response struct {
		Headers map[string]string `json:"headers"`
		Payload string            `json:"payload"`
	} `json:"response"`
}

// ListOrgHookDeliveries retrieves all deliveries for an organization hook
func (c *Client) ListOrgHookDeliveries(org string, hookID int, perPage int) ([]Delivery, error) {
	if perPage <= 0 {
		perPage = 100
	}

	var deliveries []Delivery
	path := fmt.Sprintf("orgs/%s/hooks/%d/deliveries?per_page=%d", org, hookID, perPage)

	response, err := c.rest.Request("GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list deliveries for org hook %d: %w", hookID, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if err := json.Unmarshal(body, &deliveries); err != nil {
		return nil, fmt.Errorf("failed to parse deliveries response: %w", err)
	}

	// Tag each delivery with the org and hook ID for reference
	for i := range deliveries {
		deliveries[i].Repository = org
		deliveries[i].HookID = hookID
	}

	return deliveries, nil
}

// ListRepoHookDeliveries retrieves all deliveries for a repository hook
func (c *Client) ListRepoHookDeliveries(repo string, hookID int, perPage int) ([]Delivery, error) {
	if perPage <= 0 {
		perPage = 100
	}

	var deliveries []Delivery
	path := fmt.Sprintf("repos/%s/hooks/%d/deliveries?per_page=%d", repo, hookID, perPage)

	response, err := c.rest.Request("GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list deliveries for repo hook %d: %w", hookID, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if err := json.Unmarshal(body, &deliveries); err != nil {
		return nil, fmt.Errorf("failed to parse deliveries response: %w", err)
	}

	// Tag each delivery with the repo and hook ID for reference
	for i := range deliveries {
		deliveries[i].Repository = repo
		deliveries[i].HookID = hookID
	}

	return deliveries, nil
}

// GetOrgHookDeliveryDetail retrieves detailed information for a specific delivery
func (c *Client) GetOrgHookDeliveryDetail(org string, hookID int, deliveryID int) (*DeliveryDetail, error) {
	var detail DeliveryDetail
	path := fmt.Sprintf("orgs/%s/hooks/%d/deliveries/%d", org, hookID, deliveryID)

	response, err := c.rest.Request("GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery detail: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if err := json.Unmarshal(body, &detail); err != nil {
		return nil, fmt.Errorf("failed to parse delivery detail: %w", err)
	}

	detail.Repository = org
	detail.HookID = hookID

	return &detail, nil
}

// GetRepoHookDeliveryDetail retrieves detailed information for a specific delivery
func (c *Client) GetRepoHookDeliveryDetail(repo string, hookID int, deliveryID int) (*DeliveryDetail, error) {
	var detail DeliveryDetail
	path := fmt.Sprintf("repos/%s/hooks/%d/deliveries/%d", repo, hookID, deliveryID)

	response, err := c.rest.Request("GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery detail: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if err := json.Unmarshal(body, &detail); err != nil {
		return nil, fmt.Errorf("failed to parse delivery detail: %w", err)
	}

	detail.Repository = repo
	detail.HookID = hookID

	return &detail, nil
}

// SortDeliveriesByTime sorts deliveries by timestamp
// ascending=true sorts oldest first, ascending=false sorts newest first
func SortDeliveriesByTime(deliveries []Delivery, ascending bool) {
	sort.Slice(deliveries, func(i, j int) bool {
		if ascending {
			return deliveries[i].DeliveredAt.Before(deliveries[j].DeliveredAt)
		}
		return deliveries[i].DeliveredAt.After(deliveries[j].DeliveredAt)
	})
}

// SortDeliveriesByRepository sorts deliveries alphabetically by repository name
func SortDeliveriesByRepository(deliveries []Delivery, ascending bool) {
	sort.Slice(deliveries, func(i, j int) bool {
		cmp := strings.Compare(deliveries[i].Repository, deliveries[j].Repository)
		if ascending {
			return cmp < 0
		}
		return cmp > 0
	})
}

// SortDeliveriesByStatusCode sorts deliveries numerically by HTTP status code
func SortDeliveriesByStatusCode(deliveries []Delivery, ascending bool) {
	sort.Slice(deliveries, func(i, j int) bool {
		if ascending {
			return deliveries[i].StatusCode < deliveries[j].StatusCode
		}
		return deliveries[i].StatusCode > deliveries[j].StatusCode
	})
}

// SortDeliveriesByEvent sorts deliveries alphabetically by event type
func SortDeliveriesByEvent(deliveries []Delivery, ascending bool) {
	sort.Slice(deliveries, func(i, j int) bool {
		cmp := strings.Compare(deliveries[i].Event, deliveries[j].Event)
		if ascending {
			return cmp < 0
		}
		return cmp > 0
	})
}

// ApplySort sorts deliveries based on the specified field and direction
func ApplySort(deliveries []Delivery, sortBy string, ascending bool) {
	switch sortBy {
	case "repository":
		SortDeliveriesByRepository(deliveries, ascending)
	case "code":
		SortDeliveriesByStatusCode(deliveries, ascending)
	case "event":
		SortDeliveriesByEvent(deliveries, ascending)
	case "timestamp":
		SortDeliveriesByTime(deliveries, ascending)
	default:
		// Default to timestamp descending
		SortDeliveriesByTime(deliveries, false)
	}
}
