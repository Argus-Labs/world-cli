package project

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/cmd/internal/clients/api"
	"pkg.world.dev/world-cli/cmd/internal/models"
	"pkg.world.dev/world-cli/cmd/internal/utils/slug"
	"pkg.world.dev/world-cli/cmd/internal/utils/validate"
	"pkg.world.dev/world-cli/common/printer"
	"pkg.world.dev/world-cli/common/tomlutil"
)

const MaxProjectNameLen = 50

var (
	ErrCannotCreateSwitchProject = eris.New("Cannot create/switch Project, directory belongs to another project.")
)

func (h *Handler) Create(
	ctx context.Context,
	org models.Organization,
	flags models.CreateProjectFlags,
) (models.Project, error) {
	if h.configService.GetConfig().CurrRepoKnown {
		printer.Errorf("Cannot create Project, current git working directory belongs to project: %s.",
			h.configService.GetConfig().CurrProjectName)
		return models.Project{}, ErrCannotCreateSwitchProject
	}

	repoPath, repoURL, err := h.PreCreateUpdateValidation(true)
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
		Name:     flags.Name,
		Slug:     flags.Slug,
		OrgID:    org.ID,
		RepoPath: repoPath,
		RepoURL:  repoURL,
		Update:   false,
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

func (h *Handler) getSetupInput(
	ctx context.Context,
	project *models.Project,
	regions []string,
) error {
	err := h.inputName(ctx, project)
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
	selectedRegions, err := h.regionSelector.SelectRegions(ctx, regions, project.Config.Region)
	if err != nil {
		return eris.Wrap(err, "Failed to choose region")
	}
	project.Config.Region = selectedRegions

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

	return nil
}

func (h *Handler) inputName(ctx context.Context, project *models.Project) error {
	for {
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
	if err := validate.Name(name, MaxProjectNameLen); err != nil {
		return err
	}
	project.Name = name
	return nil
}

func (h *Handler) inputSlug(ctx context.Context, project *models.Project) error {
	for {
		minLength := 3
		maxLength := 25
		if project.Slug == "" {
			// if no slug exists, create a default one from the name
			project.Slug = slug.CreateFromName(project.Name, minLength, maxLength)
		} else {
			// if a slug exists, validate it
			project.Slug = slug.CreateFromName(project.Slug, minLength, maxLength)
		}

		slugInput, err := h.inputService.Prompt(ctx, "Slug", project.Slug)
		if err != nil {
			return eris.Wrap(err, "Failed to get project slug")
		}

		slugInput, err = slug.ToSaneCheck(slugInput, minLength, maxLength)
		if err != nil {
			printer.Errorf("%s\n", err)
			printer.NewLine(1)
			continue
		}

		if err := h.checkIfProjectSlugIsTaken(ctx, project, slugInput); err != nil {
			if eris.Is(err, api.ErrProjectSlugAlreadyExists) {
				printer.Errorf("Project already exists with slug: %s\n", slugInput)
			} else {
				printer.Errorf("%s\n", err)
			}
			printer.NewLine(1)
			continue
		}
		return nil
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

func (h *Handler) inputRepoURLAndToken(ctx context.Context, project *models.Project) error {
	for {
		repoURL, err := h.inputService.Prompt(ctx, "Enter Repository URL", project.RepoURL)
		if err != nil {
			return eris.Wrap(err, "Failed to get repository URL")
		}

		// if repoURL prefix is not http or https, add https:// to the repoURL
		if !strings.HasPrefix(repoURL, "http://") && !strings.HasPrefix(repoURL, "https://") {
			repoURL = "https://" + repoURL
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
		prompt := "Enter path to Cardinal within Repo (Empty Valid)"
		repoPath, err := h.inputService.Prompt(ctx, prompt, project.RepoPath)
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
	// Get notification confirmation
	prompt := fmt.Sprintf("Do you want to set up %s notifications? (y/n)", config.name)
	enabled, err := h.inputService.Confirm(ctx, prompt, "n")
	if err != nil {
		return false, "", "", eris.Wrap(err, "Failed to get notification confirmation")
	}
	if !enabled {
		return false, "", "", nil
	}

	// Get token
	prompt = fmt.Sprintf("Enter %s %s", config.name, config.tokenName)
	token, err := h.inputService.Prompt(ctx, prompt, "")
	if err != nil {
		return false, "", "", eris.Wrap(err, "Failed to get token")
	}

	// Get channel ID
	prompt = fmt.Sprintf("Enter %s channel ID", config.name)
	channelID, err := h.inputService.Prompt(ctx, prompt, "")
	if err != nil {
		return false, "", "", eris.Wrap(err, "Failed to get channel ID")
	}
	return true, token, channelID, nil
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
}
