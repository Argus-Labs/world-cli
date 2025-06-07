package forge

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/common/printer"
)

// ErrOrganizationSlugAlreadyExists is passed from forge to world-cli, Must always match.
var ErrOrganizationSlugAlreadyExists = eris.New("organization slug already exists")

type organization struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Slug             string `json:"slug"`
	CreatedTime      string `json:"created_time"`
	UpdatedTime      string `json:"updated_time"`
	OwnerID          string `json:"owner_id"`
	Deleted          bool   `json:"deleted"`
	DeletedTime      string `json:"deleted_time"`
	BaseShardAddress string `json:"base_shard_address"`
	AvatarURL        string `json:"avatar_url"`
}

type createOrgRequest struct {
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	AvatarURL string `json:"avatar_url"`
}

func showOrganizationList(fCtx ForgeContext) error {
	selectedOrg, err := getSelectedOrganization(fCtx)
	if err != nil {
		return eris.Wrap(err, "Failed to get organization")
	}

	organizations, err := getListOfOrganizations(fCtx)
	if err != nil {
		return eris.Wrap(err, "Failed to get organization list")
	}

	printer.NewLine(1)
	printer.Headerln("  Organization Information  ")
	if selectedOrg.Name == "" {
		printer.Errorln("No organization selected")
		printer.NewLine(1)
		printer.Infoln("Use 'world forge organization switch' to choose an organization")
	} else {
		for _, org := range organizations {
			if org.ID == selectedOrg.ID {
				printer.Infof("• %s (%s) [SELECTED]\n", org.Name, org.Slug)
			} else {
				printer.Infof("  %s (%s)\n", org.Name, org.Slug)
			}
		}
	}
	return nil
}

func getSelectedOrganization(fCtx ForgeContext) (organization, error) {
	if fCtx.Config == nil || fCtx.Config.OrganizationID == "" {
		return organization{}, nil
	}

	// send request
	body, err := sendRequest(fCtx, http.MethodGet,
		fmt.Sprintf("%s/%s", organizationURL, fCtx.Config.OrganizationID), nil)
	if err != nil {
		return organization{}, eris.Wrap(err, "Failed to get organization")
	}

	// parse response
	org, err := parseResponse[organization](body)
	if err != nil {
		return organization{}, eris.Wrap(err, "Failed to parse response")
	}

	return *org, nil
}

func getOrganizationsInvitedTo(fCtx ForgeContext) ([]organization, error) {
	body, err := sendRequest(fCtx, http.MethodGet, fmt.Sprintf("%s/invited", organizationURL), nil)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get organizations")
	}

	orgs, err := parseResponse[[]organization](body)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to parse response")
	}

	return *orgs, nil
}

func (o *organization) acceptOrganizationInvitation(fCtx ForgeContext) error {
	_, err := sendRequest(fCtx, http.MethodPost, fmt.Sprintf("%s/%s/accept-invite", organizationURL, o.ID), nil)
	if err != nil {
		return eris.Wrap(err, "Failed to accept organization invitation")
	}
	return nil
}

func getListOfOrganizations(fCtx ForgeContext) ([]organization, error) {
	// Send request
	body, err := sendRequest(fCtx, http.MethodGet, organizationURL, nil)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get organizations")
	}

	// Parse response
	orgs, err := parseResponse[[]organization](body)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to parse response")
	}

	return *orgs, nil
}

func selectOrganization(fCtx ForgeContext, flags *SwitchOrganizationCmd) (organization, error) {
	if fCtx.Config.CurrRepoKnown {
		printer.Errorf("Cannot switch organization, current git working directory belongs to project: %s.",
			fCtx.Config.CurrProjectName)
		return organization{}, eris.New("Cannot switch organization, directory belongs to another project.")
	}

	// If slug is provided, select organization from slug
	if flags.Slug != "" {
		org, err := selectOrganizationFromSlug(fCtx, flags.Slug)
		if err != nil {
			return organization{}, eris.Wrap(err, "Failed command select organization from slug")
		}
		return org, nil
	}

	orgs, err := getListOfOrganizations(fCtx)
	if err != nil {
		return organization{}, eris.Wrap(err, "Failed to get organizations")
	}

	if len(orgs) == 0 {
		printNoOrganizations()
		return organization{}, nil
	}

	selectedOrg, err := promptForOrganization(fCtx, orgs, false)
	if err != nil {
		return organization{}, err
	}

	err = handleProjectSelection(fCtx)
	if err != nil {
		return organization{}, err
	}

	return selectedOrg, nil
}

func getOrganizationDataByID(fCtx ForgeContext, id string) (organization, error) {
	orgs, err := getListOfOrganizations(fCtx)
	if err != nil {
		return organization{}, eris.Wrap(err, "Failed to get organizations")
	}

	if len(orgs) == 0 {
		return organization{}, eris.New("No organizations found")
	}

	for _, org := range orgs {
		if org.ID == id {
			return org, nil
		}
	}
	return organization{}, eris.New("Organization not found with ID: " + id)
}

func selectOrganizationFromSlug(fCtx ForgeContext, slug string) (organization, error) {
	orgs, err := getListOfOrganizations(fCtx)
	if err != nil {
		return organization{}, eris.Wrap(err, "Failed to get organizations")
	}

	for _, org := range orgs {
		if org.Slug == slug {
			err = org.saveToConfig(fCtx)
			if err != nil {
				return organization{}, eris.Wrap(err, "Failed to save organization")
			}

			err = showOrganizationList(fCtx)
			if err != nil {
				return organization{}, err
			}

			err = handleProjectSelection(fCtx)
			if err != nil {
				return organization{}, err
			}
			return org, nil
		}
	}

	printer.NewLine(1)
	printer.Errorln("Organization not found with slug: " + slug)
	return organization{}, eris.New("Organization not found with slug: " + slug)
}

//nolint:gocognit // Makes sense to keep in one function.
func promptForOrganization(fCtx ForgeContext, orgs []organization, createNew bool) (organization, error) {
	// Display organizations as a numbered list
	printer.NewLine(1)
	printer.Headerln("   Available Organizations  ")
	for i, org := range orgs {
		printer.Infof("  %d. %s\n    └─ Slug: %s\n", i+1, org.Name, org.Slug)
	}

	// Get user input
	var input string
	for {
		select {
		case <-fCtx.Context.Done():
			return organization{}, fCtx.Context.Err()
		default:
			printer.NewLine(1)
			if createNew {
				input = getInput("Enter organization number ('c' to create new or 'q' to quit)", "")
			} else {
				input = getInput("Enter organization number ('q' to quit)", "")
			}

			if input == "q" {
				return organization{}, ErrOrganizationSelectionCanceled
			}

			if input == "c" && createNew {
				org, err := createOrganization(fCtx, &CreateOrganizationCmd{})
				if err != nil {
					return organization{}, eris.Wrap(err, "Failed to create organization")
				}
				return *org, nil
			}

			// Parse selection
			num, err := strconv.Atoi(input)
			if err != nil || num < 1 || num > len(orgs) {
				printer.Errorf("Invalid selection. Please enter a number between 1 and %d\n", len(orgs))
				continue
			}

			selectedOrg := orgs[num-1]

			err = selectedOrg.saveToConfig(fCtx)
			if err != nil {
				return organization{}, eris.Wrap(err, "Failed to save organization")
			}

			printer.Successf("Selected organization: %s\n", selectedOrg.Name)
			return selectedOrg, nil
		}
	}
}

func (o *organization) saveToConfig(fCtx ForgeContext) error {
	fCtx.Config.OrganizationID = o.ID
	err := fCtx.Config.Save()
	if err != nil {
		return eris.Wrap(err, "Failed to save organization")
	}
	return nil
}

//nolint:gocognit,funlen // Makes sense to keep in one function.
func createOrganization(fCtx ForgeContext, flags *CreateOrganizationCmd) (*organization, error) {
	var orgName, orgSlug, orgAvatarURL string

	for {
		// Get organization name
		printer.NewLine(1)
		printer.Headerln("  Create New Organization  ")
		for {
			orgName = getInput("Enter organization name", flags.Name)
			if orgName == "" {
				printer.NewLine(1)
				printer.Errorln("Organization name is required")
				continue
			}
			break
		}

		// Used to create slug from name
		orgSlug = orgName
		if flags.Slug != "" {
			orgSlug = flags.Slug
		}

		// Get and validate organization slug
		for {
			minLength := 3
			maxLength := 15
			orgSlug = CreateSlugFromName(orgSlug, minLength, maxLength)
			orgSlug = getInput("Enter organization slug", orgSlug)

			// Validate slug
			var err error
			orgSlug, err = slugToSaneCheck(orgSlug, minLength, maxLength)
			if err != nil {
				printer.Errorf("%s\n", err)
				printer.NewLine(1)
				continue
			}
			break
		}

		// Get and validate organization avatar URL
		for {
			orgAvatarURL = getInput("Enter organization avatar URL (Empty Valid)", flags.AvatarURL)

			if orgAvatarURL == "" {
				break
			}

			if err := isValidURL(orgAvatarURL); err != nil {
				printer.Errorln(err.Error())
				printer.NewLine(1)
				continue
			}

			break
		}

		// Show confirmation
		printer.NewLine(1)
		printer.Headerln("  Organization Details  ")
		printer.Infof("Name: %s\n", orgName)
		printer.Infof("Slug: %s\n", orgSlug)
		if orgAvatarURL != "" {
			printer.Infof("Avatar URL: %s\n", orgAvatarURL)
		} else {
			printer.Infoln("Avatar URL: None")
		}

		// Get confirmation
		for redo := true; redo; {
			printer.NewLine(1)
			confirm := getInput("Create organization with these details? (Y/n)", "n")
			switch confirm {
			case "Y":
				org, err := createOrgRequestAndSave(fCtx, orgName, orgSlug, orgAvatarURL)
				if err != nil {
					return nil, eris.Wrap(err, "Failed to create organization")
				}
				return org, nil
			case "n":
				redo = false
			default:
				printer.NewLine(1)
				printer.Errorln("Please enter capital 'Y' to confirm, 'n' to cancel, or 'redo' to start over")
			}
		}
	}
}

func createOrgRequestAndSave(fCtx ForgeContext, name, slug, avatarURL string) (*organization, error) {
	// Send request
	body, err := sendRequest(fCtx, http.MethodPost, organizationURL, createOrgRequest{
		Name:      name,
		Slug:      slug,
		AvatarURL: avatarURL,
	})
	if err != nil {
		if eris.Is(err, ErrOrganizationSlugAlreadyExists) {
			printer.Errorf("An Organization already exists with slug: %s, please choose a different slug.\n", slug)
			printer.NewLine(1)
		}
		return nil, eris.Wrap(err, "Failed to create organization")
	}

	// Parse response
	org, err := parseResponse[organization](body)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to parse response")
	}

	// Save organization to config file
	err = org.saveToConfig(fCtx)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to save organization in config")
	}

	printer.NewLine(1)
	printer.Successf("Organization '%s' with slug '%s' created successfully!\n", name, slug)
	return org, nil
}

func (o *organization) inviteUser(fCtx ForgeContext, flags *InviteUserToOrganizationCmd) error {
	printer.NewLine(1)
	printer.Headerln("   Invite User to Organization   ")

	userEmail := getInput("Enter user email to invite", flags.Email)
	if userEmail == "" {
		return eris.New("User email cannot be empty")
	}

	userRole := getRoleInput(false, flags.Role)

	payload := map[string]string{
		"invited_user_email": userEmail,
		"role":               userRole,
	}

	// Send request
	_, err := sendRequest(fCtx, http.MethodPost, fmt.Sprintf("%s/%s/invite", organizationURL, o.ID), payload)
	if err != nil {
		return eris.Wrap(err, "Failed to invite user to organization")
	}

	printer.NewLine(1)
	printer.Successf("Successfully invited user %s to organization!\n", userEmail)
	printer.Infof("Assigned role: %s\n", userRole)
	return nil
}

func (o *organization) updateUserRole(fCtx ForgeContext, flags *ChangeUserRoleInOrganizationCmd) error {
	printer.NewLine(1)
	printer.Headerln("  Update User Role in Organization  ")
	userEmail := getInput("Enter user email to update", flags.Email)

	if userEmail == "" {
		return eris.New("User email cannot be empty")
	}

	userRole := getRoleInput(true, flags.Role)

	payload := map[string]string{
		"target_user_email": userEmail,
		"role":              userRole,
	}

	// Send request
	_, err := sendRequest(fCtx, http.MethodPost, fmt.Sprintf("%s/%s/update-role", organizationURL, o.ID), payload)
	if err != nil {
		printer.Errorf("Failed to set role in organization: %s\n", err)
		return eris.Wrap(err, "Failed to set user role in organization")
	}

	printer.NewLine(1)
	printer.Successf("Successfully updated role for user %s!\n", userEmail)
	printer.Infof("New role: %s\n", userRole)
	return nil
}

func getRoleInput(allowNone bool, role string) string {
	const memberRole = "member"
	// Get and validate role
	var opts, defaultRole string
	defaultRole = role
	if defaultRole == "" {
		defaultRole = memberRole
	}

	if allowNone {
		opts = "owner, admin, member, or none"
	} else {
		opts = "owner, admin, or member"
	}
	for {
		printer.NewLine(1)
		printer.Headerln(" Role Assignment  ")
		printer.Infof("Available Roles: %s\n", opts)
		userRole := getInput("Enter organization role", defaultRole)
		if allowNone && userRole == "none" {
			printer.NewLine(1)
			printer.Infoln("Warning: Role \"none\" removes user from this organization")
			answer := getInput("Confirm removal? (Yes/no)", "no")
			if answer != "Yes" {
				printer.NewLine(1)
				printer.Errorln("User not removed")
				continue // let them try again
			}
			return userRole
		}
		if userRole == "admin" || userRole == "owner" || userRole == memberRole {
			return userRole
		}
		defaultRole = memberRole
		printer.Errorf("Role must be one of %s\n", opts)
	}
}
