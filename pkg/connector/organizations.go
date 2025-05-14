package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-beeline/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	ent "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
)

const organizationMemberEntitlement = "member"

type organizationBuilder struct {
	service client.ClientService
}

func (o *organizationBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return organizationResourceType
}

func (o *organizationBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	pageNumber, err := parsePageToken(pToken.Token)
	if err != nil {
		return nil, "", nil, fmt.Errorf("error parsing page token: %w", err)
	}

	outputAnnotations := annotations.New()
	organizations, nextPageNumber, rateLimit, err := o.service.GetOrganizations(ctx, pageNumber)
	outputAnnotations.WithRateLimiting(rateLimit)
	if err != nil {
		return nil, "", outputAnnotations, fmt.Errorf("failed to list organizations: %w", err)
	}

	resources := make([]*v2.Resource, len(organizations))
	for _, organization := range organizations {
		organizationCopy := organization
		organizationResource, err := organizationResource(&organizationCopy)
		if err != nil {
			return nil, "", outputAnnotations, fmt.Errorf("failed to create organization resource: %w", err)
		}
		resources = append(resources, organizationResource)
	}

	return resources, createPageToken(nextPageNumber), outputAnnotations, nil
}

func (o *organizationBuilder) Entitlements(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	entitlements := make([]*v2.Entitlement, 0, 1)

	// Generate display name and description
	displayName := fmt.Sprintf("%s %s", resource.DisplayName, organizationMemberEntitlement)
	description := fmt.Sprintf("%s role in %s Beeline organization", organizationMemberEntitlement, resource.DisplayName)

	// Define entitlement options
	entitlementOptions := []ent.EntitlementOption{
		ent.WithGrantableTo(userResourceType),
		ent.WithDisplayName(displayName),
		ent.WithDescription(description),
	}

	// Append new entitlement to the slice
	entitlements = append(entitlements, ent.NewPermissionEntitlement(resource, organizationMemberEntitlement, entitlementOptions...))

	return entitlements, "", nil, nil
}

// Grants will be created from userResource grants. Due to how
// the Beeline API works, it is more efficient to emit these grants while
// listing grants for each individual user. Instead of having to list users for
// each organization.
func (o *organizationBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func newOrganizationBuilder(cclient *client.Client) *organizationBuilder {
	return &organizationBuilder{
		service: client.NewClientService(cclient),
	}
}

func organizationResource(organization *client.OrganizationResponse) (*v2.Resource, error) {
	var description string
	if organization.Description != nil {
		description = *organization.Description
	}

	resource, err := rs.NewResource(
		organization.DisplayName,
		organizationResourceType,
		organization.OrganizationCode,
		rs.WithDescription(description),
	)
	if err != nil {
		return nil, err
	}

	return resource, nil
}
