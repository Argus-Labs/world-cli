package project

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/cmd/internal/clients/api"
	"pkg.world.dev/world-cli/cmd/internal/models"
	"pkg.world.dev/world-cli/cmd/internal/utils"
	"pkg.world.dev/world-cli/common/printer"
	"pkg.world.dev/world-cli/common/tomlutil"
	"pkg.world.dev/world-cli/common/util"
	"pkg.world.dev/world-cli/tea/component/multiselect"
)

const MaxProjectNameLen = 50

// TODO: un global this but atleast with new command refactor not affecting other command tests!
var regionSelector *tea.Program

var (
	ErrCannotCreateSwitchProject = eris.New("Cannot create/switch Project, directory belongs to another project.")
)

func (h *Handler) Create(
	ctx context.Context,
	flags models.CreateProjectFlags,
) (models.Project, error) {
	if h.configService.GetConfig().CurrRepoKnown {
		printer.Errorf("Cannot create Project, current git working directory belongs to project: %s.",
			h.configService.GetConfig().CurrProjectName)
		return models.Project{}, ErrCannotCreateSwitchProject
	}

	repoPath, repoURL, err := h.PreCreateUpdateValidation()
	if err != nil {
		printRequiredStepsToCreateProject()
		return models.Project{}, eris.Wrap(err, "Failed to validate project creation")
	}

	regions, err := h.getListOfAvailableRegionsForNewProject(ctx)
	if err != nil {
		return models.Project{}, eris.Wrap(err, "Failed to get available regions")
	}

	printer.NewLine(1)
	printer.Headerln("   Project Creation   ")

	p := models.Project{
		Name:      flags.Name,
		Slug:      flags.Slug,
		AvatarURL: flags.AvatarURL,
		RepoPath:  repoPath,
		RepoURL:   repoURL,
		Update:    false,
	}
	err = h.getSetupInput(ctx, &p, regions)
	if err != nil {
		return models.Project{}, eris.Wrap(err, "Failed to get project input")
	}

	// Send request
	prj, err := h.apiClient.CreateProject(ctx, p.OrgID, p)
	if err != nil {
		if eris.Is(err, api.ErrProjectSlugAlreadyExists) {
			printer.Errorf("Project already exists with slug: %s, please choose a different slug.\n", p.Slug)
			printer.NewLine(1)
		}
		return models.Project{}, eris.Wrap(err, "Failed to create project")
	}

	// Select project
	err = h.saveToConfig(&prj)
	if err != nil {
		return models.Project{}, eris.Wrap(err, "Failed to save project configuration")
	}

	displayProjectDetails(&prj)
	printer.Infof("• Deploy Secret (for deploy via CI/CD pipeline tools): ")
	printer.Infof("%s\n", prj.DeploySecret)
	printer.Notificationln("Note: Deploy Secret will not be shown again. Save it now in a secure location.")

	printer.NewLine(1)
	printer.Successf("Created project: %s [%s]\n", prj.Name, prj.Slug)
	return prj, nil
}

func (h *Handler) getSetupInput(ctx context.Context, project *models.Project, regions []string) error {
	// Get organization
	org, err := h.apiClient.GetOrganizationByID(ctx, h.configService.GetConfig().OrganizationID)
	if err != nil {
		return eris.Wrap(err, "Failed to get organization")
	}

	if org.ID == "" || eris.Is(err, api.ErrNoOrganizationID) {
		printNoSelectedOrganization()
		return nil
	}

	project.OrgID = org.ID

	err = h.inputName(ctx, project)
	if err != nil {
		return eris.Wrap(err, "Failed to get project name")
	}

	err = h.inputSlug(ctx, project)
	if err != nil {
		return eris.Wrap(err, "Failed to get project slug")
	}

	err = h.inputRepoURLAndToken(ctx, project)
	if err != nil {
		return eris.Wrap(err, "Failed to get repository URL and token")
	}

	h.inputRepoPath(ctx, project)

	// Regions
	err = h.chooseRegion(ctx, project, regions)
	if err != nil {
		return eris.Wrap(err, "Failed to choose region")
	}

	// Discord
	err = h.inputDiscord(ctx, project)
	if err != nil {
		return eris.Wrap(err, "Failed to input discord")
	}

	// Slack
	err = h.inputSlack(ctx, project)
	if err != nil {
		return eris.Wrap(err, "Failed to input slack")
	}

	err = h.inputAvatarURL(ctx, project)
	if err != nil {
		return eris.Wrap(err, "Failed to input avatar URL")
	}

	return nil
}

func (h *Handler) inputName(ctx context.Context, project *models.Project) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		// If name is not set from cmd flags, get it from world.toml
		if project.Name == "" {
			// Get project name from world.toml if it exists, fails silently
			err := getForgeProjectNameFromWorldToml(project)
			if err != nil {
				project.Name = ""
			}
		}

		name, err := h.inputService.Prompt(ctx, "Enter project name", project.Name)
		if err != nil {
			return eris.Wrap(err, "Failed to get project name")
		}

		err = validateAndSetName(name, project)
		if err == nil {
			return nil
		}
		// If validation fails, clear the name to attempt from toml
		project.Name = ""
	}
}

func getForgeProjectNameFromWorldToml(project *models.Project) error {
	cwd, err := os.Getwd()
	if err != nil {
		return eris.Wrap(err, "Failed to get current working directory")
	}

	absProjectDir := filepath.Join(cwd, "world.toml")

	// Get the forge section from world.toml
	forgeSection, err := tomlutil.GetTOMLSection(absProjectDir, "forge")
	if err != nil {
		return eris.Wrap(err, "Failed to read forge section from world.toml")
	}
	if forgeSection == nil {
		return eris.New("forge section not found in world.toml")
	}

	projectName, ok := forgeSection["PROJECT_NAME"].(string)
	if !ok {
		return eris.New("PROJECT_NAME not found in forge section")
	}

	if err := validateAndSetName(projectName, project); err != nil {
		return eris.Wrap(err, "invalid project name in world.toml")
	}
	return nil
}

func validateAndSetName(name string, project *models.Project) error {
	if err := utils.ValidateName(name, MaxProjectNameLen); err != nil {
		return err
	}
	project.Name = name
	return nil
}

//nolint:gocognit // Belongs in a single function
func (h *Handler) inputSlug(ctx context.Context, project *models.Project) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// if no slug exists, create a default one from the name
			minLength := 3
			maxLength := 25
			if project.Slug == "" {
				project.Slug = utils.CreateSlugFromName(project.Name, minLength, maxLength)
			} else {
				project.Slug = utils.CreateSlugFromName(project.Slug, minLength, maxLength)
			}

			slug, err := h.inputService.Prompt(ctx, "Slug", project.Slug)
			if err != nil {
				return eris.Wrap(err, "Failed to get project slug")
			}

			// Validate slug
			slug, err = utils.SlugToSaneCheck(slug, minLength, maxLength)
			if err != nil {
				printer.Errorf("%s\n", err)
				printer.NewLine(1)
				continue
			}

			if err := h.checkIfProjectSlugIsTaken(ctx, project, slug); err != nil {
				if eris.Is(err, api.ErrProjectSlugAlreadyExists) {
					printer.Errorf("Project already exists with slug: %s\n", slug)
				} else {
					printer.Errorf("%s\n", err)
				}
				printer.NewLine(1)
				continue
			}
			return nil
		}
	}
}

func (h *Handler) checkIfProjectSlugIsTaken(ctx context.Context, project *models.Project, slug string) error {
	var projectID string
	if project.ID == "" {
		projectID = nilUUID
	} else {
		projectID = project.ID
	}

	err := h.apiClient.CheckProjectSlugIsTaken(ctx, project.OrgID, projectID, slug)
	if err != nil {
		return err
	}
	project.Slug = slug
	return nil
}

//nolint:gocognit // Feel free to refactor this function
func (h *Handler) inputRepoURLAndToken(ctx context.Context, project *models.Project) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			repoURL, err := h.inputService.Prompt(ctx, "Enter Repository URL", project.RepoURL)
			if err != nil {
				return eris.Wrap(err, "Failed to get repository URL")
			}

			// if repoURL prefix is not http or https, add https:// to the repoURL
			if !strings.HasPrefix(repoURL, "http://") && !strings.HasPrefix(repoURL, "https://") {
				repoURL = "https://" + repoURL
			}

			if err := validateRepoURL(repoURL); err != nil {
				continue
			}

			// Try to access the repo with public token
			repoToken := ""
			if err := h.repoClient.ValidateRepoToken(ctx, repoURL, repoToken); err != nil {
				// If the repo is private, we need to get a token
				repoToken, err = h.promptForRepoToken(ctx, project)
				if err != nil {
					return eris.Wrap(err, "Failed to get repository token")
				}
				repoToken = processRepoToken(repoToken, project)

				if err := h.repoClient.ValidateRepoToken(ctx, repoURL, repoToken); err != nil {
					printer.Errorf("%v\n", err)
					printer.NewLine(1)
					continue
				}
			}

			project.RepoURL = repoURL
			project.RepoToken = repoToken
			return nil
		}
	}
}

func validateRepoURL(repoURL string) error {
	if !strings.HasPrefix(repoURL, "http://") && !strings.HasPrefix(repoURL, "https://") {
		printer.NewLine(1)
		printer.Errorln("Invalid Repository URL Format")
		printer.Infoln("The URL must start with:")
		printer.Infoln("• http://")
		printer.Infoln("• https://")
		printer.NewLine(1)
		return eris.New("invalid URL format")
	}
	return nil
}

func (h *Handler) promptForRepoToken(ctx context.Context, project *models.Project) (string, error) {
	if project.Update {
		printer.NewLine(1)
		printer.Headerln("  Update Repository Access Token   ")
		printer.Infoln("Enter new token (options):")
		printer.Infoln("• Press Enter to keep existing token")
		printer.Infoln("• Type 'public' for public repositories")
		printer.Infoln("• Enter new token for private repositories")
	}
	repoToken, err := h.inputService.Prompt(ctx, "\nEnter Token", project.RepoToken)
	if err != nil {
		return "", eris.Wrap(err, "Failed to get repository token")
	}

	return repoToken, nil
}

func processRepoToken(repoToken string, project *models.Project) string {
	// During update, empty input means keep existing token
	if repoToken == "" && project.Update {
		return project.RepoToken
	}
	if strings.ToLower(repoToken) == "public" {
		return ""
	}
	return repoToken
}

func (h *Handler) inputRepoPath(ctx context.Context, project *models.Project) {
	// Get repository Path
	for {
		repoPath, err := h.inputService.Prompt(
			ctx,
			"Enter path to Cardinal within Repo (Empty Valid)",
			project.RepoPath,
		)
		if err != nil {
			printer.Errorf("%v\n", err)
			printer.NewLine(1)
			continue
		}

		// strip off any leading slash
		repoPath = strings.TrimPrefix(repoPath, "/")

		// Validate the path exists using the new validateRepoPath function
		if len(repoPath) > 0 {
			if err := h.repoClient.ValidateRepoPath(ctx, project.RepoURL, project.RepoToken, repoPath); err != nil {
				printer.Errorf("%v\n", err)
				printer.NewLine(1)
				continue
			}
		}

		project.RepoPath = repoPath
		return
	}
}

// configureNotifications handles configuration for both Discord and Slack notifications.
func (h *Handler) configureNotifications(
	ctx context.Context,
	config notificationConfig,
) (bool, string, string, error) {
	enabled, err := h.promptEnableNotifications(ctx, config.name)
	if err != nil {
		return false, "", "", err
	}
	if !enabled {
		return false, "", "", nil
	}

	token, err := h.promptForToken(ctx, config)
	if err != nil {
		return false, "", "", err
	}

	channelID, err := h.promptForChannelID(ctx, config.name)
	if err != nil {
		return false, "", "", err
	}
	return true, token, channelID, nil
}

func (h *Handler) promptEnableNotifications(
	ctx context.Context,
	serviceName string,
) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
		prompt := fmt.Sprintf("Do you want to set up %s notifications? (y/n)", serviceName)
		return h.inputService.Confirm(ctx, prompt, "n")
	}
}

func (h *Handler) promptForToken(
	ctx context.Context,
	config notificationConfig,
) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		prompt := fmt.Sprintf("Enter %s %s", config.name, config.tokenName)
		token, err := h.inputService.Prompt(ctx, prompt, "")
		if err != nil {
			return "", eris.Wrap(err, "Failed to get token")
		}
		return token, nil
	}
}

func (h *Handler) promptForChannelID(ctx context.Context, serviceName string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		prompt := fmt.Sprintf("Enter %s channel ID", serviceName)
		channelID, err := h.inputService.Prompt(ctx, prompt, "")
		if err != nil {
			return "", eris.Wrap(err, "Failed to get channel ID")
		}
		return channelID, nil
	}
}

func (h *Handler) inputDiscord(ctx context.Context, project *models.Project) error {
	enabled, token, channelID, err := h.configureNotifications(ctx, notificationConfig{
		name:      "Discord",
		tokenName: "bot token",
	})
	if err != nil {
		return err
	}

	project.Config.Discord = models.ProjectConfigDiscord{
		Enabled: enabled,
		Token:   token,
		Channel: channelID,
	}
	return nil
}

func (h *Handler) inputSlack(ctx context.Context, project *models.Project) error {
	enabled, token, channelID, err := h.configureNotifications(ctx, notificationConfig{
		name:      "Slack",
		tokenName: "token",
	})
	if err != nil {
		return err
	}

	project.Config.Slack = models.ProjectConfigSlack{
		Enabled: enabled,
		Token:   token,
		Channel: channelID,
	}
	return nil
}

// chooseRegion displays an interactive menu for selecting one or more AWS regions
// using the bubbletea TUI library. Returns error if no regions selected after max attempts
// or context cancellation.
func (h *Handler) chooseRegion(ctx context.Context, project *models.Project, regions []string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			aborted, err := h.runRegionSelector(ctx, project, regions)
			if err != nil {
				printer.Errorln(err.Error())
				printer.NewLine(1)
				if aborted {
					return err
				}
				continue
			}
			if len(project.Config.Region) > 0 {
				return nil
			}
			printer.NewLine(1)
			printer.Errorln("At least one region must be selected")
			printer.Infoln("🔄 Please try again")
		}
	}
}

func (h *Handler) runRegionSelector(ctx context.Context, project *models.Project, regions []string) (bool, error) {
	if regionSelector == nil {
		if project.Update {
			selectedRegions := make(map[int]bool)
			for i, region := range regions {
				if slices.Contains(project.Config.Region, region) {
					selectedRegions[i] = true
				}
			}
			regionSelector = util.NewTeaProgram(multiselect.UpdateMultiselectModel(ctx, regions, selectedRegions))
		} else {
			regionSelector = util.NewTeaProgram(multiselect.InitialMultiselectModel(ctx, regions))
		}
	}
	m, err := regionSelector.Run()
	regionSelector = nil
	if err != nil {
		return false, eris.Wrap(err, "failed to run region selector")
	}

	model, ok := m.(multiselect.Model)
	if !ok {
		return false, eris.New("failed to get selected regions")
	}
	if model.Aborted {
		return true, eris.New("Region selection aborted")
	}

	var selectedRegions []string
	for i, item := range regions {
		if model.Selected[i] {
			selectedRegions = append(selectedRegions, item)
		}
	}

	project.Config.Region = selectedRegions

	return false, nil
}

func (h *Handler) inputAvatarURL(ctx context.Context, project *models.Project) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			avatarURL, err := h.inputService.Prompt(ctx, "Enter avatar URL (Empty Valid)", project.AvatarURL)
			if err != nil {
				return eris.Wrap(err, "Failed to get avatar URL")
			}

			if avatarURL == "" {
				// No avatar URL provided
				project.AvatarURL = ""
				return nil
			}

			if err := utils.IsValidURL(avatarURL); err != nil {
				printer.Errorln(err.Error())
				printer.NewLine(1)
				project.AvatarURL = ""
				continue
			}

			project.AvatarURL = avatarURL
			return nil
		}
	}
}

func displayProjectDetails(project *models.Project) {
	printer.NewLine(1)
	printer.Infoln("Project Details:")
	printer.Infof("• Name: %s\n", project.Name)
	printer.Infof("• Slug: %s\n", project.Slug)
	printer.Infof("• ID: %s\n", project.ID)
	printer.Infof("• Repository URL: %s\n", project.RepoURL)
	printer.Infof("• Repository Path: %s\n", project.RepoPath)
	printer.Infoln("• Regions:")
	for _, region := range project.Config.Region {
		printer.Infof("    - %s\n", region)
	}
	printer.Infoln("• Discord Configuration:")
	if project.Config.Discord.Enabled {
		printer.Infoln("  - Enabled: Yes")
		printer.Infof("  - Channel ID: %s\n", project.Config.Discord.Channel)
		printer.Infof("  - Bot Token: %s\n", project.Config.Discord.Token)
	} else {
		printer.Infoln("  - Enabled: No")
	}
	printer.Infoln("• Slack Configuration:")
	if project.Config.Slack.Enabled {
		printer.Infoln("  - Enabled: Yes")
		printer.Infof("  - Channel ID: %s\n", project.Config.Slack.Channel)
		printer.Infof("  - Token: %s\n", project.Config.Slack.Token)
	} else {
		printer.Infoln("  - Enabled: No")
	}
	printer.Infof("• Avatar URL: %s\n", project.AvatarURL)
}

func printRequiredStepsToCreateProject() {
	printer.NewLine(1)
	printer.Headerln(" To create a new project follow these steps ")
	printer.Infoln("1. Move to the root of your World project.")
	printer.Infoln("   This is the directory that contains world.toml and the cardinal directory")
	printer.Infoln("2. Must be within a git repository")
	printer.Info("3. Use command ")
	printer.Notificationln("'world project create'")
}
