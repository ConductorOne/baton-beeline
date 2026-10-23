package connector

import (
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
)

var (
	organizationResourceType = &v2.ResourceType{
		Id:          "organization",
		DisplayName: "Organization",
		Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_GROUP},
		Annotations: capabilityPermissions("read:org", "write:org"),
	}

	userResourceType = &v2.ResourceType{
		Id:          "user",
		DisplayName: "User",
		Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_USER},
		Annotations: capabilityPermissions("read:user", "write:user"),
	}

	roleResourceType = &v2.ResourceType{
		Id:          "role",
		DisplayName: "Role",
		Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_ROLE},
		Annotations: capabilityPermissions("read:iam", "write:iam"),
	}
)

func capabilityPermissions(permissions ...string) annotations.Annotations {
	perms := make([]*v2.CapabilityPermission, 0, len(permissions))
	for _, p := range permissions {
		perms = append(perms, v2.CapabilityPermission_builder{Permission: p}.Build())
	}
	return annotations.New(v2.CapabilityPermissions_builder{Permissions: perms}.Build())
}
