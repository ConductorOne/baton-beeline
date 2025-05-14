package client

import (
	"context"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
)

type MockBeelineService struct {
	GetUsersFunc            func(ctx context.Context, pageNumber uint) ([]UserResponse, *uint, *v2.RateLimitDescription, error)
	GetOrganizationsFunc    func(ctx context.Context, pageNumber uint) ([]OrganizationResponse, *uint, *v2.RateLimitDescription, error)
	GetRolesFunc            func(ctx context.Context, pageNumber uint) ([]RoleResponse, *uint, *v2.RateLimitDescription, error)
	ListRoleAssignmentsFunc func(ctx context.Context, roleCode string, pageNumber uint) ([]string, *uint, *v2.RateLimitDescription, error)
	AssignRoleToUserFunc    func(ctx context.Context, roleCode string, userID string) (*v2.RateLimitDescription, error)
	RemoveRoleFromUserFunc  func(ctx context.Context, roleCode string, userID string) (*v2.RateLimitDescription, error)
}

func (m *MockBeelineService) GetUsers(ctx context.Context, pageNumber uint) ([]UserResponse, *uint, *v2.RateLimitDescription, error) {
	return m.GetUsersFunc(ctx, pageNumber)
}

func (m *MockBeelineService) GetOrganizations(ctx context.Context, pageNumber uint) ([]OrganizationResponse, *uint, *v2.RateLimitDescription, error) {
	return m.GetOrganizationsFunc(ctx, pageNumber)
}

func (m *MockBeelineService) GetRoles(ctx context.Context, pageNumber uint) ([]RoleResponse, *uint, *v2.RateLimitDescription, error) {
	return m.GetRolesFunc(ctx, pageNumber)
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
