package client

import "fmt"

type ApiResponse[T any] struct {
	MaxItems int  `json:"maxItems"`
	Value    []*T `json:"value"`
}

type OrganizationResponse struct {
	OrganizationCode string  `json:"organizationCode"`
	DisplayName      string  `json:"displayName"`
	Description      *string `json:"description,omitempty"`
}

type UserResponse struct {
	UserID              string  `json:"userId"`
	UserName            string  `json:"userName"`
	FirstName           string  `json:"firstName"`
	MiddleName          *string `json:"middleName,omitempty"`
	LastName            string  `json:"lastName"`
	LocalizedFirstName  *string `json:"localizedFirstName,omitempty"`
	LocalizedMiddleName *string `json:"localizedMiddleName,omitempty"`
	LocalizedLastName   *string `json:"localizedLastName,omitempty"`
	SecondaryLastName   *string `json:"secondaryLastName,omitempty"`
	Email               *string `json:"email,omitempty"`
	Title               *string `json:"title,omitempty"`
	ManagerUserName     *string `json:"managerUserName,omitempty"`
	OrganizationCode    string  `json:"organizationCode"`
	OUCode              string  `json:"ouCode"`
	CostCenterNumber    string  `json:"costCenterNumber"`
	LocationCode        string  `json:"locationCode"`
	LanguageCode        string  `json:"languageCode"`
}

type RoleResponse struct {
	RoleCode    string  `json:"roleCode"`
	DisplayName string  `json:"displayName"`
	Description *string `json:"description,omitempty"`
}

type ErrorResponse struct {
	Code   string  `json:"code"`
	Msg    string  `json:"message"`
	Target *string `json:"target,omitempty"`
}

// Implement the required method for the interface.
func (e *ErrorResponse) Message() string {
	target := "none"
	if e.Target != nil {
		target = *e.Target
	}
	return fmt.Sprintf("code: %s, message: %s, target: %s", e.Code, e.Msg, target)
}
