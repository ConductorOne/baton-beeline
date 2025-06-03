package client

import (
	"context"
	"fmt"
	"net/url"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

const (
	ResourcesPageSize = 100 // [ 0 .. 1000 ]

	// API Docs: https://developers.beeline.com/core_2023-02-28
	apiVersion = "2023-02-28"
	// The first %s is the baseURL and the second %s is the clientSiteId.
	apiURLTemplate = "%s/api/sites/%s"
)

type Client struct {
	baseAPIURL *url.URL
	httpClient *uhttp.BaseHttpClient
}

// NewClient creates a new BeelineClient.
func NewClient(ctx context.Context, baseURL, authServerURL, beelineClientID, beelineClientSecret, beelineClientSiteID string) (*Client, error) {
	// Set up client credentials config.
	config := &clientcredentials.Config{
		EndpointParams: url.Values{
			"audience": {baseURL},
		},
		ClientID:     beelineClientID,
		ClientSecret: beelineClientSecret,
		TokenURL:     authServerURL,
	}

	// Create token source
	tokenSource := config.TokenSource(ctx)

	// Create HTTP client with automatic token handling
	oauthClient := oauth2.NewClient(ctx, tokenSource)

	// Create base API URL
	clientAPIURL, err := url.Parse(fmt.Sprintf(apiURLTemplate, baseURL, beelineClientSiteID))
	if err != nil {
		return nil, fmt.Errorf("error parsing base API URL: %w", err)
	}

	client := &Client{
		baseAPIURL: clientAPIURL,
		httpClient: uhttp.NewBaseHttpClient(oauthClient),
	}

	return client, nil
}

func (c *Client) listOrganizations(ctx context.Context, pageNumber uint) (
	[]*OrganizationResponse,
	*uint,
	*v2.RateLimitDescription,
	error,
) {
	// Doc: https://developers.beeline.com/core_2023-02-28#tag/Organization/operation/get-org-organizations
	// Required scopes: read:org write:org
	path := "/organizations"

	var response struct {
		MaxItems int                     `json:"maxItems"`
		Value    []*OrganizationResponse `json:"value"`
	}

	pageSize := uint(ResourcesPageSize)
	url, err := c.constructURL(path, nil, nil, &pageNumber, &pageSize)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error generating user list URL: %w", err)
	}

	rateLimit, err := c.get(ctx, url, &response)
	if err != nil {
		return nil, nil, rateLimit, fmt.Errorf("error executing request: %w", err)
	}

	nextPageNumber := getNextPageNumber(len(response.Value), pageNumber)

	return response.Value, nextPageNumber, rateLimit, nil
}

func (c *Client) listUsers(ctx context.Context, pageNumber uint) (
	[]*UserResponse,
	*uint,
	*v2.RateLimitDescription,
	error,
) {
	// Doc: https://developers.beeline.com/core_2023-02-28#tag/User/operation/get-users-list
	// Required scopes: read:user write:user
	path := "/users"

	var response ApiResponse[UserResponse]

	pageSize := uint(ResourcesPageSize)
	url, err := c.constructURL(path, nil, nil, &pageNumber, &pageSize)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error generating user list URL: %w", err)
	}

	rateLimit, err := c.get(ctx, url, &response)
	if err != nil {
		return nil, nil, rateLimit, fmt.Errorf("error executing request: %w", err)
	}

	nextPageNumber := getNextPageNumber(len(response.Value), pageNumber)

	return response.Value, nextPageNumber, rateLimit, nil
}

func (c *Client) listRoles(ctx context.Context, pageNumber uint) (
	[]*RoleResponse,
	*uint,
	*v2.RateLimitDescription,
	error,
) {
	// Doc: https://developers.beeline.com/core_2023-02-28#tag/Identity-and-Access-Management/operation/get-iam-roles
	// Required scopes: read:iam write:iam
	path := "/roles"

	var response ApiResponse[RoleResponse]

	pageSize := uint(ResourcesPageSize)
	url, err := c.constructURL(path, nil, nil, &pageNumber, &pageSize)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error generating user list URL: %w", err)
	}

	rateLimit, err := c.get(ctx, url, &response)
	if err != nil {
		return nil, nil, rateLimit, fmt.Errorf("error executing request: %w", err)
	}

	nextPageNumber := getNextPageNumber(len(response.Value), pageNumber)

	return response.Value, nextPageNumber, rateLimit, nil
}

func (c *Client) listRoleAssignments(ctx context.Context, roleCode string, pageNumber uint) (
	[]*string,
	*uint,
	*v2.RateLimitDescription,
	error,
) {
	// Doc: https://developers.beeline.com/core_2023-02-28#tag/Identity-and-Access-Management/operation/get-iam-users-by-role
	// Required scopes: read:iam write:iam
	path := "/roles/{{.roleCode}}/users"
	pathParameters := map[string]string{"roleCode": roleCode}

	var response ApiResponse[string]

	pageSize := uint(ResourcesPageSize)
	url, err := c.constructURL(path, pathParameters, nil, &pageNumber, &pageSize)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error generating user list URL: %w", err)
	}

	rateLimit, err := c.get(ctx, url, &response)
	if err != nil {
		return nil, nil, rateLimit, fmt.Errorf("error executing request: %w", err)
	}

	nextPageNumber := getNextPageNumber(len(response.Value), pageNumber)

	return response.Value, nextPageNumber, rateLimit, nil
}

func (c *Client) assignRoleToUser(ctx context.Context, roleCode string, userID string) (
	*v2.RateLimitDescription,
	error,
) {
	// Doc: https://developers.beeline.com/core_2023-02-28#tag/Identity-and-Access-Management/operation/post-iam-add-user
	// Required scopes: write:iam
	path := "/roles/{{.roleCode}}/users/add"
	pathParameters := map[string]string{"roleCode": roleCode}

	url, err := c.constructURL(path, pathParameters, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error generating user list URL: %w", err)
	}

	payload := map[string]interface{}{"value": []string{userID}}
	// If status code is 200, the request was successful and the target will be empty.
	rateLimit, err := c.post(ctx, url, nil, payload)
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
	path := "/roles/{{.roleCode}}/users/remove"
	pathParameters := map[string]string{"roleCode": roleCode}

	url, err := c.constructURL(path, pathParameters, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error generating user list URL: %w", err)
	}

	payload := map[string]interface{}{"value": []string{userID}}
	// If status code is 200, the request was successful and the target will be empty.
	rateLimit, err := c.post(ctx, url, nil, payload)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}

	return rateLimit, nil
}
