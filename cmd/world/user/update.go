package user

import (
	"context"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/cmd/internal/models"
	"pkg.world.dev/world-cli/cmd/internal/utils"
	"pkg.world.dev/world-cli/common/printer"
)

func (h *Handler) Update(ctx context.Context, flags models.UpdateUserFlags) error {
	// get the current user
	currentUser, err := h.apiClient.GetUser(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get current user")
	}

	printer.NewLine(1)
	printer.Headerln("   Update User   ")

	// prompt update name
	if flags.Name == "" {
		flags.Name = currentUser.Name
	}
	flags.Name, err = h.inputUserName(ctx, flags.Name)
	if err != nil {
		return eris.Wrap(err, "Failed to input user name")
	}

	// prompt for avatar url
	if flags.AvatarURL == "" {
		flags.AvatarURL = currentUser.AvatarURL
	}
	flags.AvatarURL, err = h.inputUserAvatarURL(ctx, flags.AvatarURL)
	if err != nil {
		return eris.Wrap(err, "Failed to input user avatar URL")
	}

	err = h.apiClient.UpdateUser(ctx, flags.Name, currentUser.Email, flags.AvatarURL)
	if err != nil {
		return eris.Wrap(err, "Failed to update user")
	}

	printer.NewLine(1)
	printer.Success("User updated successfully")

	return nil
}

func (h *Handler) inputUserName(ctx context.Context, currentUserName string) (string, error) {
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
			name, err := h.inputService.Prompt(ctx, "Enter name", currentUserName)
			if err != nil {
				return "", err
			}

			if name == "" {
				printer.Errorf("Name cannot be empty\n")
				printer.NewLine(1)
				continue
			}
			return name, nil
		}
	}
}

func (h *Handler) inputUserAvatarURL(ctx context.Context, currentUserAvatarURL string) (string, error) {
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
			avatarURL, err := h.inputService.Prompt(ctx, "Enter avatar URL (Empty Valid)", currentUserAvatarURL)
			if err != nil {
				return "", err
			}
			if avatarURL == "" {
				return avatarURL, nil
			}
			if err := utils.IsValidURL(avatarURL); err != nil {
				printer.Errorln(err.Error())
				printer.NewLine(1)
				continue
			}
			return avatarURL, nil
		}
	}
}
