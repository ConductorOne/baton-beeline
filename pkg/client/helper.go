package client

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

func (c *Client) constructURL(
	path string,
	pathParameters map[string]string,
	queryParams map[string]string,
	pageNumber *uint,
	pageSize *uint,
) (*url.URL, error) {
	// Add path parameters
	for _, param := range pathParameters {
		path = strings.Replace(path, "%s", param, 1)
	}

	// Parse the path
	parsed, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse request path '%s': %w", path, err)
	}

	// Resolve the base URL
	url := c.baseAPIURL.ResolveReference(parsed)

	// Add query parameters
	q := url.Query()
	for key, value := range queryParams {
		q.Set(key, value)
	}
	if pageNumber != nil {
		// Safely convert to string without potential integer overflow
		skip := fmt.Sprintf("%d", (*pageNumber)*(*pageSize))
		size := fmt.Sprintf("%d", *pageSize)

		q.Set("skip", skip) // skip a number of records (start) (>= 0)
		q.Set("top", size)  // number of records to return (limit) ([ 0 .. 1000 ])
	}
	q.Set("api-version", apiVersion)
	url.RawQuery = q.Encode()

	return url, nil
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

// toValues converts a map of query parameters to a string.
func toValues(queryParameters map[string]interface{}) string {
	params := url.Values{}
	for key, valueAny := range queryParameters {
		switch value := valueAny.(type) {
		case string:
			params.Add(key, value)
		case int:
			params.Add(key, strconv.Itoa(value))
		case bool:
			params.Add(key, strconv.FormatBool(value))
		default:
			continue
		}
	}
	return params.Encode()
}
