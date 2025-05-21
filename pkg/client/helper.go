package client

import (
	"fmt"
	"net/url"
	"strings"
)

// constructURL builds the full URL for an API request.
func (c *Client) constructURL(path string, pathParams map[string]string, queryParams map[string]string, pageNumber, pageSize *uint) (*url.URL, error) {
	// Start with the base URL
	u := *c.baseAPIURL

	// Add the path
	if path != "" {
		// Replace path parameters
		for k, v := range pathParams {
			path = strings.ReplaceAll(path, "{"+k+"}", url.PathEscape(v))
		}
		u.Path += path
	}

	// Add query parameters
	q := u.Query()
	q.Set("api-version", apiVersion)

	// Add pagination parameters if provided
	if pageNumber != nil {
		q.Set("skip", fmt.Sprintf("%d", *pageNumber*(*pageSize)))
	}
	if pageSize != nil {
		q.Set("top", fmt.Sprintf("%d", *pageSize))
	}

	// Add any additional query parameters
	for k, v := range queryParams {
		q.Set(k, v)
	}

	u.RawQuery = q.Encode()

	return &u, nil
}

// GetNextPageNumber calculates the next page number for pagination
// based on the current results and page size.
// Returns nil if there are no more pages.
func GetNextPageNumber(resultCount int, pageNumber uint) *uint {
	if resultCount < ResourcesPageSize {
		return nil
	}

	nextPage := pageNumber + 1
	return &nextPage
}
