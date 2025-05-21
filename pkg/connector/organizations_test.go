package connector

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/conductorone/baton-beeline/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Helper function to create a test builder with mocks.
func newTestOrganizationBuilder() (*organizationBuilder, *client.MockBeelineService) {
	mockClient := &client.Client{}
	mockClientService := &client.MockBeelineService{}

	builder := newOrganizationBuilder(mockClient)
	// Replace the service with our mock.
	builder.service = mockClientService

	return builder, mockClientService
}

func TestOrganizationsList(t *testing.T) {
	ctx := context.Background()

	t.Run("should get ratelimit annotations", func(t *testing.T) {
		// Create a new organization builder with a mock client service.
		organizationBuilder, mockClientService := newTestOrganizationBuilder()

		mockClientService.ListOrganizationsFunc = func(
			ctx context.Context,
			pageNumber uint,
		) (
			[]client.OrganizationResponse,
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

		resources, token, annotations, err := organizationBuilder.List(ctx, nil, &pagination.Token{})

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
		// Create a new organization builder with a mock client service.
		organizationBuilder, mockClientService := newTestOrganizationBuilder()

		startToken := "1"
		mockClientService.ListOrganizationsFunc = func(
			ctx context.Context,
			pageNumber uint,
		) (
			[]client.OrganizationResponse,
			*uint,
			*v2.RateLimitDescription,
			error,
		) {
			require.Equal(t, uint(1), pageNumber)
			return nil, nil, nil, nil
		}

		_, _, _, _ = organizationBuilder.List(ctx, nil, &pagination.Token{Token: startToken})
	})

	t.Run("should get organizations", func(t *testing.T) {
		// Create a new organization builder with a mock client service.
		organizationBuilder, mockClientService := newTestOrganizationBuilder()

		mockClientService.ListOrganizationsFunc = func(
			ctx context.Context,
			pageNumber uint,
		) (
			[]client.OrganizationResponse,
			*uint,
			*v2.RateLimitDescription,
			error,
		) {
			description := "Test Organization"
			organizations := []client.OrganizationResponse{
				{
					OrganizationCode: "ORG1",
					DisplayName:      "Organization 1",
					Description:      &description,
				},
			}
			return organizations, nil, nil, nil
		}

		resources, token, annotations, err := organizationBuilder.List(ctx, nil, &pagination.Token{})

		// Assert the returned organization has an ID.
		require.NotNil(t, resources)
		require.Len(t, resources, 1)
		require.NotEmpty(t, resources[0].Id)

		require.NotNil(t, token)
		AssertNoRatelimitAnnotations(t, annotations)
		require.Nil(t, err)
	})
}
