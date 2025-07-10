package models

import "time"

type Credential struct {
	Token          string    `json:"token"`
	TokenExpiresAt time.Time `json:"token_expires_at,omitempty"`
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Email          string    `json:"email"`
}

// LoginToken struct for argusID.
type LoginToken struct {
	Status string `json:"status"`
	JWT    string `json:"jwt"`
}
