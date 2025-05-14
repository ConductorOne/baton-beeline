package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-beeline/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	ent "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

const roleAssignmentEntitlement = "assigned"

type roleBuilder struct {
	service client.ClientService
}

func (o *roleBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return roleResourceType
}

func (o *roleBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	pageNumber, err := parsePageToken(pToken.Token)
	if err != nil {
		return nil, "", nil, fmt.Errorf("error parsing page token: %w", err)
	}

	outputAnnotations := annotations.New()
	roles, nextPageNumber, rateLimit, err := o.service.GetRoles(ctx, pageNumber)
	outputAnnotations.WithRateLimiting(rateLimit)
	if err != nil {
		return nil, "", outputAnnotations, fmt.Errorf("failed to list roles: %w", err)
	}

	resources := make([]*v2.Resource, len(roles))
	for _, role := range roles {
		roleCopy := role
		roleResource, err := roleResource(&roleCopy)
		if err != nil {
			return nil, "", outputAnnotations, fmt.Errorf("failed to create role resource: %w", err)
		}
		resources = append(resources, roleResource)
	}

	return resources, createPageToken(nextPageNumber), outputAnnotations, nil
}

func (o *roleBuilder) Entitlements(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var rv []*v2.Entitlement

	assignmentOptions := []ent.EntitlementOption{
		ent.WithGrantableTo(userResourceType),
		ent.WithDisplayName(fmt.Sprintf("%s Role Member", resource.DisplayName)),
		ent.WithDescription(fmt.Sprintf("Has the %s role in Beeline", resource.DisplayName)),
	}

	rv = append(rv, ent.NewAssignmentEntitlement(resource, roleAssignmentEntitlement, assignmentOptions...))

	return rv, "", nil, nil
}

func (o *roleBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	pageNumber, err := parsePageToken(pToken.Token)
	if err != nil {
		return nil, "", nil, fmt.Errorf("error parsing page token: %w", err)
	}

	outputAnnotations := annotations.New()
	roleAssignments, nextPageNumber, rateLimit, err := o.service.ListRoleAssignments(ctx, resource.Id.Resource, pageNumber)
	outputAnnotations.WithRateLimiting(rateLimit)
	if err != nil {
		return nil, "", outputAnnotations, fmt.Errorf("failed to list role assignments: %w", err)
	}

	rv := make([]*v2.Grant, len(roleAssignments))
	for _, roleAssignment := range roleAssignments {
		userResource := &v2.Resource{
			Id: &v2.ResourceId{
				ResourceType: userResourceType.Id,
				Resource:     roleAssignment,
			},
		}

		rv = append(rv, grant.NewGrant(resource, roleAssignmentEntitlement, userResource))
	}

	return rv, createPageToken(nextPageNumber), outputAnnotations, nil
}

func (o *roleBuilder) Grant(
	ctx context.Context,
	principal *v2.Resource,
	entitlement *v2.Entitlement,
) (annotations.Annotations, error) {
	logger := ctxzap.Extract(ctx)

	if principal.Id.ResourceType != userResourceType.Id {
		logger.Warn(
			"baton-beeline: only users can be assigned to a role",
			zap.String("principal_type", principal.Id.ResourceType),
			zap.String("principal_id", principal.Id.Resource),
		)
		return nil, fmt.Errorf("baton-beeline: only users can be assigned to a role")
	}

	outputAnnotations := annotations.New()
	// Add the user to the workspace directly without requiring confirmation
	rateLimitData, err := o.service.AssignRoleToUser(
		ctx,
		entitlement.Resource.Id.Resource,
		principal.Id.Resource,
	)
	outputAnnotations.WithRateLimiting(rateLimitData)

	if err != nil {
		// We are not checking if the grant is already exists because the API DOC does not provide specific information.
		return outputAnnotations, fmt.Errorf("baton-beeline: failed to add user to workspace: %w", err)
	}

	return outputAnnotations, nil
}

func (o *roleBuilder) Revoke(
	ctx context.Context,
	grant *v2.Grant,
) (
	annotations.Annotations,
	error,
) {
	logger := ctxzap.Extract(ctx)

	if grant.Principal.Id.ResourceType != userResourceType.Id {
		logger.Warn(
			"baton-beeline: only users can be assigned to a role",
			zap.String("principal_type", grant.Principal.Id.ResourceType),
			zap.String("principal_id", grant.Principal.Id.Resource),
		)
		return nil, fmt.Errorf("baton-beeline: only users can be assigned to a role")
	}

	outputAnnotations := annotations.New()

	rateLimitData, err := o.service.RemoveRoleFromUser(
		ctx,
		grant.Entitlement.Resource.Id.Resource,
		grant.Principal.Id.Resource,
	)
	outputAnnotations.WithRateLimiting(rateLimitData)

	if err != nil {
		// We are not checking if the grant was already revoked because the API DOC does not provide specific information.
		return outputAnnotations, fmt.Errorf("baton-beeline: failed to remove user from workspace: %w", err)
	}

	return outputAnnotations, nil
}

func newRoleBuilder(cclient *client.Client) *roleBuilder {
	return &roleBuilder{
		service: client.NewClientService(cclient),
	}
}

func roleResource(role *client.RoleResponse) (*v2.Resource, error) {
	var description string
	if role.Description != nil {
		description = *role.Description
	}

	profile := map[string]interface{}{
		"role_id":          role.RoleCode,
		"role_name":        role.DisplayName,
		"role_description": description,
	}

	roleTraitOptions := []rs.RoleTraitOption{
		rs.WithRoleProfile(profile),
	}

	resource, err := rs.NewRoleResource(
		role.DisplayName,
		roleResourceType,
		role.RoleCode,
		roleTraitOptions,
	)
	if err != nil {
		return nil, err
	}

	return resource, nil
}
