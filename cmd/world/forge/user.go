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

func getUser(fCtx ForgeContext) (User, error) {
	body, err := sendRequest(fCtx, http.MethodGet, userURL, nil)
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

func updateUser(fCtx ForgeContext, flags *UpdateUserCmd) error {
	// get the current user
	currentUser, err := getUser(fCtx)
	if err != nil {
		return eris.Wrap(err, "Failed to get current user")
	}

	printer.NewLine(1)
	printer.Headerln("   Update User   ")

	// prompt update name
	if flags.Name == "" {
		flags.Name = currentUser.Name
	}
	flags.Name, err = inputUserName(fCtx.Context, flags.Name)
	if err != nil {
		return eris.Wrap(err, "Failed to input user name")
	}

	// prompt for avatar url
	if flags.AvatarURL == "" {
		flags.AvatarURL = currentUser.AvatarURL
	}
	flags.AvatarURL, err = inputUserAvatarURL(fCtx.Context, flags.AvatarURL)
	if err != nil {
		return eris.Wrap(err, "Failed to input user avatar URL")
	}

	err = updateUserRequest(fCtx, flags.Name, currentUser.Email, flags.AvatarURL)
	if err != nil {
		return err
	}

	printer.NewLine(1)
	printer.Success("User updated successfully")

	return nil
}

func updateUserRequest(fCtx ForgeContext, name, email, avatarURL string) error {
	payload := User{
		Name:      name,
		Email:     email,
		AvatarURL: avatarURL,
	}

	_, err := sendRequest(fCtx, http.MethodPut, userURL, payload)
	if err != nil {
		return eris.Wrap(err, "Failed to update user")
	}

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
