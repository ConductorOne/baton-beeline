package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"golang.org/x/oauth2"
)

const (
	ResourcesPageSize = 100 // [ 0 .. 1000 ]

	// API Docs: https://developers.beeline.com/core_2023-02-28
	apiVersion = "2023-02-28"
	// The first %s is the baseURL and the second %s is the clientSiteId.
	apiURLTemplate = "%s/api/sites/%s"
)

type Client struct {
	baseAPIURL  *url.URL
	httpClient  *uhttp.BaseHttpClient
	tokenSource oauth2.TokenSource
}

// NewClient creates a new BeelineClient.
func NewClient(ctx context.Context, baseURL, clientSiteID, clientID, clientSecret string) (*Client, error) {
	if clientSiteID == "" || clientID == "" || clientSecret == "" {
		return nil, errors.New("clientSiteID, clientID, and clientSecret are required")
	}

	// Create base HTTP client with longer timeout for API operations
	httpClient, err := uhttp.NewBaseHttpClientWithContext(ctx, &http.Client{
		Timeout: 60 * time.Second, // Increased from 30s to 60s for longer operations
	})
	if err != nil {
		return nil, fmt.Errorf("Error creating http client: %w", err)
	}

	// Create base API URL
	clientAPIURL, err := url.Parse(fmt.Sprintf(apiURLTemplate, baseURL, clientSiteID))
	if err != nil {
		return nil, fmt.Errorf("Error parsing base API URL: %w", err)
	}

	// Create token source
	tokenSource, err := newTokenSource(clientID, clientSecret, clientAPIURL, httpClient.HttpClient)
	if err != nil {
		return nil, fmt.Errorf("Error creating token source: %w", err)
	}

	client := &Client{
		baseAPIURL:  clientAPIURL,
		httpClient:  httpClient,
		tokenSource: tokenSource,
	}

	return client, nil
}

func (c *Client) listOrganizations(ctx context.Context, pageNumber uint) (
	[]OrganizationResponse,
	*uint,
	*v2.RateLimitDescription,
	error,
) {
	// Doc: https://developers.beeline.com/core_2023-02-28#tag/Organization/operation/get-org-organizations
	// Required scopes: read:org write:org
	path := "/organizations"

	var response struct {
		MaxItems int                    `json:"maxItems"`
		Value    []OrganizationResponse `json:"value"`
	}

	pageSize := uint(ResourcesPageSize)
	url, err := c.constructURL(path, nil, nil, &pageNumber, &pageSize)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Error generating user list URL: %w", err)
	}

	rateLimit, err := c.get(ctx, url, &response)
	if err != nil {
		return nil, nil, rateLimit, fmt.Errorf("Error executing request: %w", err)
	}

	nextPageNumber := GetNextPageNumber(len(response.Value), pageNumber)

	return response.Value, nextPageNumber, rateLimit, nil
}

func (c *Client) listUsers(ctx context.Context, pageNumber uint) (
	[]UserResponse,
	*uint,
	*v2.RateLimitDescription,
	error,
) {
	// Doc: https://developers.beeline.com/core_2023-02-28#tag/User/operation/get-users-list
	// Required scopes: read:user write:user
	path := "/users"

	var response struct {
		MaxItems int            `json:"maxItems"`
		Value    []UserResponse `json:"value"`
	}

	pageSize := uint(ResourcesPageSize)
	url, err := c.constructURL(path, nil, nil, &pageNumber, &pageSize)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Error generating user list URL: %w", err)
	}

	rateLimit, err := c.get(ctx, url, &response)
	if err != nil {
		return nil, nil, rateLimit, fmt.Errorf("Error executing request: %w", err)
	}

	nextPageNumber := GetNextPageNumber(len(response.Value), pageNumber)

	return response.Value, nextPageNumber, rateLimit, nil
}

func (c *Client) listRoles(ctx context.Context, pageNumber uint) (
	[]RoleResponse,
	*uint,
	*v2.RateLimitDescription,
	error,
) {
	// Doc: https://developers.beeline.com/core_2023-02-28#tag/Identity-and-Access-Management/operation/get-iam-roles
	// Required scopes: read:iam write:iam
	path := "/roles"

	var response struct {
		MaxItems int            `json:"maxItems"`
		Value    []RoleResponse `json:"value"`
	}

	pageSize := uint(ResourcesPageSize)
	url, err := c.constructURL(path, nil, nil, &pageNumber, &pageSize)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Error generating user list URL: %w", err)
	}

	rateLimit, err := c.get(ctx, url, &response)
	if err != nil {
		return nil, nil, rateLimit, fmt.Errorf("Error executing request: %w", err)
	}

	nextPageNumber := GetNextPageNumber(len(response.Value), pageNumber)

	return response.Value, nextPageNumber, rateLimit, nil
}

func (c *Client) listRoleAssignments(ctx context.Context, roleCode string, pageNumber uint) (
	[]string,
	*uint,
	*v2.RateLimitDescription,
	error,
) {
	// Doc: https://developers.beeline.com/core_2023-02-28#tag/Identity-and-Access-Management/operation/get-iam-users-by-role
	// Required scopes: read:iam write:iam
	path := "/roles/%s/users" // %s is the roleCode

	var response struct {
		MaxItems int      `json:"maxItems"`
		Value    []string `json:"value"` // list of userIds
	}

	pathParameters := map[string]string{"roleCode": roleCode}

	pageSize := uint(ResourcesPageSize)
	url, err := c.constructURL(path, pathParameters, nil, &pageNumber, &pageSize)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error generating user list URL: %w", err)
	}

	rateLimit, err := c.get(ctx, url, &response)
	if err != nil {
		return nil, nil, rateLimit, fmt.Errorf("error executing request: %w", err)
	}

	nextPageNumber := GetNextPageNumber(len(response.Value), pageNumber)

	return response.Value, nextPageNumber, rateLimit, nil
}

func (c *Client) assignRoleToUser(ctx context.Context, roleCode string, userID string) (
	*v2.RateLimitDescription,
	error,
) {
	// Doc: https://developers.beeline.com/core_2023-02-28#tag/Identity-and-Access-Management/operation/post-iam-add-user
	// Required scopes: write:iam
	path := "/roles/%s/users/add" // %s is the roleCode
	pathParameters := map[string]string{"roleCode": roleCode}

	url, err := c.constructURL(path, pathParameters, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error generating user list URL: %w", err)
	}

	payload := map[string]interface{}{"userIds": userID}
	// If status code is 200, the request was successful and the target will be empty.
	response := map[string]interface{}{}

	rateLimit, err := c.post(ctx, url, &response, payload)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}

	return rateLimit, nil
}

func (c *Client) removeRoleFromUser(ctx context.Context, roleCode string, userID string) (
	*v2.RateLimitDescription,
	error,
) {
	// Doc: https://developers.beeline.com/core_2023-02-28#tag/Identity-and-Access-Management/operation/post-iam-remove-user
	// Required scopes: write:iam
	path := "/roles/%s/users/remove" // %s is the roleCode
	pathParameters := map[string]string{"roleCode": roleCode}

	url, err := c.constructURL(path, pathParameters, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error generating user list URL: %w", err)
	}

	payload := map[string]interface{}{"userIds": userID}
	// If status code is 200, the request was successful and the target will be empty.
	response := map[string]interface{}{}

	rateLimit, err := c.post(ctx, url, &response, payload)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}

	return rateLimit, nil
}

// Validate checks if the client can connect to the API.
func (c *Client) Validate(ctx context.Context) error {
	// Test API access by attempting to list users
	_, _, _, err := c.listUsers(ctx, 1)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	return nil
}

// ConcurError represents custom error types for the API client.
type BeelineError struct {
	Code    string
	Message string
}

// Error implements the error interface.
func (e *BeelineError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Common error types.
var (
	ErrRateLimit = &BeelineError{
		Code:    "rate_limit_exceeded",
		Message: "Rate limit exceeded.",
	}
	ErrUnauthorized = &BeelineError{
		Code:    "unauthorized",
		Message: "Unauthorized access.",
	}
	ErrInvalidRequest = &BeelineError{
		Code:    "invalid_request",
		Message: "Invalid request.",
	}
)

// doRequest performs an HTTP request to the SAP Concur API.
// func (c *Client) doRequest(req *http.Request) (*http.Response, error) {
// 	// Get current token
// 	token, err := c.tokenSource.Token()
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get token: %w", err)
// 	}

// 	// Set Authorization header.
// 	token.SetAuthHeader(req) // Add required headers per API spec
// 	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
// 	req.Header.Set("Accept", "application/json")
// 	req.Header.Set("Content-Type", "application/json")
// 	req.Header.Set("concur-correlationid", "baton-concur")
// 	req.Header.Set("Accept-Language", "en")

// 	// Execute request
// 	resp, err := c.httpClient.Do(req)
// 	if err != nil {
// 		return nil, fmt.Errorf("error executing request: %w", err)
// 	}

// 	// Handle rate limiting
// 	if resp.StatusCode == http.StatusTooManyRequests {
// 		return nil, ErrRateLimit
// 	}

// 	// Handle 401 Unauthorized - token may have expired
// 	if resp.StatusCode == http.StatusUnauthorized {
// 		token, err = c.tokenSource.Token()
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to refresh token: %w", err)
// 		}

// 		// Retry request with new token
// 		token.SetAuthHeader(req)
// 		return c.httpClient.Do(req)
// 	}

// 	return resp, nil
// }

// func (c *BeelineClient) genURLNonPaginated(
// 	path string,
// 	pathParameters ...string,
// ) (*url.URL, error) {
// 	for _, param := range pathParameters {
// 		path = strings.Replace(path, "%s", param, 1)
// 	}
// 	parsed, err := url.Parse(path)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to parse request path '%s': %w", path, err)
// 	}
// 	u := c.baseAPIURL.ResolveReference(parsed)
// 	return u, nil
// }

// func (c *Client) genURL(
// 	pageToken string,
// 	path string,
// 	queryParams map[string]string,
// 	pathParameters ...string,
// ) (*url.URL, error) {
// 	for _, param := range pathParameters {
// 		path = strings.Replace(path, "%s", param, 1)
// 	}
// 	parsed, err := url.Parse(path)
// 	if err != nil {
// 		return nil, fmt.Errorf("Failed to parse request path '%s': %w", path, err)
// 	}

// 	u := c.baseAPIURL.ResolveReference(parsed)

// 	q := u.Query()
// 	for key, value := range queryParams {
// 		q.Set(key, value)
// 	}
// 	q.Set("skip", pageToken)                      // skip a number of records (start)
// 	q.Set("top", strconv.Itoa(ResourcesPageSize)) // number of records to return (limit)
// 	q.Set("api-version", apiVersion)
// 	u.RawQuery = q.Encode()

// 	return u, nil
// }
