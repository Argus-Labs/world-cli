package forge

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rotisserie/eris"

	"pkg.world.dev/world-cli/common/globalconfig"
	"pkg.world.dev/world-cli/tea/component/multiselect"
)

const MaxProjectNameLen = 50

var regionSelector *tea.Program

type project struct {
	ID          string        `json:"id"`
	OrgID       string        `json:"org_id"`
	OwnerID     string        `json:"owner_id"`
	Name        string        `json:"name"`
	Slug        string        `json:"slug"`
	CreatedTime string        `json:"created_time"`
	UpdatedTime string        `json:"updated_time"`
	Deleted     bool          `json:"deleted"`
	DeletedTime string        `json:"deleted_time"`
	RepoURL     string        `json:"repo_url"`
	RepoToken   string        `json:"repo_token"`
	RepoPath    string        `json:"repo_path"`
	Config      projectConfig `json:"config"`

	update bool `json:"-"`
}

type projectConfig struct {
	TickRate int      `json:"tick_rate"`
	Region   []string `json:"region"`
}

// Show list of projects in selected organization
func showProjectList(ctx context.Context) error {
	projects, err := getListOfProjects(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get projects")
	}

	if len(projects) == 0 {
		printNoProjectsInOrganization()
		return nil
	}

	project, err := getSelectedProject(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get selected project")
	}

	fmt.Println("\n📁 Project Information")
	fmt.Println("--------------------")
	if project.Name == "" {
		fmt.Println("No project selected")
	} else {
		fmt.Println("\nAvailable Projects:")
		for _, prj := range projects {
			if prj.ID == project.ID {
				fmt.Printf("* %s (%s) [SELECTED]\n", prj.Name, prj.Slug)
			} else {
				fmt.Printf("  %s (%s)\n", prj.Name, prj.Slug)
			}
		}
	}

	return nil
}

// Get selected project
func getSelectedProject(ctx context.Context) (project, error) {
	selectedOrg, err := getSelectedOrganization(ctx)
	if err != nil {
		return project{}, eris.Wrap(err, "Failed to get organization")
	}

	if selectedOrg.ID == "" {
		printNoSelectedOrganization()
		return project{}, nil
	}

	// Get config
	config, err := globalconfig.GetGlobalConfig()
	if err != nil {
		return project{}, eris.Wrap(err, "Failed to get config")
	}

	if config.ProjectID == "" {
		printNoSelectedProject()
		return project{}, nil
	}

	// Send request
	projectURL := fmt.Sprintf(projectURLPattern, baseURL, selectedOrg.ID) + "/" + config.ProjectID
	body, err := sendRequest(ctx, http.MethodGet, projectURL, nil)
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

// Get list of projects in selected organization
func getListOfProjects(ctx context.Context) ([]project, error) {
	selectedOrg, err := getSelectedOrganization(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get organization")
	}

	if selectedOrg.ID == "" {
		printNoSelectedOrganization()
		return nil, nil
	}

	url := fmt.Sprintf(projectURLPattern, baseURL, selectedOrg.ID)
	body, err := sendRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get projects")
	}

	projects, err := parseResponse[[]project](body)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to parse projects")
	}

	return *projects, nil
}

func getListRegions(ctx context.Context, orgID, projID string) ([]string, error) {
	url := fmt.Sprintf(projectURLPattern+"/%s/regions", baseURL, orgID, projID)
	body, err := sendRequest(ctx, http.MethodGet, url, nil)
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

// Get list of projects in selected organization
func getListOfAvailableRegionsForNewProject(ctx context.Context) ([]string, error) {
	selectedOrg, err := getSelectedOrganization(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get organization")
	}
	if selectedOrg.ID == "" {
		printNoSelectedOrganization()
		return nil, nil
	}
	return getListRegions(ctx, selectedOrg.ID, "00000000-0000-0000-0000-000000000000")
}

// Get list of projects in selected organization
func getListOfAvailableRegionsForProject(ctx context.Context) ([]string, error) {
	selectedProj, err := getSelectedProject(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get project")
	}
	if selectedProj.ID == "" {
		printNoSelectedProject()
		return nil, nil
	}
	return getListRegions(ctx, selectedProj.OrgID, selectedProj.ID)
}

func createProject(ctx context.Context) error {
	regions, err := getListOfAvailableRegionsForNewProject(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get available regions")
	}
	// fmt.Println(regions)

	p := project{
		update: false,
	}
	err = p.projectInput(ctx, regions)
	if err != nil {
		return eris.Wrap(err, "Failed to get project input")
	}

	// Send request
	url := fmt.Sprintf(projectURLPattern, baseURL, p.OrgID)
	body, err := sendRequest(ctx, http.MethodPost, url, map[string]interface{}{
		"name":       p.Name,
		"slug":       p.Slug,
		"repo_url":   p.RepoURL,
		"repo_token": p.RepoToken,
		"repo_path":  p.RepoPath,
		"org_id":     p.OrgID,
		"config":     p.Config,
	})
	if err != nil {
		return eris.Wrap(err, "Failed to create project")
	}

	prj, err := parseResponse[project](body)
	if err != nil {
		return eris.Wrap(err, "Failed to parse response")
	}

	fmt.Printf("Project created successfully: %s (%s)\n", prj.Name, prj.Slug)
	return nil
}

func (p *project) inputProjectName(ctx context.Context) error {
	maxAttempts := 5
	attempts := 0

	for {
		if attempts >= maxAttempts {
			return eris.New("Maximum attempts reached for entering project name")
		}

		if err := ctx.Err(); err != nil {
			return err
		}

		name, err := p.promptForName()
		if err != nil {
			return err
		}

		err = p.validateAndSetName(name, &attempts)
		if err == nil {
			return nil
		}
	}
}

func (p *project) promptForName() (string, error) {
	if p.Name != "" {
		fmt.Printf("Change project name [Enter for \"%s\"]: ", p.Name)
	} else {
		fmt.Print("Enter project name: ")
	}

	name, err := getInput()
	if err != nil {
		return "", eris.Wrap(err, "Failed to read project name")
	}

	if name == "" && p.update {
		name = p.Name
	}

	return name, nil
}

func (p *project) validateAndSetName(name string, attempts *int) error {
	if name == "" {
		fmt.Printf("Error: Project name cannot be empty\n")
		*attempts++
		return eris.New("empty name")
	}

	if len(name) > MaxProjectNameLen {
		fmt.Printf("Error: Project name cannot be longer than 50 characters\n")
		*attempts++
		return eris.New("name too long")
	}

	if strings.ContainsAny(name, "<>:\"/\\|?*") {
		fmt.Printf("Error: Project name contains invalid characters\n")
		*attempts++
		return eris.New("invalid characters")
	}

	p.Name = name
	return nil
}

func (p *project) inputProjectSlug(ctx context.Context) error {
	attempts := 0
	maxAttempts := 5
	for attempts < maxAttempts {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if p.Slug != "" {
				fmt.Printf("Change project slug [Enter for \"%s\"] (5 characters, alphanumeric only): ", p.Slug)
			} else {
				fmt.Print("Enter project slug (5 characters, alphanumeric only): ")
			}
			slug, err := getInput()
			if err != nil {
				return eris.Wrap(err, "Failed to read project slug")
			}

			// set existing slug if empty
			if slug == "" && p.update {
				slug = p.Slug
			}

			// Validate slug
			if len(slug) != 5 { //nolint:gomnd
				fmt.Printf("Error: Slug must be exactly 5 characters\n")
				attempts++
				continue
			}

			if !isAlphanumeric(slug) {
				fmt.Printf("Error: Slug must contain only letters (a-z|A-Z) and numbers (0-9)\n")
				attempts++
				continue
			}

			p.Slug = slug
			return nil
		}
	}

	return eris.New("Maximum attempts reached for project slug")
}

func (p *project) inputRepoURLAndToken(ctx context.Context) error {
	attempts := 0
	maxAttempts := 5
	for attempts < maxAttempts {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			repoURL, err := p.promptForRepoURL()
			if err != nil {
				return err
			}

			if err := p.validateRepoURL(repoURL, &attempts); err != nil {
				continue
			}

			repoToken, err := p.promptForRepoToken()
			if err != nil {
				return err
			}

			repoToken = p.processRepoToken(repoToken)

			if err := validateRepoToken(ctx, repoURL, repoToken); err != nil {
				fmt.Printf("Error: %v\n", err)
				attempts++
				continue
			}

			p.RepoURL = repoURL
			p.RepoToken = repoToken
			return nil
		}
	}

	return eris.New("Maximum attempts reached for repository validation")
}

func (p *project) promptForRepoURL() (string, error) {
	if p.RepoURL != "" {
		fmt.Printf("Change repository URL [Enter for \"%s\"] (https format): ", p.RepoURL)
	} else {
		fmt.Print("Enter repository URL (https format): ")
	}

	repoURL, err := getInput()
	if err != nil {
		return "", eris.Wrap(err, "Failed to read repository URL")
	}

	if repoURL == "" && p.update {
		repoURL = p.RepoURL
	}

	return repoURL, nil
}

func (p *project) validateRepoURL(repoURL string, attempts *int) error {
	if !strings.HasPrefix(repoURL, "http://") && !strings.HasPrefix(repoURL, "https://") {
		fmt.Printf("Error: Repository URL must start with http:// or https://\n")
		*attempts++
		return eris.New("invalid URL format")
	}
	return nil
}

func (p *project) promptForRepoToken() (string, error) {
	if p.update {
		fmt.Print("Change repository personal access token " +
			"[Leave empty to use existing token or type 'public' for public repositories]: ")
	} else {
		fmt.Print("Enter repository personal access token " +
			"[Leave empty or type 'public' for public repositories]: ")
	}

	repoToken, err := getInput()
	if err != nil {
		return "", eris.Wrap(err, "Failed to read personal access token")
	}

	return repoToken, nil
}

func (p *project) processRepoToken(repoToken string) string {
	if repoToken == "" && p.update {
		return p.RepoToken
	}
	if repoToken == "public" {
		return ""
	}
	return repoToken
}

func (p *project) inputRepoPath(ctx context.Context) error {
	// Get repository Path
	var repoPath string
	var err error
	attempts := 0
	maxAttempts := 5
	for attempts < maxAttempts {
		// Get repository URL
		if p.update {
			fmt.Printf("Change repository Cardinal path [Enter for \"%s\"] (empty for default path): ", p.RepoPath)
		} else {
			fmt.Print("Enter repository Cardinal path (empty for default path): ")
		}
		repoPath, err = getInput()
		if err != nil {
			return eris.Wrap(err, "Failed to read repository path")
		}

		// strip off any leading slash
		repoPath = strings.TrimPrefix(repoPath, "/")

		// set existing path if empty
		if repoPath == "" && p.update {
			repoPath = p.RepoPath
		}

		// Validate the path exists using the new validateRepoPath function
		if len(repoPath) > 0 {
			if err := validateRepoPath(ctx, p.RepoURL, p.RepoToken, repoPath); err != nil {
				fmt.Printf("Error: %v\n", err)
				attempts++
				continue
			}
		}

		p.RepoPath = repoPath
		return nil
	}
	return eris.New("Maximum attempts reached for entering repo path")
}

func selectProject(ctx context.Context) (project, error) {
	// Get projects from selected organization
	projects, err := getListOfProjects(ctx)
	if err != nil {
		return project{}, eris.Wrap(err, "Failed to get projects")
	}

	if len(projects) == 0 {
		printNoProjectsInOrganization()
		return project{}, nil
	}

	// Display projects as a numbered list
	fmt.Println("\nAvailable projects:")
	for i, proj := range projects {
		fmt.Printf("%d. %s (%s)\n", i+1, proj.Name, proj.Slug)
	}

	// Get user input
	attempts := 0
	maxAttempts := 5
	for attempts < maxAttempts {
		fmt.Print("\nEnter project number (or 'q' to quit): ")
		input, err := getInput()
		if err != nil {
			return project{}, eris.Wrap(err, "Failed to read input")
		}

		input = strings.TrimSpace(input)
		if input == "q" {
			return project{}, eris.New("Project selection canceled")
		}

		// Parse selection
		num, err := strconv.Atoi(input)
		if err != nil || num < 1 || num > len(projects) {
			fmt.Printf("Invalid selection. Please enter a number between 1 and %d\n", len(projects))
			attempts++
			continue
		}

		selectedProject := projects[num-1]

		// Save project to config file
		config, err := globalconfig.GetGlobalConfig()
		if err != nil {
			return project{}, eris.Wrap(err, "Failed to get config")
		}
		config.ProjectID = selectedProject.ID
		err = globalconfig.SaveGlobalConfig(config)
		if err != nil {
			return project{}, eris.Wrap(err, "Failed to save project")
		}

		return selectedProject, nil
	}

	return project{}, eris.New("Maximum attempts reached for selecting project")
}

func deleteProject(ctx context.Context) error {
	project, err := getSelectedProject(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get project")
	}

	// Print project details with formatting
	fmt.Println("\n🗑️  Project Deletion")
	fmt.Println("------------------")
	fmt.Printf("Project Name: %s\n", project.Name)
	fmt.Printf("Project Slug: %s\n\n", project.Slug)

	// Warning message
	fmt.Println("⚠️  WARNING")
	fmt.Println("  This will permanently delete:")
	fmt.Println("  • All deployments")
	fmt.Println("  • All logs")
	fmt.Println("  • All associated resources")
	fmt.Println("")

	// Confirmation prompt
	fmt.Printf("❓ Are you sure you want to delete %s? (Y/n): ", project.Name)
	confirmation, err := getInput()
	if err != nil {
		return eris.Wrap(err, "Failed to read confirmation")
	}

	if confirmation != "Y" {
		if confirmation == "y" {
			fmt.Println("You need to put Y (uppercase) to confirm deletion")
			fmt.Println("\n❌ Project deletion canceled")
			return nil
		}

		fmt.Println("\n❌ Project deletion canceled")
		return nil
	}

	// Send request
	url := fmt.Sprintf(projectURLPattern, baseURL, project.OrgID) + "/" + project.ID
	body, err := sendRequest(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return eris.Wrap(err, "Failed to delete project")
	}

	// Parse response
	_, err = parseResponse[any](body)
	if err != nil {
		return eris.Wrap(err, "Failed to parse response")
	}

	fmt.Printf("Project deleted successfully: %s (%s)\n", project.Name, project.Slug)

	// Remove project from config
	config, err := globalconfig.GetGlobalConfig()
	if err != nil {
		return eris.Wrap(err, "Failed to get config")
	}
	config.ProjectID = ""
	err = globalconfig.SaveGlobalConfig(config)
	if err != nil {
		return eris.Wrap(err, "Failed to save config")
	}

	return nil
}

func updateProject(ctx context.Context) error {
	// get selected project
	p, err := getSelectedProject(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get selected project")
	}

	regions, err := getListOfAvailableRegionsForProject(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get available regions")
	}

	// set update to true
	p.update = true

	// get project input
	err = p.projectInput(ctx, regions)
	if err != nil {
		return eris.Wrap(err, "Failed to get project input")
	}

	// Send request
	url := fmt.Sprintf(projectURLPattern, baseURL, p.OrgID) + "/" + p.ID
	body, err := sendRequest(ctx, http.MethodPut, url, map[string]interface{}{
		"name":       p.Name,
		"slug":       p.Slug,
		"repo_url":   p.RepoURL,
		"repo_token": p.RepoToken,
		"repo_path":  p.RepoPath,
		"config":     p.Config,
	})
	if err != nil {
		return eris.Wrap(err, "Failed to update project")
	}

	_, err = parseResponse[any](body)
	if err != nil {
		return eris.Wrap(err, "Failed to parse response")
	}

	fmt.Printf("Project updated successfully: %s (%s)\n", p.Name, p.Slug)

	return nil
}

func (p *project) projectInput(ctx context.Context, regions []string) error {
	// Get organization
	org, err := getSelectedOrganization(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get organization")
	}
	p.OrgID = org.ID

	if org.ID == "" {
		printNoSelectedOrganization()
		return nil
	}

	err = p.inputProjectName(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get project name")
	}

	err = p.inputProjectSlug(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get project slug")
	}

	err = p.inputRepoURLAndToken(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get repository URL and token")
	}

	err = p.inputRepoPath(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get repository path")
	}

	// Tick Rate
	err = p.inputTickRate(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get environment name")
	}

	// Regions
	err = p.chooseRegion(ctx, regions)
	if err != nil {
		return eris.Wrap(err, "Failed to choose region")
	}

	return nil
}

// inputTickRate prompts the user to enter a tick rate value (default is 1)
// and validates that it is a valid number. Returns error after max attempts or context cancellation.
func (p *project) inputTickRate(ctx context.Context) error {
	attempts := 0
	maxAttempts := 5
	for attempts < maxAttempts {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if p.Config.TickRate != 0 {
				fmt.Printf("Change tick rate [Enter for \"%d\"] (e.g. 10, 20, 30, default is 1): ", p.Config.TickRate)
			} else {
				fmt.Print("Enter tick rate (e.g. 10, 20, 30, default is 1): ")
			}
			tickRate, err := getInput()
			if err != nil {
				attempts++
				fmt.Printf("Error: Invalid input. Please enter a number\n")
				continue
			}

			// set existing tick rate if empty
			if tickRate == "" && p.update {
				tickRate = strconv.Itoa(p.Config.TickRate)
			}

			p.Config.TickRate, err = strconv.Atoi(tickRate)
			if err != nil {
				attempts++
				fmt.Printf("Error: Invalid input. Please enter a number\n")
				continue
			}
			return nil
		}
	}
	return eris.New("Maximum attempts reached for entering tick rate")
}

// chooseRegion displays an interactive menu for selecting one or more AWS regions
// using the bubbletea TUI library. Returns error if no regions selected after max attempts
// or context cancellation.
func (p *project) chooseRegion(ctx context.Context, regions []string) error {
	for attempts := 0; attempts < 5; attempts++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			err := p.runRegionSelector(ctx, regions)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}
			if len(p.Config.Region) > 0 {
				return nil
			}
			fmt.Println("Error: At least one region must be selected")
		}
	}

	return eris.New("Maximum attempts reached for selecting regions")
}

func (p *project) runRegionSelector(ctx context.Context, regions []string) error {
	if regionSelector == nil {
		if p.update {
			selectedRegions := make(map[int]bool)
			for i, region := range regions {
				if slices.Contains(p.Config.Region, region) {
					selectedRegions[i] = true
				}
			}
			regionSelector = tea.NewProgram(multiselect.UpdateMultiselectModel(ctx, regions, selectedRegions))
		} else {
			regionSelector = tea.NewProgram(multiselect.InitialMultiselectModel(ctx, regions))
		}
	}
	m, err := regionSelector.Run()
	if err != nil {
		return eris.Wrap(err, "failed to run region selector")
	}

	model, ok := m.(multiselect.Model)
	if !ok {
		return eris.New("failed to get selected regions")
	}

	var selectedRegions []string
	for i, item := range regions {
		if model.Selected[i] {
			selectedRegions = append(selectedRegions, item)
		}
	}

	p.Config.Region = selectedRegions

	return nil
}
