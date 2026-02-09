package api

//go:generate go run github.com/Khan/genqlient genqlient.yaml

import (
	"net/http"
	"time"

	"github.com/Khan/genqlient/graphql"
)

// LinearAPIEndpoint is the default Linear GraphQL API endpoint.
const LinearAPIEndpoint = "https://api.linear.app/graphql"

// authTransport is an http.RoundTripper that injects an Authorization header.
type authTransport struct {
	apiKey    string
	wrapped  http.RoundTripper
}

// RoundTrip implements http.RoundTripper. It clones the request before modifying
// headers, as required by the RoundTripper contract.
func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", t.apiKey)
	return t.wrapped.RoundTrip(req)
}

// NewClient creates a new authenticated GraphQL client for the Linear API.
// If endpoint is empty, LinearAPIEndpoint is used.
func NewClient(apiKey string, endpoint string) graphql.Client {
	if endpoint == "" {
		endpoint = LinearAPIEndpoint
	}
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &authTransport{
			apiKey:  apiKey,
			wrapped: http.DefaultTransport,
		},
	}
	return graphql.NewClient(endpoint, httpClient)
}

// NewClientWithHTTPClient creates a new GraphQL client using the provided
// http.Client. This is useful for testing where a custom transport is needed.
// If endpoint is empty, LinearAPIEndpoint is used.
func NewClientWithHTTPClient(httpClient *http.Client, endpoint string) graphql.Client {
	if endpoint == "" {
		endpoint = LinearAPIEndpoint
	}
	return graphql.NewClient(endpoint, httpClient)
}
