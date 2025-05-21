package client

import (
	"context"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
)

type MockBeelineService struct {
	ListUsersFunc           func(ctx context.Context, pageNumber uint) ([]UserResponse, *uint, *v2.RateLimitDescription, error)
	ListOrganizationsFunc   func(ctx context.Context, pageNumber uint) ([]OrganizationResponse, *uint, *v2.RateLimitDescription, error)
	ListRolesFunc           func(ctx context.Context, pageNumber uint) ([]RoleResponse, *uint, *v2.RateLimitDescription, error)
	ListRoleAssignmentsFunc func(ctx context.Context, roleCode string, pageNumber uint) ([]string, *uint, *v2.RateLimitDescription, error)
	AssignRoleToUserFunc    func(ctx context.Context, roleCode string, userID string) (*v2.RateLimitDescription, error)
	RemoveRoleFromUserFunc  func(ctx context.Context, roleCode string, userID string) (*v2.RateLimitDescription, error)
}

func (m *MockBeelineService) ListUsers(ctx context.Context, pageNumber uint) ([]UserResponse, *uint, *v2.RateLimitDescription, error) {
	return m.ListUsersFunc(ctx, pageNumber)
}

func (m *MockBeelineService) ListOrganizations(ctx context.Context, pageNumber uint) ([]OrganizationResponse, *uint, *v2.RateLimitDescription, error) {
	return m.ListOrganizationsFunc(ctx, pageNumber)
}

func (m *MockBeelineService) ListRoles(ctx context.Context, pageNumber uint) ([]RoleResponse, *uint, *v2.RateLimitDescription, error) {
	return m.ListRolesFunc(ctx, pageNumber)
}

func (m *MockBeelineService) ListRoleAssignments(ctx context.Context, roleCode string, pageNumber uint) ([]string, *uint, *v2.RateLimitDescription, error) {
	return m.ListRoleAssignmentsFunc(ctx, roleCode, pageNumber)
}

func (m *MockBeelineService) AssignRoleToUser(ctx context.Context, roleCode string, userID string) (*v2.RateLimitDescription, error) {
	return m.AssignRoleToUserFunc(ctx, roleCode, userID)
}

func (m *MockBeelineService) RemoveRoleFromUser(ctx context.Context, roleCode string, userID string) (*v2.RateLimitDescription, error) {
	return m.RemoveRoleFromUserFunc(ctx, roleCode, userID)
}
