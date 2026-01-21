package github

import (
	"github.com/cli/go-gh/v2/pkg/api"
)

// Client wraps the GitHub API client
type Client struct {
	rest *api.RESTClient
}

// NewClient creates a new GitHub API client
// Uses gh CLI's authentication automatically
func NewClient() (*Client, error) {
	rest, err := api.DefaultRESTClient()
	if err != nil {
		return nil, err
	}

	return &Client{
		rest: rest,
	}, nil
}
