package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-beeline/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
)

type userBuilder struct {
	service client.ClientService
}

func (o *userBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return userResourceType
}

// List returns all the users from the database as resource objects.
// Users include a UserTrait because they are the 'shape' of a standard user.
func (o *userBuilder) List(ctx context.Context, _ *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	pageNumber, err := parsePageToken(pToken.Token)
	if err != nil {
		return nil, "", nil, fmt.Errorf("error parsing page token: %w", err)
	}

	outputAnnotations := annotations.New()
	users, nextPageNumber, rateLimit, err := o.service.ListUsers(ctx, pageNumber)
	outputAnnotations.WithRateLimiting(rateLimit)
	if err != nil {
		return nil, "", outputAnnotations, fmt.Errorf("failed to list users: %w", err)
	}

	resources := make([]*v2.Resource, 0, len(users))
	for _, user := range users {
		userCopy := user
		userResource, err := userResource(&userCopy)
		if err != nil {
			return nil, "", outputAnnotations, fmt.Errorf("failed to create user resource: %w", err)
		}
		resources = append(resources, userResource)
	}

	return resources, createPageToken(nextPageNumber), outputAnnotations, nil
}

// Entitlements always returns an empty slice for users.
func (o *userBuilder) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

// Grants emits a grant for the organization that the user belongs to.
// This is more efficient than emitting organization grants in the organization resource.
func (o *userBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	var rv []*v2.Grant
	organizationResource := &v2.Resource{
		Id: &v2.ResourceId{
			ResourceType: organizationResourceType.Id,
			Resource:     resource.ParentResourceId.Resource,
		},
	}

	rv = append(rv, grant.NewGrant(
		organizationResource,
		organizationMemberEntitlement,
		resource,
	))

	return rv, "", nil, nil
}

func newUserBuilder(cclient *client.Client) *userBuilder {
	return &userBuilder{
		service: client.NewClientService(cclient),
	}
}

// userResource creates a resource object for a userResponse.
func userResource(user *client.UserResponse) (*v2.Resource, error) {
	fullName := user.FirstName + " " + user.LastName

	// Create profile map with non-pointer fields
	profile := map[string]interface{}{
		"username":      user.UserName,
		"name":          fullName,
		"ou_code":       user.OUCode, // The organizational unit within organization to which the user belongs.
		"location_code": user.LocationCode,
		"language_code": user.LanguageCode,
	}

	// Add pointer fields only if they're not nil
	if user.Email != nil {
		profile["email"] = *user.Email
	}
	if user.Title != nil {
		profile["title"] = *user.Title
	}

	userTraitOptions := []rs.UserTraitOption{
		rs.WithUserProfile(profile),
		rs.WithStatus(v2.UserTrait_Status_STATUS_ENABLED),
	}

	// Add email trait only if email exists
	if user.Email != nil {
		userTraitOptions = append(userTraitOptions, rs.WithEmail(*user.Email, true))
	}

	resource, err := rs.NewUserResource(
		fullName,
		userResourceType,
		user.UserID,
		userTraitOptions,
		rs.WithParentResourceID(&v2.ResourceId{
			ResourceType: organizationResourceType.Id,
			Resource:     user.OrganizationCode,
		}),
	)

	if err != nil {
		return nil, err
	}

	return resource, nil
}
