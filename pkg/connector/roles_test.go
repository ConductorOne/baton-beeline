package connector

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/conductorone/baton-beeline/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Helper function to create a test builder with mocks.
func newTestRoleBuilder() (*roleBuilder, *client.MockBeelineService) {
	mockClient := &client.Client{}
	mockClientService := &client.MockBeelineService{}

	builder := newRoleBuilder(mockClient)
	// Replace the service with our mock.
	builder.service = mockClientService

	return builder, mockClientService
}

func TestRolesList(t *testing.T) {
	ctx := context.Background()

	t.Run("should get ratelimit annotations", func(t *testing.T) {
		// Create a new role builder with a mock client service.
		roleBuilder, mockClientService := newTestRoleBuilder()

		mockClientService.ListRolesFunc = func(
			ctx context.Context,
			pageNumber uint,
		) (
			[]client.RoleResponse,
			*uint,
			*v2.RateLimitDescription,
			error,
		) {
			rateLimitData := v2.RateLimitDescription{
				ResetAt: timestamppb.New(time.Now().Add(10 * time.Second)),
			}
			err := fmt.Errorf("ratelimit error")
			return nil, nil, &rateLimitData, err
		}

		resources, token, annotations, err := roleBuilder.List(ctx, nil, &pagination.Token{})

		require.Nil(t, resources)
		require.Empty(t, token)
		require.NotNil(t, err)

		// There should be annotations.
		require.Len(t, annotations, 1)
		rateLimitData := v2.RateLimitDescription{}
		err = annotations[0].UnmarshalTo(&rateLimitData)
		if err != nil {
			t.Errorf("couldn't unmarshal the ratelimit annotation")
		}
		require.NotNil(t, rateLimitData.ResetAt)
	})

	t.Run("should get passed a pagination token", func(t *testing.T) {
		// Create a new role builder with a mock client service.
		roleBuilder, mockClientService := newTestRoleBuilder()

		startToken := "1"
		mockClientService.ListRolesFunc = func(
			ctx context.Context,
			pageNumber uint,
		) (
			[]client.RoleResponse,
			*uint,
			*v2.RateLimitDescription,
			error,
		) {
			require.Equal(t, uint(1), pageNumber)
			return nil, nil, nil, nil
		}

		_, _, _, _ = roleBuilder.List(ctx, nil, &pagination.Token{Token: startToken})
	})

	t.Run("should get roles", func(t *testing.T) {
		// Create a new role builder with a mock client service.
		roleBuilder, mockClientService := newTestRoleBuilder()

		mockClientService.ListRolesFunc = func(
			ctx context.Context,
			pageNumber uint,
		) (
			[]client.RoleResponse,
			*uint,
			*v2.RateLimitDescription,
			error,
		) {
			description := "Test Role"
			roles := []client.RoleResponse{
				{
					RoleCode:    "ROLE1",
					DisplayName: "Role 1",
					Description: &description,
				},
			}
			return roles, nil, nil, nil
		}

		resources, token, annotations, err := roleBuilder.List(ctx, nil, &pagination.Token{})

		// Assert the returned role has an ID.
		require.NotNil(t, resources)
		require.Len(t, resources, 1)
		require.NotEmpty(t, resources[0].Id)

		require.NotNil(t, token)
		AssertNoRatelimitAnnotations(t, annotations)
		require.Nil(t, err)
	})
}

func TestRoleGrantAndRevoke(t *testing.T) {
	ctx := context.Background()

	t.Run("Grant operation for role with valid principal and entitlement", func(t *testing.T) {
		roleBuilder, mockService := newTestRoleBuilder()
		// Mock the add user to role call.
		mockService.AssignRoleToUserFunc = func(ctx context.Context, roleID, userID string) (*v2.RateLimitDescription, error) {
			assert.Equal(t, "test-role", roleID)
			assert.Equal(t, "test-user", userID)
			return nil, nil
		}

		// Create a grant request.
		principal := &v2.Resource{
			Id: &v2.ResourceId{
				ResourceType: userResourceType.Id,
				Resource:     "test-user",
			},
		}

		entitlement := &v2.Entitlement{
			Resource: &v2.Resource{
				Id: &v2.ResourceId{
					Resource: "test-role",
				},
			},
		}

		// Execute Grant.
		_, err := roleBuilder.Grant(ctx, principal, entitlement)

		// Verify the result.
		require.NoError(t, err)
	})

	t.Run("Grant operation for role with invalid principal", func(t *testing.T) {
		roleBuilder, _ := newTestRoleBuilder()

		principal := &v2.Resource{
			Id: &v2.ResourceId{
				ResourceType: "invalid-type",
				Resource:     "test-user",
			},
		}

		entitlement := &v2.Entitlement{
			Resource: &v2.Resource{
				Id: &v2.ResourceId{
					Resource: "test-role",
				},
			},
		}

		// Execute Grant.
		_, err := roleBuilder.Grant(ctx, principal, entitlement)

		// Verify the error.
		require.Error(t, err)
		assert.Contains(t, err.Error(), "baton-beeline: only users can be assigned to a role")
	})

	t.Run("Revoke operation for role with valid principal and entitlement", func(t *testing.T) {
		roleBuilder, mockService := newTestRoleBuilder()
		// Mock the remove user from role call.
		mockService.RemoveRoleFromUserFunc = func(ctx context.Context, roleID, userID string) (*v2.RateLimitDescription, error) {
			assert.Equal(t, "test-role", roleID)
			assert.Equal(t, "test-user", userID)
			return nil, nil
		}

		principal := &v2.Resource{
			Id: &v2.ResourceId{
				ResourceType: userResourceType.Id,
				Resource:     "test-user",
			},
		}

		entitlement := &v2.Entitlement{
			Resource: &v2.Resource{
				Id: &v2.ResourceId{
					Resource: "test-role",
				},
			},
		}

		grant := &v2.Grant{
			Principal:   principal,
			Entitlement: entitlement,
		}

		// Execute Revoke.
		_, err := roleBuilder.Revoke(ctx, grant)

		// Verify the result.
		require.NoError(t, err)
	})

	t.Run("Revoke operation for role with invalid principal", func(t *testing.T) {
		roleBuilder, _ := newTestRoleBuilder()

		principal := &v2.Resource{
			Id: &v2.ResourceId{
				ResourceType: "invalid-type",
				Resource:     "test-user",
			},
		}

		entitlement := &v2.Entitlement{
			Resource: &v2.Resource{
				Id: &v2.ResourceId{
					Resource: "test-role",
				},
			},
		}

		grant := &v2.Grant{
			Principal:   principal,
			Entitlement: entitlement,
		}

		// Execute Revoke.
		_, err := roleBuilder.Revoke(ctx, grant)

		// Verify the error.
		require.Error(t, err)
		assert.Contains(t, err.Error(), "baton-beeline: only users can be revoked from a role")
	})
}
