package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-beeline/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	ent "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

const roleAssignmentEntitlement = "assigned"

var _ connectorbuilder.ResourceSyncerV2 = (*roleBuilder)(nil)

type roleBuilder struct {
	service client.ClientService
}

func (o *roleBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return roleResourceType
}

func (o *roleBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, opts rs.SyncOpAttrs) ([]*v2.Resource, *rs.SyncOpResults, error) {
	pageNumber, err := parsePageToken(opts.PageToken.Token)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing page token: %w", err)
	}

	outputAnnotations := annotations.New()
	roles, nextPageNumber, rateLimit, err := o.service.ListRoles(ctx, pageNumber)
	outputAnnotations.WithRateLimiting(rateLimit)
	if err != nil {
		return nil, &rs.SyncOpResults{Annotations: outputAnnotations}, fmt.Errorf("failed to list roles: %w", err)
	}

	resources := make([]*v2.Resource, 0, len(roles))
	for _, role := range roles {
		roleResource, err := roleResource(role)
		if err != nil {
			return nil, &rs.SyncOpResults{Annotations: outputAnnotations}, fmt.Errorf("failed to create role resource: %w", err)
		}
		resources = append(resources, roleResource)
	}

	return resources, &rs.SyncOpResults{NextPageToken: createPageToken(nextPageNumber), Annotations: outputAnnotations}, nil
}

func (o *roleBuilder) Entitlements(ctx context.Context, resource *v2.Resource, opts rs.SyncOpAttrs) ([]*v2.Entitlement, *rs.SyncOpResults, error) {
	var rv []*v2.Entitlement

	assignmentOptions := []ent.EntitlementOption{
		ent.WithGrantableTo(userResourceType),
		ent.WithDisplayName(fmt.Sprintf("%s Role Member", resource.DisplayName)),
		ent.WithDescription(fmt.Sprintf("Has the %s role in Beeline", resource.DisplayName)),
	}

	rv = append(rv, ent.NewAssignmentEntitlement(resource, roleAssignmentEntitlement, assignmentOptions...))

	return rv, nil, nil
}

func (o *roleBuilder) Grants(ctx context.Context, resource *v2.Resource, opts rs.SyncOpAttrs) ([]*v2.Grant, *rs.SyncOpResults, error) {
	pageNumber, err := parsePageToken(opts.PageToken.Token)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing page token: %w", err)
	}

	outputAnnotations := annotations.New()
	roleAssignments, nextPageNumber, rateLimit, err := o.service.ListRoleAssignments(ctx, resource.Id.Resource, pageNumber)
	outputAnnotations.WithRateLimiting(rateLimit)
	if err != nil {
		return nil, &rs.SyncOpResults{Annotations: outputAnnotations}, fmt.Errorf("failed to list role assignments: %w", err)
	}

	rv := make([]*v2.Grant, 0, len(roleAssignments))
	for _, roleAssignment := range roleAssignments {
		userResource := &v2.Resource{
			Id: &v2.ResourceId{
				ResourceType: userResourceType.Id,
				Resource:     *roleAssignment,
			},
		}

		rv = append(rv, grant.NewGrant(resource, roleAssignmentEntitlement, userResource))
	}

	return rv, &rs.SyncOpResults{NextPageToken: createPageToken(nextPageNumber), Annotations: outputAnnotations}, nil
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
	rateLimitData, err := o.service.AssignRoleToUser(
		ctx,
		entitlement.Resource.Id.Resource,
		principal.Id.Resource,
	)
	outputAnnotations.WithRateLimiting(rateLimitData)

	if err != nil {
		// We are not checking if the grant is already exists because the API DOC does not provide specific information.
		return outputAnnotations, fmt.Errorf("baton-beeline: failed to assign role to user: %w", err)
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
		return nil, fmt.Errorf("baton-beeline: only users can be revoked from a role")
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
		return outputAnnotations, fmt.Errorf("baton-beeline: failed to revoke role from user: %w", err)
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

	resource, err := rs.NewRoleResource(
		role.DisplayName,
		roleResourceType,
		role.RoleCode,
		nil,
		rs.WithResourceProfile(profile),
	)
	if err != nil {
		return nil, err
	}

	return resource, nil
}
