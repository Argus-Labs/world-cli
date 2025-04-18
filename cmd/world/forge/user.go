package forge

import (
	"context"
	"fmt"
	"net/http"

	"github.com/rotisserie/eris"
)

type User struct {
	ID        string `json:"id"`
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
	maxAttempts := 5
	attempts := 0

	for attempts < maxAttempts {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
			fmt.Println("\n   Update User Name")
			fmt.Println("======================")

			if currentUserName != "" {
				fmt.Printf("\nCurrent name: %s\n", currentUserName)
				fmt.Printf("\nEnter new name (or press Enter to keep current): ")
			} else {
				fmt.Printf("\nEnter name: ")
			}

			name, err := getInput()
			if err != nil {
				attempts++
				fmt.Printf("\n❌ Invalid input. Please enter a name (attempt %d/%d)\n", attempts, maxAttempts)
				continue
			}

			if name == "" && currentUserName != "" {
				// Keep current name if empty input
				return currentUserName, nil
			}

			if name == "" {
				fmt.Printf("\n❌ Error: Name cannot be empty (attempt %d/%d)\n", attempts+1, maxAttempts)
				attempts++
				continue
			}

			fmt.Printf("\n✅ Name updated to: %s\n", name)
			return name, nil
		}
	}

	return "", eris.New("Maximum attempts reached for entering name")
}

func inputUserEmail(ctx context.Context, currentUserEmail string) (string, error) { //nolint:dupl // TODO: refactor
	maxAttempts := 5
	attempts := 0

	for attempts < maxAttempts {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
			fmt.Println("\n   Update User Email")
			fmt.Println("=======================")

			if currentUserEmail != "" {
				fmt.Printf("\nCurrent email: %s\n", currentUserEmail)
				fmt.Printf("\nEnter new email (or press Enter to keep current): ")
			} else {
				fmt.Printf("\nEnter email: ")
			}

			email, err := getInput()
			if err != nil {
				attempts++
				fmt.Printf("\n❌ Invalid input. Please enter an email (attempt %d/%d)\n", attempts, maxAttempts)
				continue
			}

			if email == "" && currentUserEmail != "" {
				// Keep current email if empty input
				return currentUserEmail, nil
			}

			if !isValidEmail(email) {
				fmt.Printf("\n❌ Error: Invalid email format (attempt %d/%d)\n", attempts+1, maxAttempts)
				attempts++
				continue
			}

			fmt.Printf("\n✅ Email updated to: %s\n", email)
			return email, nil
		}
	}

	return "", eris.New("Maximum attempts reached for entering email")
}

func inputUserAvatarURL(ctx context.Context, //nolint:dupl // TODO: refactor
	currentUserAvatarURL string) (string, error) {
	attempts := 0
	maxAttempts := 5

	for attempts < maxAttempts {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
			fmt.Println("\n   Update User Avatar URL")
			fmt.Println("============================")

			if currentUserAvatarURL != "" {
				fmt.Printf("\nCurrent avatar URL: %s\n", currentUserAvatarURL)
				fmt.Print("\nEnter new avatar URL (press Enter to keep current): ")
			} else {
				fmt.Print("\nEnter avatar URL: ")
			}

			avatarURL, err := getInput()
			if err != nil {
				attempts++
				fmt.Printf("\n❌ Invalid input. Please enter an avatar URL (attempt %d/%d)\n",
					attempts, maxAttempts)
				continue
			}

			if avatarURL == "" && currentUserAvatarURL != "" {
				// Keep current avatar URL if empty input
				return currentUserAvatarURL, nil
			}

			if !isValidURL(avatarURL) {
				fmt.Printf("\n❌ Error: Invalid URL format (attempt %d/%d)\n",
					attempts+1, maxAttempts)
				attempts++
				continue
			}

			fmt.Printf("\n✅ Avatar URL updated to: %s\n", avatarURL)
			return avatarURL, nil
		}
	}

	return "", eris.New("Maximum attempts reached for entering avatar URL")
}
