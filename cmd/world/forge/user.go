package forge

import (
	"context"
	"net/http"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/common/printer"
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

func updateUser(ctx context.Context, email, name, avatarURL string) error {
	// get the current user
	currentUser, err := getUser(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get current user")
	}

	// prompt update name
	if name == "" {
		name = currentUser.Name
	}
	name, err = inputUserName(ctx, name)
	if err != nil {
		return eris.Wrap(err, "Failed to input user name")
	}

	// prompt update email
	if email == "" {
		email = currentUser.Email
	}
	email, err = inputUserEmail(ctx, email)
	if err != nil {
		return eris.Wrap(err, "Failed to input user email")
	}

	// prompt for avatar url
	if avatarURL == "" {
		avatarURL = currentUser.AvatarURL
	}
	avatarURL, err = inputUserAvatarURL(ctx, avatarURL)
	if err != nil {
		return eris.Wrap(err, "Failed to input user avatar URL")
	}

	payload := User{
		Name:      name,
		Email:     email,
		AvatarURL: avatarURL,
	}

	_, err = sendRequest(ctx, http.MethodPut, userURL, payload)
	if err != nil {
		return eris.Wrap(err, "Failed to update user")
	}

	printer.NewLine(1)
	printer.Success("User updated successfully")

	return nil
}

func inputUserName(ctx context.Context, currentUserName string) (string, error) {
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
			printer.NewLine(1)
			printer.Header("   Update User Name   ")

			printer.NewLine(1)
			name := getInput("Enter name", currentUserName)

			if name == "" {
				printer.NewLine(1)
				printer.Errorf("Error: Name cannot be empty\n")
				continue
			}

			printer.NewLine(1)
			printer.Successf("Name updated to: %s\n", name)
			return name, nil
		}
	}
}

func inputUserEmail(ctx context.Context, currentUserEmail string) (string, error) { // TODO: refactor
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
			printer.NewLine(1)
			printer.Header("   Update User Email   ")

			printer.NewLine(1)
			email := getInput("Enter email", currentUserEmail)
			if !isValidEmail(email) {
				printer.NewLine(1)
				printer.Errorf("Error: Invalid email format\n")
				continue
			}

			printer.NewLine(1)
			printer.Successf("Email updated to: %s\n", email)
			return email, nil
		}
	}
}

func inputUserAvatarURL(ctx context.Context, // TODO: refactor
	currentUserAvatarURL string) (string, error) {
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
			printer.NewLine(1)
			printer.Header("   Update User Avatar URL   ")

			printer.NewLine(1)
			avatarURL := getInput("Enter avatar URL", currentUserAvatarURL)
			if !isValidURL(avatarURL) {
				printer.NewLine(1)
				printer.Errorf("Error: Invalid URL format\n")
				continue
			}

			printer.NewLine(1)
			printer.Successf("Avatar URL updated to: %s\n", avatarURL)
			return avatarURL, nil
		}
	}
}
