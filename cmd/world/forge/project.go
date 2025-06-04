package forge

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/common/logger"
	"pkg.world.dev/world-cli/common/printer"
	"pkg.world.dev/world-cli/common/tomlutil"
	"pkg.world.dev/world-cli/tea/component/multiselect"
)

const MaxProjectNameLen = 50

var regionSelector *tea.Program

// ErrProjectSlugAlreadyExists is passed from forge to world-cli, Must always match.
var ErrProjectSlugAlreadyExists = eris.New("project slug already exists")

type project struct {
	ID           string        `json:"id"`
	OrgID        string        `json:"org_id"`
	OwnerID      string        `json:"owner_id"`
	Name         string        `json:"name"`
	Slug         string        `json:"slug"`
	CreatedTime  string        `json:"created_time"`
	UpdatedTime  string        `json:"updated_time"`
	Deleted      bool          `json:"deleted"`
	DeletedTime  string        `json:"deleted_time"`
	RepoURL      string        `json:"repo_url"`
	RepoToken    string        `json:"repo_token"`
	RepoPath     string        `json:"repo_path"`
	DeploySecret string        `json:"deploy_secret,omitempty"`
	Config       projectConfig `json:"config"`
	AvatarURL    string        `json:"avatar_url"`

	update bool `json:"-"`
}

type projectConfig struct {
	TickRate int                  `json:"tick_rate"`
	Region   []string             `json:"region"`
	Discord  projectConfigDiscord `json:"discord"`
	Slack    projectConfigSlack   `json:"slack"`
}

type projectConfigDiscord struct {
	Enabled bool   `json:"enabled"`
	Token   string `json:"token"`
	Channel string `json:"channel"`
}

type projectConfigSlack struct {
	Enabled bool   `json:"enabled"`
	Token   string `json:"token"`
	Channel string `json:"channel"`
}

// notificationConfig holds common notification configuration fields.
type notificationConfig struct {
	name      string // "Discord" or "Slack"
	tokenName string // What to call the token ("bot token" or "token")
}

// Show list of projects in selected organization.
func showProjectList(fCtx ForgeContext) error {
	projects, err := getListOfProjects(fCtx)
	if err != nil {
		return eris.Wrap(err, "Failed to get projects")
	}

	if len(projects) == 0 {
		printNoProjectsInOrganization()
		return nil
	}

	selectedProject, err := getSelectedProject(fCtx)
	if err != nil {
		return eris.Wrap(err, "Failed to get selected project")
	}

	printer.NewLine(1)
	printer.Headerln("   Project Information   ")
	if selectedProject.Name == "" {
		printer.Errorln("No project selected")
		printer.NewLine(1)
		printer.Infoln("Use 'world forge project switch' to choose a project")
	} else {
		for _, prj := range projects {
			if prj.ID == selectedProject.ID {
				printer.Infof("• %s (%s) [SELECTED]\n", prj.Name, prj.Slug)
			} else {
				printer.Infof("  %s (%s)\n", prj.Name, prj.Slug)
			}
		}
	}

	return nil
}

// Get selected project.
func getSelectedProject(fCtx ForgeContext) (project, error) {
	selectedOrg, err := getSelectedOrganization(fCtx)
	if err != nil {
		return project{}, eris.Wrap(err, "Failed to get organization")
	}

	if selectedOrg.ID == "" {
		printNoSelectedOrganization()
		return project{}, nil
	}

	if fCtx.Config.ProjectID == "" {
		projects, err := getListOfProjects(fCtx)
		if err != nil {
			return project{}, eris.Wrap(err, "Failed to get projects")
		}
		if len(projects) == 0 {
			printNoProjectsInOrganization()
		}
		return project{}, nil
	}

	// Send request
	projectURL := fmt.Sprintf(projectURLPattern, baseURL, selectedOrg.ID) + "/" + fCtx.Config.ProjectID
	body, err := sendRequest(fCtx, http.MethodGet, projectURL, nil)
	if err != nil {
		return project{}, eris.Wrap(err, "Failed to get project")
	}

	// Parse response
	prj, err := parseResponse[project](body)
	if err != nil {
		return project{}, eris.Wrap(err, "Failed to parse project")
	}

	return *prj, nil
}

// Get list of projects in selected organization.
func getListOfProjects(fCtx ForgeContext) ([]project, error) {
	selectedOrg, err := getSelectedOrganization(fCtx)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get organization")
	}

	if selectedOrg.ID == "" {
		printNoSelectedOrganization()
		return nil, nil
	}

	url := fmt.Sprintf(projectURLPattern, baseURL, selectedOrg.ID)
	body, err := sendRequest(fCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get projects")
	}

	projects, err := parseResponse[[]project](body)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to parse projects")
	}

	return *projects, nil
}

func getListRegions(fCtx ForgeContext, orgID, projID string) ([]string, error) {
	url := fmt.Sprintf(projectURLPattern+"/%s/regions", baseURL, orgID, projID)
	body, err := sendRequest(fCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get regions")
	}

	regionMap, err := parseResponse[map[string]string](body)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to parse regions")
	}

	regions := make([]string, 0, len(*regionMap))
	for _, region := range *regionMap {
		regions = append(regions, region)
	}

	sort.Strings(regions)
	return regions, nil
}

// Get list of projects in selected organization.
func getListOfAvailableRegionsForNewProject(fCtx ForgeContext) ([]string, error) {
	selectedOrg, err := getSelectedOrganization(fCtx)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get organization")
	}
	if selectedOrg.ID == "" {
		printNoSelectedOrganization()
		return nil, nil
	}
	return getListRegions(fCtx, selectedOrg.ID, nilUUID)
}

// Get list of projects in selected organization.
func (p *project) getListOfAvailableRegions(fCtx ForgeContext) ([]string, error) {
	if p.ID == "" || p.OrgID == "" {
		printNoSelectedProject()
		return nil, nil
	}
	return getListRegions(fCtx, p.OrgID, p.ID)
}

// projectPreCreateUpdateValidation returns the repo path and URL, and an error.
func projectPreCreateUpdateValidation() (string, string, error) {
	repoPath, repoURL, err := FindGitPathAndURL()
	if err != nil && !strings.Contains(err.Error(), ErrNotInGitRepository.Error()) {
		return repoPath, repoURL, eris.Wrap(err, "Failed to find git path and URL")
	} else if repoURL == "" { // Path is ok as empty, This means it's in repo root.
		printer.Errorln(" Not in a git repository")
		return repoPath, repoURL, ErrNotInGitRepository
	}

	inRoot, err := isInWorldCadinalRoot()
	if err != nil {
		return repoPath, repoURL, eris.Wrap(err, "Failed to check if in World project root")
	} else if !inRoot {
		printer.Errorln(" Not in a World project root")
		return repoPath, repoURL, eris.New("Not in a World project root")
	}

	return repoPath, repoURL, nil
}

func createProject(fCtx ForgeContext, flags *CreateProjectCmd) (*project, error) {
	if fCtx.Config.CurrRepoKnown {
		printer.Errorf("Cannot create Project, current git working directory belongs to project: %s.",
			fCtx.Config.CurrProjectName)
		return nil, eris.New("Cannot create Project, directory belongs to another project.")
	}

	repoPath, repoURL, err := projectPreCreateUpdateValidation()
	if err != nil {
		printRequiredStepsToCreateProject()
		return nil, eris.Wrap(err, "Failed to validate project creation")
	}

	regions, err := getListOfAvailableRegionsForNewProject(fCtx)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get available regions")
	}

	printer.NewLine(1)
	printer.Headerln("   Project Creation   ")

	p := project{
		Name:      flags.Name,
		Slug:      flags.Slug,
		AvatarURL: flags.AvatarURL,
		RepoPath:  repoPath,
		RepoURL:   repoURL,
		update:    false,
	}
	err = p.getSetupInput(fCtx, regions)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get project input")
	}

	// Send request
	url := fmt.Sprintf(projectURLPattern, baseURL, p.OrgID)
	body, err := sendRequest(fCtx, http.MethodPost, url, map[string]interface{}{
		"name":       p.Name,
		"slug":       p.Slug,
		"repo_url":   p.RepoURL,
		"repo_token": p.RepoToken,
		"repo_path":  p.RepoPath,
		"org_id":     p.OrgID,
		"config":     p.Config,
		"avatar_url": p.AvatarURL,
	})
	if err != nil {
		if eris.Is(err, ErrProjectSlugAlreadyExists) {
			printer.Errorf("Project already exists with slug: %s, please choose a different slug.\n", p.Slug)
			printer.NewLine(1)
		}
		return nil, eris.Wrap(err, "Failed to create project")
	}

	prj, err := parseResponse[project](body)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to parse response")
	}

	// Select project
	prj.saveToConfig(fCtx)

	prj.displayProjectDetails()
	printer.Infof("• Deploy Secret (for deploy via CI/CD pipeline tools): ")
	printer.Infof("%s\n", prj.DeploySecret)
	printer.Notificationln("Note: Deploy Secret will not be shown again. Save it now in a secure location.")

	printer.NewLine(1)
	printer.Successf("Created project: %s [%s]\n", prj.Name, prj.Slug)
	return prj, nil
}

func (p *project) displayProjectDetails() {
	printer.NewLine(1)
	printer.Infoln("Project Details:")
	printer.Infof("• Name: %s\n", p.Name)
	printer.Infof("• Slug: %s\n", p.Slug)
	printer.Infof("• ID: %s\n", p.ID)
	printer.Infof("• Repository URL: %s\n", p.RepoURL)
	printer.Infof("• Repository Path: %s\n", p.RepoPath)
	printer.Infof("• Tick Rate: %d\n", p.Config.TickRate)
	printer.Infoln("• Regions:")
	for _, region := range p.Config.Region {
		printer.Infof("    - %s\n", region)
	}
	printer.Infoln("• Discord Configuration:")
	if p.Config.Discord.Enabled {
		printer.Infoln("  - Enabled: Yes")
		printer.Infof("  - Channel ID: %s\n", p.Config.Discord.Channel)
		printer.Infof("  - Bot Token: %s\n", p.Config.Discord.Token)
	} else {
		printer.Infoln("  - Enabled: No")
	}
	printer.Infoln("• Slack Configuration:")
	if p.Config.Slack.Enabled {
		printer.Infoln("  - Enabled: Yes")
		printer.Infof("  - Channel ID: %s\n", p.Config.Slack.Channel)
		printer.Infof("  - Token: %s\n", p.Config.Slack.Token)
	} else {
		printer.Infoln("  - Enabled: No")
	}
	printer.Infof("• Avatar URL: %s\n", p.AvatarURL)
}

func (p *project) inputName(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		// If name is not set from cmd flags, get it from world.toml
		if p.Name == "" {
			// Get project name from world.toml if it exists, fails silently
			err := p.getForgeProjectNameFromWorldToml()
			if err != nil {
				p.Name = ""
			}
		}

		name := getInput("Enter project name", p.Name)

		err := p.validateAndSetName(name)
		if err == nil {
			return nil
		}
		// If validation fails, clear the name to attempt from toml
		p.Name = ""
	}
}

func (p *project) getForgeProjectNameFromWorldToml() error {
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

	if err := p.validateAndSetName(projectName); err != nil {
		return eris.Wrap(err, "invalid project name in world.toml")
	}
	return nil
}

func (p *project) validateAndSetName(name string) error {
	if name == "" {
		printer.Errorln("Error: Project name cannot be empty")
		printer.NewLine(1)
		return eris.New("empty name")
	}

	if len(name) > MaxProjectNameLen {
		printer.Errorf("Error: Project name cannot be longer than %d characters\n", MaxProjectNameLen)
		printer.NewLine(1)
		return eris.New("name too long")
	}

	if strings.ContainsAny(name, "<>:\"/\\|?*") {
		printer.Errorln("Error: Project name contains invalid characters" +
			"   Invalid characters: < > : \" / \\ | ? *")
		printer.NewLine(1)
		return eris.New("invalid characters")
	}

	p.Name = name
	return nil
}

func (p *project) inputSlug(fCtx ForgeContext) error {
	for {
		select {
		case <-fCtx.Context.Done():
			return fCtx.Context.Err()
		default:
			// if no slug exists, create a default one from the name
			minLength := 3
			maxLength := 25
			if p.Slug == "" {
				p.Slug = CreateSlugFromName(p.Name, minLength, maxLength)
			} else {
				p.Slug = CreateSlugFromName(p.Slug, minLength, maxLength)
			}

			slug := getInput("Slug", p.Slug)

			// Validate slug
			var err error
			slug, err = slugToSaneCheck(slug, minLength, maxLength)
			if err != nil {
				printer.Errorf("%s\n", err)
				printer.NewLine(1)
				continue
			}

			if err := p.checkIfProjectSlugIsTaken(fCtx, slug); err != nil {
				if eris.Is(err, ErrProjectSlugAlreadyExists) {
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

func (p *project) checkIfProjectSlugIsTaken(fCtx ForgeContext, slug string) error {
	var projectID string
	if p.ID == "" {
		projectID = nilUUID
	} else {
		projectID = p.ID
	}

	url := fmt.Sprintf(projectURLPattern+"/%s/%s/check_slug", baseURL, p.OrgID, projectID, slug)
	_, err := sendRequest(fCtx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	p.Slug = slug
	return nil
}

func (p *project) inputRepoURLAndToken(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			repoURL := getInput("Enter Repository URL", p.RepoURL)

			// if repoURL prefix is not http or https, add https:// to the repoURL
			if !strings.HasPrefix(repoURL, "http://") && !strings.HasPrefix(repoURL, "https://") {
				repoURL = "https://" + repoURL
			}

			if err := p.validateRepoURL(repoURL); err != nil {
				continue
			}

			// Try to access the repo with public token
			repoToken := ""
			if err := validateRepoToken(ctx, repoURL, repoToken); err != nil {
				// If the repo is private, we need to get a token
				repoToken = p.promptForRepoToken()
				repoToken = p.processRepoToken(repoToken)

				if err := validateRepoToken(ctx, repoURL, repoToken); err != nil {
					printer.Errorf("%v\n", err)
					printer.NewLine(1)
					continue
				}
			}

			p.RepoURL = repoURL
			p.RepoToken = repoToken
			return nil
		}
	}
}

func (p *project) validateRepoURL(repoURL string) error {
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

func (p *project) promptForRepoToken() string {
	if p.update {
		printer.NewLine(1)
		printer.Headerln("  Update Repository Access Token   ")
		printer.Infoln("Enter new token (options):")
		printer.Infoln("• Press Enter to keep existing token")
		printer.Infoln("• Type 'public' for public repositories")
		printer.Infoln("• Enter new token for private repositories")
	}
	repoToken := getInput("\nEnter Token", p.RepoToken)

	return repoToken
}

func (p *project) processRepoToken(repoToken string) string {
	// During update, empty input means keep existing token
	if repoToken == "" && p.update {
		return p.RepoToken
	}
	if strings.ToLower(repoToken) == "public" {
		return ""
	}
	return repoToken
}

func (p *project) inputRepoPath(ctx context.Context) {
	// Get repository Path
	var repoPath string
	for {
		repoPath = getInput("Enter path to Cardinal within Repo (Empty Valid)", p.RepoPath)

		// strip off any leading slash
		repoPath = strings.TrimPrefix(repoPath, "/")

		// Validate the path exists using the new validateRepoPath function
		if len(repoPath) > 0 {
			if err := validateRepoPath(ctx, p.RepoURL, p.RepoToken, repoPath); err != nil {
				printer.Errorf("%v\n", err)
				printer.NewLine(1)
				continue
			}
		}

		p.RepoPath = repoPath
		return
	}
}

func selectProjectBySlug(fCtx ForgeContext, projects []project, slug string) (*project, error) {
	for _, project := range projects {
		if project.Slug == slug {
			err := project.saveToConfig(fCtx)
			if err != nil {
				return nil, eris.Wrap(err, "selectProjectBySlug")
			}
			showProjectList(fCtx)
			return &project, nil
		}
	}
	showProjectList(fCtx)
	printer.NewLine(1)
	printer.Errorln("Project not found in organization under the slug: " + slug)
	return nil, ErrProjectSelectionCanceled
}

func selectProject(fCtx ForgeContext, flags *SwitchProjectCmd) (*project, error) {
	if fCtx.Config.CurrRepoKnown {
		printer.Errorf("Cannot switch Project, current git working directory belongs to project: %s.",
			fCtx.Config.CurrProjectName)
		return nil, eris.New("Cannot switch Project, directory belongs to another project.")
	}

	// Get projects from selected organization
	projects, err := getListOfProjects(fCtx)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get projects")
	}

	if len(projects) == 0 {
		printNoProjectsInOrganization()
		return nil, nil //nolint: nilnil // bad linter! sentinel errors are slow
	}

	// If slug is provided, select the project by slug
	if flags.Slug != "" {
		return selectProjectBySlug(fCtx, projects, flags.Slug)
	}

	// Display projects as a numbered list
	printer.NewLine(1)
	printer.Headerln("   Available Projects   ")
	for i, proj := range projects {
		printer.Infof("  %d. %s\n     └─ Slug: %s\n", i+1, proj.Name, proj.Slug)
	}

	// Get user input
	for {
		input := getInput("\nEnter project number (or 'q' to quit)", "")
		input = strings.TrimSpace(input)
		if input == "q" {
			return nil, nil //nolint: nilnil // bad linter! sentinel errors are slow
		}

		// Parse selection
		num, err := strconv.Atoi(input)
		if err != nil || num < 1 || num > len(projects) {
			printer.Errorf("Please enter a number between 1 and %d\n", len(projects))
			continue
		}

		selectedProject := projects[num-1]

		err = selectedProject.saveToConfig(fCtx)
		if err != nil {
			return nil, eris.Wrap(err, "selectProject")
		}

		printer.NewLine(1)
		printer.Successf("Switched to project: %s\n", selectedProject.Name)
		return &selectedProject, nil
	}
}

func getProjectDataByID(fCtx ForgeContext, id string) (project, error) {
	projects, err := getListOfProjects(fCtx)
	if err != nil {
		return project{}, eris.Wrap(err, "Failed to get projects")
	}

	if len(projects) == 0 {
		return project{}, eris.New("No projects found")
	}

	for _, project := range projects {
		if project.ID == id {
			return project, nil
		}
	}
	return project{}, eris.New("Project not found with ID: " + id)
}

func (p *project) delete(fCtx ForgeContext) error {
	// Print project details with fancy formatting
	printer.NewLine(1)
	printer.Headerln("   Project Deletion   ")
	printer.Infoln("Project Details:")
	printer.Infof("• Name: %s\n", p.Name)
	printer.Infof("• Slug: %s\n", p.Slug)

	// Warning message with fancy formatting
	printer.NewLine(1)
	printer.Headerln("  ⚠️WARNING!⚠️  ")
	printer.Infoln("This action will permanently delete:")
	printer.Infoln("• All deployments")
	printer.Infoln("• All logs")
	printer.Infoln("• All associated resources")
	printer.NewLine(1)

	printer.Info("Type 'Yes' to confirm deletion of project ")
	printer.Notificationf("'%s'", p.Name)
	confirmation := getInput("", "")

	if confirmation != "Yes" {
		if confirmation == "yes" {
			printer.Errorln("You must type 'Yes' with uppercase Y to confirm deletion")
		}
		printer.Errorln("Project deletion canceled")
		return nil
	}

	// Send request
	url := fmt.Sprintf(projectURLPattern, baseURL, p.OrgID) + "/" + p.ID
	body, err := sendRequest(fCtx, http.MethodDelete, url, nil)
	if err != nil {
		return eris.Wrap(err, "Failed to delete project")
	}

	p.RemoveKnownProject(fCtx.Config)

	// Parse response
	_, err = parseResponse[any](body)
	if err != nil {
		return eris.Wrap(err, "Failed to parse response")
	}

	printer.Successf("Project deleted: %s (%s)\n", p.Name, p.Slug)

	// Remove project from config
	(&project{ID: "", Name: ""}).saveToConfig(fCtx)

	return nil
}

func (p *project) updateProject(fCtx ForgeContext, flags *UpdateProjectCmd) error {
	if fCtx.State.Project.Name == "" || fCtx.State.Project.Slug == "" {
		return eris.New("Forge setup failed, no project selected")
	}
	printer.Infof("Updating Project: %s [%s]\n", fCtx.State.Project.Name, fCtx.State.Project.Slug)

	repoPath, repoURL, err := projectPreCreateUpdateValidation()
	if err != nil {
		printRequiredStepsToCreateProject()
		return eris.Wrap(err, "Failed to validate project update")
	}

	regions, err := p.getListOfAvailableRegions(fCtx)
	if err != nil {
		return eris.Wrap(err, "Failed to get available regions")
	}

	// set update to true
	p.update = true
	p.Name = flags.Name
	p.Slug = flags.Slug
	p.AvatarURL = flags.AvatarURL
	p.RepoPath = repoPath
	p.RepoURL = repoURL
	p.ID = fCtx.State.Project.ID

	printer.NewLine(1)
	printer.Headerln("  Project Update  ")

	// get project input
	err = p.getSetupInput(fCtx, regions)
	if err != nil {
		return eris.Wrap(err, "Failed to get project input")
	}

	printer.NewLine(1)
	printer.Infoln("Updating project...")

	// Send request
	url := fmt.Sprintf(projectURLPattern, baseURL, p.OrgID) + "/" + p.ID
	body, err := sendRequest(fCtx, http.MethodPut, url, map[string]interface{}{
		"name":       p.Name,
		"slug":       p.Slug,
		"repo_url":   p.RepoURL,
		"repo_token": p.RepoToken,
		"repo_path":  p.RepoPath,
		"config":     p.Config,
		"avatar_url": p.AvatarURL,
	})
	if err != nil {
		if eris.Is(err, ErrProjectSlugAlreadyExists) {
			printer.Errorf("Project already exists with slug: %s, please choose a different slug.\n", p.Slug)
			printer.NewLine(1)
		}
		return eris.Wrap(err, "Failed to update project")
	}

	p.RemoveKnownProject(fCtx.Config)

	_, err = parseResponse[any](body)
	if err != nil {
		return eris.Wrap(err, "Failed to parse response")
	}

	p.displayProjectDetails()
	printer.NewLine(1)
	printer.Successf("Project '%s [%s]' updated successfully!\n", p.Name, p.Slug)
	return nil
}

func (p *project) getSetupInput(fCtx ForgeContext, regions []string) error {
	// Get organization
	org, err := getSelectedOrganization(fCtx)
	if err != nil {
		return eris.Wrap(err, "Failed to get organization")
	}

	if org.ID == "" {
		printNoSelectedOrganization()
		return nil
	}

	p.OrgID = org.ID

	err = p.inputName(fCtx.Context)
	if err != nil {
		return eris.Wrap(err, "Failed to get project name")
	}

	err = p.inputSlug(fCtx)
	if err != nil {
		return eris.Wrap(err, "Failed to get project slug")
	}

	err = p.inputRepoURLAndToken(fCtx.Context)
	if err != nil {
		return eris.Wrap(err, "Failed to get repository URL and token")
	}

	p.inputRepoPath(fCtx.Context)

	// Tick Rate
	err = p.inputTickRate(fCtx.Context)
	if err != nil {
		return eris.Wrap(err, "Failed to get environment name")
	}

	// Regions
	err = p.chooseRegion(fCtx.Context, regions)
	if err != nil {
		return eris.Wrap(err, "Failed to choose region")
	}

	// Discord
	err = p.inputDiscord(fCtx.Context)
	if err != nil {
		return eris.Wrap(err, "Failed to input discord")
	}

	// Slack
	err = p.inputSlack(fCtx.Context)
	if err != nil {
		return eris.Wrap(err, "Failed to input slack")
	}

	err = p.inputAvatarURL(fCtx.Context)
	if err != nil {
		return eris.Wrap(err, "Failed to input avatar URL")
	}

	return nil
}

// inputTickRate prompts the user to enter a tick rate value (default is 1)
// and validates that it is a valid number. Returns error after max attempts or context cancellation.
func (p *project) inputTickRate(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			var defaultValStr string
			if p.Config.TickRate != 0 {
				printer.Infof("Current tick rate: %d\n", p.Config.TickRate)
				defaultValStr = strconv.Itoa(p.Config.TickRate)
			} else {
				printer.Infoln("Enter tick rate for your project")
				defaultValStr = "1"
			}

			tickRateStr := getInput("  └─ Examples: 10, 20, 30", defaultValStr)

			newTickRate, err := strconv.Atoi(tickRateStr)
			p.Config.TickRate = newTickRate
			if p.Config.TickRate <= 0 || err != nil {
				printer.Errorln("Invalid input. Please enter a non-zero positive number")
				printer.NewLine(1)
				continue
			}

			return nil
		}
	}
}

// configureNotifications handles configuration for both Discord and Slack notifications.
func (p *project) configureNotifications(ctx context.Context, config notificationConfig) (bool, string, string, error) {
	enabled, err := p.promptEnableNotifications(ctx, config.name)
	if err != nil {
		return false, "", "", err
	}
	if !enabled {
		return false, "", "", nil
	}

	token, err := p.promptForToken(ctx, config)
	if err != nil {
		return false, "", "", err
	}

	channelID, err := p.promptForChannelID(ctx, config.name)
	if err != nil {
		return false, "", "", err
	}
	return true, token, channelID, nil
}

func (p *project) promptEnableNotifications(ctx context.Context, serviceName string) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
		for {
			prompt := fmt.Sprintf("Do you want to set up %s notifications? (y/n)", serviceName)

			confirmation := getInput(prompt, "n")

			switch strings.ToLower(confirmation) {
			case "y":
				return true, nil
			case "n":
				return false, nil
			default:
				printer.Errorf("Invalid input. Please enter 'y' or 'n'")
				printer.NewLine(1)
			}
		}
	}
}

func (p *project) promptForToken(ctx context.Context, config notificationConfig) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		prompt := fmt.Sprintf("Enter %s %s", config.name, config.tokenName)
		token := getInput(prompt, "")
		return token, nil
	}
}

func (p *project) promptForChannelID(ctx context.Context, serviceName string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		prompt := fmt.Sprintf("Enter %s channel ID", serviceName)
		channelID := getInput(prompt, "")
		return channelID, nil
	}
}

func (p *project) inputDiscord(ctx context.Context) error {
	enabled, token, channelID, err := p.configureNotifications(ctx, notificationConfig{
		name:      "Discord",
		tokenName: "bot token",
	})
	if err != nil {
		return err
	}

	p.Config.Discord = projectConfigDiscord{
		Enabled: enabled,
		Token:   token,
		Channel: channelID,
	}
	return nil
}

func (p *project) inputSlack(ctx context.Context) error {
	enabled, token, channelID, err := p.configureNotifications(ctx, notificationConfig{
		name:      "Slack",
		tokenName: "token",
	})
	if err != nil {
		return err
	}

	p.Config.Slack = projectConfigSlack{
		Enabled: enabled,
		Token:   token,
		Channel: channelID,
	}
	return nil
}

// chooseRegion displays an interactive menu for selecting one or more AWS regions
// using the bubbletea TUI library. Returns error if no regions selected after max attempts
// or context cancellation.
func (p *project) chooseRegion(ctx context.Context, regions []string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			aborted, err := p.runRegionSelector(ctx, regions)
			if err != nil {
				printer.Errorln(err.Error())
				printer.NewLine(1)
				if aborted {
					return err
				}
				continue
			}
			if len(p.Config.Region) > 0 {
				return nil
			}
			printer.NewLine(1)
			printer.Errorln("At least one region must be selected")
			printer.Infoln("🔄 Please try again")
		}
	}
}

func (p *project) runRegionSelector(ctx context.Context, regions []string) (bool, error) {
	if regionSelector == nil {
		if p.update {
			selectedRegions := make(map[int]bool)
			for i, region := range regions {
				if slices.Contains(p.Config.Region, region) {
					selectedRegions[i] = true
				}
			}
			regionSelector = NewTeaProgram(multiselect.UpdateMultiselectModel(ctx, regions, selectedRegions))
		} else {
			regionSelector = NewTeaProgram(multiselect.InitialMultiselectModel(ctx, regions))
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

	p.Config.Region = selectedRegions

	return false, nil
}

func (p *project) saveToConfig(fCtx ForgeContext) error {
	fCtx.Config.ProjectID = p.ID
	err := fCtx.Config.Save()
	if err != nil {
		return eris.Wrap(err, "Failed to save project configuration")
	}
	return nil
}

// handleProjectSelection manages the project selection logic.
func handleProjectSelection(fCtx ForgeContext) error {
	projects, err := getListOfProjects(fCtx)
	if err != nil {
		return eris.Wrap(err, "Failed to get projects")
	}

	switch numProjects := len(projects); {
	case numProjects == 1:
		return projects[0].handleSingleProject(fCtx)
	case numProjects > 1:
		return handleMultipleProjects(fCtx, projects)
	default:
		return handleNoProjects(fCtx)
	}
}

func (p *project) handleSingleProject(fCtx ForgeContext) error {
	p.saveToConfig(fCtx)
	showProjectList(fCtx)
	return nil
}

// handleMultipleProjects handles the case when there are multiple projects.
func handleMultipleProjects(fCtx ForgeContext, projects []project) error {
	for _, project := range projects {
		if project.ID == fCtx.Config.ProjectID {
			showProjectList(fCtx)
			return nil
		}
	}

	project, err := selectProject(fCtx, &SwitchProjectCmd{})
	if err != nil {
		return eris.Wrap(err, "Failed to select project")
	}
	if project == nil {
		return nil
	}

	project.saveToConfig(fCtx)
	return nil
}

// handleNoProjects handles the case when there are no projects.
func handleNoProjects(fCtx ForgeContext) error {
	// Confirmation prompt
	printNoProjectsInOrganization()
	printer.NewLine(1)
	confirmation := getInput("Do you want to create a new project now? (y/n)", "y")

	if strings.ToLower(confirmation) != "y" {
		printer.Errorln("Project creation canceled")
		return nil
	}

	project, err := createProject(fCtx, &CreateProjectCmd{})
	if err != nil {
		return eris.Wrap(err, "Failed to create project")
	}

	project.saveToConfig(fCtx)
	return nil
}

func (p *project) inputAvatarURL(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			avatarURL := getInput("Enter avatar URL (Empty Valid)", p.AvatarURL)

			if avatarURL == "" {
				// No avatar URL provided
				p.AvatarURL = ""
				return nil
			}

			if err := isValidURL(avatarURL); err != nil {
				printer.Errorln(err.Error())
				printer.NewLine(1)
				p.AvatarURL = ""
				continue
			}

			p.AvatarURL = avatarURL
			return nil
		}
	}
}

func (p *project) AddKnownProject(config *Config) {
	config.KnownProjects = append(config.KnownProjects, KnownProject{
		ProjectID:      p.ID,
		OrganizationID: p.OrgID,
		RepoURL:        p.RepoURL,
		RepoPath:       p.RepoPath,
		ProjectName:    p.Name,
	})

	err := config.Save()
	if err != nil {
		printer.Notificationf("Warning: Failed to save config: %s", err)
		logger.Error(eris.Wrap(err, "AddKnownProject failed to save config"))
		// continue on, this is not fatal
	}
}

func (p *project) RemoveKnownProject(config *Config) error {
	newKnownProjects := make([]KnownProject, 0)

	for _, knownProj := range config.KnownProjects {
		if knownProj.OrganizationID != p.OrgID || knownProj.ProjectID != p.ID {
			newKnownProjects = append(newKnownProjects, knownProj)
		}
	}

	config.KnownProjects = newKnownProjects

	err := config.Save()
	if err != nil {
		printer.Notificationf("Warning: RemoveKnownProject failed to save config: %s", err)
		logger.Error(eris.Wrap(err, "RemoveKnownProject failed to save config"))
		return ErrCannotSaveConfig
	}
	return nil
}
