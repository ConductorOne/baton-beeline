package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-beeline/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
)

var _ connectorbuilder.ResourceSyncerV2 = (*userBuilder)(nil)

type userBuilder struct {
	service client.ClientService
}

func (o *userBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return userResourceType
}

// List returns all the users from the database as resource objects.
// Users include a UserTrait because they are the 'shape' of a standard user.
func (o *userBuilder) List(ctx context.Context, _ *v2.ResourceId, opts rs.SyncOpAttrs) ([]*v2.Resource, *rs.SyncOpResults, error) {
	pageNumber, err := parsePageToken(opts.PageToken.Token)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing page token: %w", err)
	}

	outputAnnotations := annotations.New()
	users, nextPageNumber, rateLimit, err := o.service.ListUsers(ctx, pageNumber)
	outputAnnotations.WithRateLimiting(rateLimit)
	if err != nil {
		return nil, &rs.SyncOpResults{Annotations: outputAnnotations}, fmt.Errorf("failed to list users: %w", err)
	}

	resources := make([]*v2.Resource, 0, len(users))
	for _, user := range users {
		userResource, err := userResource(user)
		if err != nil {
			return nil, &rs.SyncOpResults{Annotations: outputAnnotations}, fmt.Errorf("failed to create user resource: %w", err)
		}
		resources = append(resources, userResource)
	}

	return resources, &rs.SyncOpResults{NextPageToken: createPageToken(nextPageNumber), Annotations: outputAnnotations}, nil
}

// Entitlements always returns an empty slice for users.
func (o *userBuilder) Entitlements(_ context.Context, resource *v2.Resource, _ rs.SyncOpAttrs) ([]*v2.Entitlement, *rs.SyncOpResults, error) {
	return nil, nil, nil
}

// Grants emits a grant for the organization that the user belongs to.
// This is more efficient than emitting organization grants in the organization resource.
func (o *userBuilder) Grants(ctx context.Context, resource *v2.Resource, opts rs.SyncOpAttrs) ([]*v2.Grant, *rs.SyncOpResults, error) {
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

	return rv, nil, nil
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

	var userTraitOptions []rs.UserTraitOption

	// Add email trait only if email exists
	if user.Email != nil {
		userTraitOptions = append(userTraitOptions, rs.WithEmail(*user.Email, true))
	}

	resource, err := rs.NewUserResource(
		fullName,
		userResourceType,
		user.UserID,
		userTraitOptions,
		rs.WithResourceProfile(profile),
		rs.WithResourceStatus(v2.Status_RESOURCE_STATUS_ENABLED, ""),
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
