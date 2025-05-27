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

func updateUser(ctx context.Context, flags *UpdateUserCmd) error {
	// get the current user
	currentUser, err := getUser(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get current user")
	}

	printer.NewLine(1)
	printer.Headerln("   Update User   ")

	// prompt update name
	if flags.Name == "" {
		flags.Name = currentUser.Name
	}
	flags.Name, err = inputUserName(ctx, flags.Name)
	if err != nil {
		return eris.Wrap(err, "Failed to input user name")
	}

	// prompt update email
	if flags.Email == "" {
		flags.Email = currentUser.Email
	}
	flags.Email, err = inputUserEmail(ctx, flags.Email)
	if err != nil {
		return eris.Wrap(err, "Failed to input user email")
	}

	// prompt for avatar url
	if flags.AvatarURL == "" {
		flags.AvatarURL = currentUser.AvatarURL
	}
	flags.AvatarURL, err = inputUserAvatarURL(ctx, flags.AvatarURL)
	if err != nil {
		return eris.Wrap(err, "Failed to input user avatar URL")
	}

	payload := User{
		Name:      flags.Name,
		Email:     flags.Email,
		AvatarURL: flags.AvatarURL,
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
			name := getInput("Enter name", currentUserName)

			if name == "" {
				printer.Errorf("Name cannot be empty\n")
				printer.NewLine(1)
				continue
			}
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
			email := getInput("Enter email", currentUserEmail)
			if !isValidEmail(email) {
				printer.Errorf("Invalid email format\n")
				printer.NewLine(1)
				continue
			}
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
			avatarURL := getInput("Enter avatar URL (Empty Valid)", currentUserAvatarURL)
			if avatarURL == "" {
				return avatarURL, nil
			}
			if err := isValidURL(avatarURL); err != nil {
				printer.Errorln(err.Error())
				printer.NewLine(1)
				continue
			}
			return avatarURL, nil
		}
	}
}
