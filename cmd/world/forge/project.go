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

	fmt.Println("\n📁 ✨ Project Information ✨")
	fmt.Println("============================")
	if project.Name == "" {
		fmt.Println("\n❌ No project selected")
		fmt.Println("\nℹ️  Use 'world forge project select' to choose a project")
	} else {
		fmt.Println("\n📋 Available Projects:")
		fmt.Println("---------------------------")
		for _, prj := range projects {
			if prj.ID == project.ID {
				fmt.Printf("🌟 %s (%s) [SELECTED]\n", prj.Name, prj.Slug)
			} else {
				fmt.Printf("📎 %s (%s)\n", prj.Name, prj.Slug)
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

func createProject(ctx context.Context) (*project, error) {
	regions, err := getListOfAvailableRegionsForNewProject(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get available regions")
	}
	// fmt.Println(regions)

	p := project{
		update: false,
	}
	err = p.projectInput(ctx, regions)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get project input")
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
		return nil, eris.Wrap(err, "Failed to create project")
	}

	prj, err := parseResponse[project](body)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to parse response")
	}

	// Select project
	config, err := globalconfig.GetGlobalConfig()
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get config")
	}
	config.ProjectID = prj.ID

	err = globalconfig.SaveGlobalConfig(config)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to select project")
	}

	fmt.Printf("\n✨ Project '%s' created successfully! ✨\n", prj.Name)
	fmt.Printf("📋 Project Details:\n")
	fmt.Printf("  • Name: %s\n", prj.Name)
	fmt.Printf("  • Slug: %s\n", prj.Slug)
	fmt.Printf("  • ID: %s\n", prj.ID)
	fmt.Printf("  • Repository URL: %s\n", prj.RepoURL)
	fmt.Printf("  • Repository Path: %s\n", prj.RepoPath)
	fmt.Printf("  • Tick Rate: %d\n", prj.Config.TickRate)
	fmt.Printf("  • Regions:\n")
	for _, region := range prj.Config.Region {
		fmt.Printf("    - %s\n", region)
	}
	fmt.Printf("  • Deploy Secret (for deploy via CI/CD pipeline tools):\n")
	fmt.Printf("      %s\n", prj.DeploySecret)
	fmt.Printf("ℹ️ Deploy Secret will not be shown again. Save it now in a secure location.\n")

	return prj, nil
}

func (p *project) inputProjectName(ctx context.Context) error {
	maxAttempts := 5
	attempts := 0

	fmt.Println("\n🎨 ✨ Project Name Configuration ✨")
	fmt.Println("=================================")
	fmt.Println("\nℹ️  Project name requirements:")
	fmt.Println("  • Must not be empty")
	fmt.Printf("  • Maximum length: %d characters\n", MaxProjectNameLen)
	fmt.Println("  • Cannot contain: < > : \" / \\ | ? *")

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
			fmt.Printf("\n✅ Project name \"%s\" accepted!\n", name)
			return nil
		}
	}
}

func (p *project) promptForName() (string, error) {
	fmt.Println("\n📝 ✨ Project Name Configuration ✨")
	fmt.Println("================================")
	if p.Name != "" {
		fmt.Printf("\n📋 Current name: \"%s\"\n", p.Name)
		fmt.Print("\n✨ Enter new name (or press Enter to keep current): ")
	} else {
		fmt.Print("\n✨ Enter project name: ")
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
	maxAttempts := 5
	if name == "" {
		fmt.Printf("\n❌ Error: Project name cannot be empty (attempt %d/%d)\n", *attempts+1, maxAttempts)
		*attempts++
		return eris.New("empty name")
	}

	if len(name) > MaxProjectNameLen {
		fmt.Printf("\n❌ Error: Project name cannot be longer than %d characters (attempt %d/%d)\n",
			MaxProjectNameLen, *attempts+1, maxAttempts)
		*attempts++
		return eris.New("name too long")
	}

	if strings.ContainsAny(name, "<>:\"/\\|?*") {
		fmt.Printf("\n❌ Error: Project name contains invalid characters (attempt %d/%d)\n"+
			"   Invalid characters: < > : \" / \\ | ? *\n", *attempts+1, maxAttempts)
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
			fmt.Println("\n🔖 ✨ Project Slug Configuration ✨")
			fmt.Println("================================")
			if p.Slug != "" {
				fmt.Printf("\n📝 Current slug: \"%s\"\n", p.Slug)
				fmt.Print("\n✨ Enter new slug (or press Enter to keep current)")
			} else {
				fmt.Print("\n✨ Enter new project slug")
			}
			fmt.Print("\n\n📋 Requirements:")
			fmt.Print("\n   • Exactly 5 characters")
			fmt.Print("\n   • Letters (a-z|A-Z) and numbers (0-9) only")
			fmt.Print("\n\n👉 Slug: ")

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
				fmt.Printf("\n❌ Error: Slug must be exactly 5 characters (attempt %d/%d)\n", attempts+1, maxAttempts)
				attempts++
				continue
			}

			if !isAlphanumeric(slug) {
				fmt.Printf("\n❌ Error: Slug must contain only letters (a-z|A-Z) and numbers (0-9) (attempt %d/%d)\n",
					attempts+1, maxAttempts)
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
	fmt.Printf("\n🔗 Repository URL Configuration")
	fmt.Printf("\n============================")
	fmt.Printf("\n\n✨ Enter repository URL:")
	if p.RepoURL != "" {
		fmt.Printf("\n   • Press Enter to keep: %s", p.RepoURL)
		fmt.Printf("\n   • Or enter new URL (https format)")
		fmt.Printf("\n\nURL: ")
	} else {
		fmt.Printf("\n   • Must use https format")
		fmt.Printf("\n\nURL: ")
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
		fmt.Printf("\n❌ Error: Invalid Repository URL Format\n")
		fmt.Printf("============================\n")
		fmt.Printf("🔍 The URL must start with:\n")
		fmt.Printf("   • http://\n")
		fmt.Printf("   • https://\n\n")
		*attempts++
		return eris.New("invalid URL format")
	}
	return nil
}

func (p *project) promptForRepoToken() (string, error) {
	if p.update {
		fmt.Printf("\n🔑 Update Repository Access Token\n")
		fmt.Printf("==============================\n")
		fmt.Printf("\n✨ Enter new token (options):\n")
		fmt.Printf("   • Press Enter to keep existing token\n")
		fmt.Printf("   • Type 'public' for public repositories\n")
		fmt.Printf("   • Enter new token for private repositories\n")
		fmt.Printf("\nToken: ")
	} else {
		fmt.Printf("\n🔑 Repository Access Token\n")
		fmt.Printf("=======================\n")
		fmt.Printf("\n✨ Enter token (options):\n")
		fmt.Printf("   • Type 'public' for public repositories\n")
		fmt.Printf("   • Enter token for private repositories\n")
		fmt.Printf("\nToken: ")
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
			fmt.Printf("\n📂 Change Repository Cardinal Path\n")
			fmt.Printf("================================\n")
			fmt.Printf("Current path: \"%s\"\n", p.RepoPath)
			fmt.Printf("\n✨ Enter new path (or press Enter to keep current, empty for default): ")
		} else {
			fmt.Printf("\n📂 Set Repository Cardinal Path\n")
			fmt.Printf("============================\n")
			fmt.Printf("\n✨ Enter path (empty for default): ")
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
				fmt.Printf("\n❌ Error: %v\n", err)
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
	fmt.Println("\n📁 ✨ Available Projects ✨")
	fmt.Println("==========================")
	fmt.Println("\n📋 Project List:")
	fmt.Println("--------------")
	for i, proj := range projects {
		fmt.Printf("  %d. 📂 %s\n     └─ 🔖 Slug: %s\n", i+1, proj.Name, proj.Slug)
	}

	// Get user input
	attempts := 0
	maxAttempts := 5
	for attempts < maxAttempts {
		fmt.Print("\n✨ Enter project number (or 'q' to quit): ")
		input, err := getInput()
		if err != nil {
			return project{}, eris.Wrap(err, "Failed to read input")
		}

		input = strings.TrimSpace(input)
		if input == "q" {
			fmt.Println("\n❌ Project selection canceled")
			return project{}, eris.New("Project selection canceled")
		}

		// Parse selection
		num, err := strconv.Atoi(input)
		if err != nil || num < 1 || num > len(projects) {
			fmt.Printf("\n❌ Invalid selection. Please enter a number between 1 and %d (attempt %d/%d)\n",
				len(projects), attempts+1, maxAttempts)
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

		fmt.Printf("\n✅ Selected project: %s\n", selectedProject.Name)
		return selectedProject, nil
	}

	return project{}, eris.New("Maximum attempts reached for selecting project")
}

func deleteProject(ctx context.Context) error {
	project, err := getSelectedProject(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get project")
	}

	// Print project details with fancy formatting
	fmt.Println("\n🗑️  ✨ Project Deletion ✨")
	fmt.Println("===========================")
	fmt.Printf("\n📋 Project Details:")
	fmt.Printf("\n  • 📝 Name: %s", project.Name)
	fmt.Printf("\n  • 🔖 Slug: %s\n", project.Slug)

	// Warning message with fancy formatting
	fmt.Println("\n⚠️  WARNING!")
	fmt.Println("===========")
	fmt.Println("\n❗ This action will permanently delete:")
	fmt.Println("  • 🚀 All deployments")
	fmt.Println("  • 📜 All logs")
	fmt.Println("  • 🔧 All associated resources")
	fmt.Println("")

	// Confirmation prompt with fancy formatting
	fmt.Printf("❓ Type 'Y' (uppercase) to confirm deletion of '%s': ", project.Name)
	confirmation, err := getInput()
	if err != nil {
		return eris.Wrap(err, "Failed to read confirmation")
	}

	if confirmation != "Y" {
		if confirmation == "y" {
			fmt.Println("\n❌ Error: You must type 'Y' (uppercase) to confirm deletion")
			fmt.Println("\n🚫 Project deletion canceled")
			return nil
		}

		fmt.Println("\n🚫 Project deletion canceled")
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

	fmt.Println("\n✨ Success! ✨")
	fmt.Println("==============")
	fmt.Printf("\n✅ Project deleted: %s (%s)\n", project.Name, project.Slug)

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

	fmt.Println("\n📝 ✨ Project Update ✨")
	fmt.Println("=======================")

	// get project input
	err = p.projectInput(ctx, regions)
	if err != nil {
		return eris.Wrap(err, "Failed to get project input")
	}

	fmt.Println("\n🔄 Updating project...")

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

	fmt.Printf("\n✨ Project '%s' updated successfully! ✨\n", p.Name)
	fmt.Printf("📋 Project Details:\n")
	fmt.Printf("  • Name: %s\n", p.Name)
	fmt.Printf("  • Slug: %s\n", p.Slug)
	fmt.Printf("  • ID: %s\n", p.ID)
	fmt.Printf("  • Repository URL: %s\n", p.RepoURL)
	fmt.Printf("  • Repository Path: %s\n", p.RepoPath)
	fmt.Printf("  • Tick Rate: %d\n", p.Config.TickRate)
	fmt.Printf("  • Regions:\n")
	for _, region := range p.Config.Region {
		fmt.Printf("    - %s\n", region)
	}

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
			fmt.Println("\n⚡ Tick Rate Configuration")
			fmt.Println("========================")
			if p.Config.TickRate != 0 {
				fmt.Printf("\n🔄 Current tick rate: %d\n", p.Config.TickRate)
				fmt.Print("✨ Enter new tick rate [press Enter to keep current]\n")
			} else {
				fmt.Print("\n✨ Enter tick rate for your project:\n")
			}
			fmt.Print("   └─ Examples: 10, 20, 30 (default is 1): ")

			tickRate, err := getInput()
			if err != nil {
				attempts++
				fmt.Printf("\n❌ Invalid input. Please enter a number (attempt %d/%d)\n", attempts, maxAttempts)
				continue
			}

			// set existing tick rate if empty
			if tickRate == "" && p.update {
				tickRate = strconv.Itoa(p.Config.TickRate)
			}

			p.Config.TickRate, err = strconv.Atoi(tickRate)
			if err != nil {
				attempts++
				fmt.Printf("\n❌ Invalid input. Please enter a number (attempt %d/%d)\n", attempts, maxAttempts)
				continue
			}
			fmt.Printf("\n✅ Tick rate set to: %d\n", p.Config.TickRate)
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
				fmt.Printf("\n❌ Error: %v\n", err)
				continue
			}
			if len(p.Config.Region) > 0 {
				return nil
			}
			fmt.Println("\n⚠️  Error: At least one region must be selected")
			fmt.Printf("\n🔄 Attempt %d/5 - Please try again\n", attempts+1)
		}
	}

	fmt.Println("\n❌ Region Selection Failed")
	fmt.Println("========================")
	fmt.Println("\nℹ️  Maximum attempts reached. Please try the command again.")
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

// handleProjectSelection manages the project selection logic
func handleProjectSelection(ctx context.Context, projectID string) (string, error) {
	projects, err := getListOfProjects(ctx)
	if err != nil {
		return "", eris.Wrap(err, "Failed to get projects")
	}

	switch numProjects := len(projects); {
	case numProjects == 1:
		return projects[0].ID, nil
	case numProjects > 1:
		return handleMultipleProjects(ctx, projectID, projects)
	default:
		return handleNoProjects(ctx)
	}
}

// handleMultipleProjects handles the case when there are multiple projects
func handleMultipleProjects(ctx context.Context, projectID string, projects []project) (string, error) {
	for _, project := range projects {
		if project.ID == projectID {
			return projectID, nil
		}
	}

	project, err := selectProject(ctx)
	if err != nil {
		return "", eris.Wrap(err, "Failed to select project")
	}
	return project.ID, nil
}

// handleNoProjects handles the case when there are no projects
func handleNoProjects(ctx context.Context) (string, error) {
	// Confirmation prompt
	fmt.Printf("❓ You don't have any projects in this organization. Do you want to create a new project now? (Y/n): ")
	confirmation, err := getInput()
	if err != nil {
		return "", eris.Wrap(err, "Failed to read confirmation")
	}

	if confirmation != "Y" {
		if confirmation == "y" {
			fmt.Println("You need to put Y (uppercase) to confirm creation")
			fmt.Println("\n❌ Project creation canceled")
			return "", nil
		}

		return "", nil
	}

	project, err := createProject(ctx)
	if err != nil {
		return "", eris.Wrap(err, "Failed to create project")
	}
	return project.ID, nil
}
