package models

type Organization struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Slug             string `json:"slug"`
	CreatedTime      string `json:"created_time"`
	UpdatedTime      string `json:"updated_time"`
	OwnerID          string `json:"owner_id"`
	Deleted          bool   `json:"deleted"`
	DeletedTime      string `json:"deleted_time"`
	BaseShardAddress string `json:"base_shard_address"`
}

type OrganizationMember struct {
	Role Role `json:"role"`
	User User `json:"user"`
}

type Role string

const (
	RoleMember Role = "member"
	RoleAdmin  Role = "admin"
	RoleOwner  Role = "owner"
	RoleNone   Role = "none"
)

// RolesMap is used for checking if a role is valid.
var RolesMap = map[Role]struct{}{
	RoleOwner:  {},
	RoleAdmin:  {},
	RoleMember: {},
	RoleNone:   {},
}

type CreateOrganizationFlags struct {
	Name string
	Slug string
}

type SwitchOrganizationFlags struct {
	Slug string
}

type MembersListFlags struct {
	IncludeRemoved bool
}
