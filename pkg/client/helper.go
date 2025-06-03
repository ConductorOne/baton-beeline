package client

import (
	"fmt"
	"net/url"
	"strings"
	"text/template"
)

// constructURL builds the full URL for an API request.
func (c *Client) constructURL(path string, pathParams map[string]string, queryParams map[string]string, pageNumber, pageSize *uint) (*url.URL, error) {
	// Start with the base URL
	u := *c.baseAPIURL

	// Add the path parameters
	if path != "" {
		// Create a template for path parameter replacement
		tmpl, err := template.New("path").Parse(path)
		if err != nil {
			return nil, fmt.Errorf("failed to parse path template: %w", err)
		}

		// Create a buffer to hold the result
		var buf strings.Builder

		// Execute the template with path parameters
		if err := tmpl.Execute(&buf, pathParams); err != nil {
			return nil, fmt.Errorf("failed to execute path template: %w", err)
		}

		// Use the processed path
		u.Path += buf.String()
	}

	// Add pagination query parameters
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

// getNextPageNumber calculates the next page number for pagination
// based on the current results and page size.
// Returns nil if there are no more pages.
func getNextPageNumber(resultCount int, pageNumber uint) *uint {
	if resultCount < ResourcesPageSize {
		return nil
	}

	nextPage := pageNumber + 1
	return &nextPage
}
