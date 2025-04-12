package forge

import (
	"context"
	"fmt"
	"net/http"

	"github.com/rotisserie/eris"
)

type User struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

func getUser(ctx context.Context) (User, error) {
	body, err := sendRequest(ctx, http.MethodGet, userURL, nil)
	if err != nil {
		return User{}, eris.Wrap(err, "Failed to get user")
	}

	user, err := parseResponse[User](body)
	if err != nil {
		return User{}, eris.Wrap(err, "Failed to parse user")
	}

	if user == nil {
		return User{}, eris.New("User not found")
	}

	return *user, nil
}

func updateUser(ctx context.Context) error {
	// get the current user
	currentUser, err := getUser(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get current user")
	}

	payload := User{}

	// prompt update name
	name, err := inputUserName(ctx, currentUser.Name)
	if err != nil {
		return eris.Wrap(err, "Failed to input user name")
	}

	payload.Name = name

	// prompt update email
	email, err := inputUserEmail(ctx, currentUser.Email)
	if err != nil {
		return eris.Wrap(err, "Failed to input user email")
	}

	payload.Email = email

	// prompt for avatar url
	avatarURL, err := inputUserAvatarURL(ctx, currentUser.AvatarURL)
	if err != nil {
		return eris.Wrap(err, "Failed to input user avatar URL")
	}

	payload.AvatarURL = avatarURL

	_, err = sendRequest(ctx, http.MethodPut, userURL, payload)
	if err != nil {
		return eris.Wrap(err, "Failed to update user")
	}

	fmt.Println("\n✅ User updated successfully")

	return nil
}

func inputUserName(ctx context.Context, currentUserName string) (string, error) {
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
			fmt.Println("\n   Update User Name")
			fmt.Println("======================")

			name := getInput("\nEnter name", currentUserName)

			if name == "" {
				fmt.Printf("\n❌ Error: Name cannot be empty\n")
				continue
			}

			fmt.Printf("\n✅ Name updated to: %s\n", name)
			return name, nil
		}
	}
}

func inputUserEmail(ctx context.Context, currentUserEmail string) (string, error) { //nolint:dupl // TODO: refactor
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
			fmt.Println("\n   Update User Email")
			fmt.Println("=======================")

			email := getInput("\nEnter email", currentUserEmail)
			if !isValidEmail(email) {
				fmt.Printf("\n❌ Error: Invalid email format\n")
				continue
			}

			fmt.Printf("\n✅ Email updated to: %s\n", email)
			return email, nil
		}
	}
}

func inputUserAvatarURL(ctx context.Context, //nolint:dupl // TODO: refactor
	currentUserAvatarURL string) (string, error) {
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
			fmt.Println("\n  Update User Avatar URL")
			fmt.Println("=============================")
			avatarURL := getInput("\nEnter avatar URL", currentUserAvatarURL)
			if !isValidURL(avatarURL) {
				fmt.Printf("\n❌ Error: Invalid URL format\n")
				continue
			}

			fmt.Printf("\n✅ Avatar URL updated to: %s\n", avatarURL)
			return avatarURL, nil
		}
	}
}
