package client

import (
	"context"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
)

// ConfluenceService defines the interface for group operations.
type ClientService interface {
	ListUsers(ctx context.Context, pageNumber uint) ([]UserResponse, *uint, *v2.RateLimitDescription, error)
	ListOrganizations(ctx context.Context, pageNumber uint) ([]OrganizationResponse, *uint, *v2.RateLimitDescription, error)
	ListRoles(ctx context.Context, pageNumber uint) ([]RoleResponse, *uint, *v2.RateLimitDescription, error)
	ListRoleAssignments(ctx context.Context, roleCode string, pageNumber uint) ([]string, *uint, *v2.RateLimitDescription, error)
	AssignRoleToUser(ctx context.Context, roleCode string, userID string) (*v2.RateLimitDescription, error)
	RemoveRoleFromUser(ctx context.Context, roleCode string, userID string) (*v2.RateLimitDescription, error)
}

// ClientServiceImpl is the default implementation that calls the actual API.
type ClientServiceImpl struct {
	client Client
}

func NewClientService(client *Client) ClientService {
	return &ClientServiceImpl{client: *client}
}

func (s *ClientServiceImpl) ListUsers(ctx context.Context, pageNumber uint) ([]UserResponse, *uint, *v2.RateLimitDescription, error) {
	return s.client.listUsers(ctx, pageNumber)
}

func (s *ClientServiceImpl) ListOrganizations(ctx context.Context, pageNumber uint) ([]OrganizationResponse, *uint, *v2.RateLimitDescription, error) {
	return s.client.listOrganizations(ctx, pageNumber)
}

func (s *ClientServiceImpl) ListRoles(ctx context.Context, pageNumber uint) ([]RoleResponse, *uint, *v2.RateLimitDescription, error) {
	return s.client.listRoles(ctx, pageNumber)
}

func (s *ClientServiceImpl) ListRoleAssignments(ctx context.Context, roleCode string, pageNumber uint) ([]string, *uint, *v2.RateLimitDescription, error) {
	return s.client.listRoleAssignments(ctx, roleCode, pageNumber)
}

func (s *ClientServiceImpl) AssignRoleToUser(ctx context.Context, roleCode string, userID string) (*v2.RateLimitDescription, error) {
	return s.client.assignRoleToUser(ctx, roleCode, userID)
}

func (s *ClientServiceImpl) RemoveRoleFromUser(ctx context.Context, roleCode string, userID string) (*v2.RateLimitDescription, error) {
	return s.client.removeRoleFromUser(ctx, roleCode, userID)
}
