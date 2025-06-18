package models

type User struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

type InviteUserToOrganizationFlags struct {
	Email string
	Role  string
}

type ChangeUserRoleInOrganizationFlags struct {
	Email string
	Role  string
}

type UpdateUserFlags struct {
	Name      string
	AvatarURL string
}
