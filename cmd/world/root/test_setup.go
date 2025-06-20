package root

import (
	"context"
	"fmt"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/cmd/internal/clients/api"
	"pkg.world.dev/world-cli/cmd/internal/clients/repo"
	cmdsetup "pkg.world.dev/world-cli/cmd/internal/controllers/cmd_setup"
	"pkg.world.dev/world-cli/cmd/internal/interfaces"
	"pkg.world.dev/world-cli/cmd/internal/models"
	"pkg.world.dev/world-cli/cmd/internal/services/config"
	"pkg.world.dev/world-cli/cmd/internal/services/input"
	orgHandler "pkg.world.dev/world-cli/cmd/world/organization_refactor"
	projHandler "pkg.world.dev/world-cli/cmd/world/project_refactor"
	"pkg.world.dev/world-cli/common/printer"
)

// TestSetupCmd is a command for testing the cmd_setup service end-to-end.
type TestSetupCmd struct {
	Parent *RootCmd `kong:"-"`
	// Test scenarios
	TestLoginOnly           bool `         flag:"" help:"Test login requirement only"`
	TestOrgOnly             bool `         flag:"" help:"Test organization requirement only"`
	TestProjectOnly         bool `         flag:"" help:"Test project requirement only"`
	TestFullSetup           bool `         flag:"" help:"Test full setup flow"`
	TestRepoLookup          bool `         flag:"" help:"Test repository lookup"`
	TestExistingOrgOnly     bool `         flag:"" help:"Test existing organization requirement only"`
	TestExistingProjectOnly bool `         flag:"" help:"Test existing project requirement only"`
}

func (c *TestSetupCmd) Run() error {
	printer.NewLine(1)
	printer.Headerln("  Testing cmd_setup Service  ")
	printer.NewLine(1)

	// Initialize clients
	configService, err := config.NewService("LOCAL")
	if err != nil {
		return eris.Wrap(err, "failed to create config client")
	}

	repoClient := repo.NewClient()
	inputService := input.NewService()
	apiClient := api.NewClient("http://localhost:8001")
	apiClient.SetAuthToken(configService.GetConfig().Credential.Token)

	// Initialize handlers
	projectHandler := projHandler.NewHandler(
		repoClient,
		configService,
		apiClient,
		&inputService,
	)

	orgHandler := orgHandler.NewHandler(
		projectHandler,
		&inputService,
		apiClient,
		configService,
	)

	// Create the setup controller
	setupController := cmdsetup.NewController(
		configService,
		repoClient,
		orgHandler,
		projectHandler,
		apiClient,
		&inputService,
	)

	// Run tests based on flags
	if c.TestLoginOnly {
		return c.testLoginOnly(setupController)
	}
	if c.TestOrgOnly {
		return c.testOrgOnly(setupController)
	}
	if c.TestProjectOnly {
		return c.testProjectOnly(setupController)
	}
	if c.TestFullSetup {
		return c.testFullSetup(setupController)
	}
	if c.TestRepoLookup {
		return c.testRepoLookup(setupController)
	}
	if c.TestExistingOrgOnly {
		return c.testExistingOrgOnly(setupController)
	}
	if c.TestExistingProjectOnly {
		return c.testExistingProjectOnly(setupController)
	}

	// Default: run all tests
	return c.runAllTests(setupController)
}

func (c *TestSetupCmd) testLoginOnly(controller interfaces.CommandSetupController) error {
	printer.Infof("Testing login requirement only...\n")

	req := models.SetupRequest{
		LoginRequired:        models.NeedLogin,
		OrganizationRequired: models.Ignore,
		ProjectRequired:      models.Ignore,
	}

	ctx := context.Background()
	state, err := controller.SetupCommandState(ctx, req)
	if err != nil {
		printer.Errorf("Login test failed: %s\n", err)
		return eris.Wrap(err, "login test failed")
	}

	printer.Successf("Login test passed! Logged in: %v\n", state.LoggedIn)
	if state.User != nil {
		printer.Infof("User: %s (%s)\n", state.User.Name, state.User.Email)
	}
	return nil
}

func (c *TestSetupCmd) testOrgOnly(controller interfaces.CommandSetupController) error {
	printer.Infof("Testing organization requirement only...\n")

	req := models.SetupRequest{
		LoginRequired:        models.NeedLogin,
		OrganizationRequired: models.NeedData,
		ProjectRequired:      models.Ignore,
	}

	ctx := context.Background()
	state, err := controller.SetupCommandState(ctx, req)
	if err != nil {
		printer.Errorf("Organization test failed: %s\n", err)
		return eris.Wrap(err, "organization test failed")
	}

	printer.Successf("Organization test passed! Logged in: %v\n", state.LoggedIn)
	if state.Organization != nil {
		printer.Infof("Organization: %s (%s)\n", state.Organization.Name, state.Organization.Slug)
	}
	return nil
}

func (c *TestSetupCmd) testProjectOnly(controller interfaces.CommandSetupController) error {
	printer.Infof("Testing project requirement only...\n")

	req := models.SetupRequest{
		LoginRequired:        models.NeedLogin,
		OrganizationRequired: models.NeedExistingData,
		ProjectRequired:      models.NeedData,
	}

	ctx := context.Background()
	state, err := controller.SetupCommandState(ctx, req)
	if err != nil {
		printer.Errorf("Project test failed: %s\n", err)
		return eris.Wrap(err, "project test failed")
	}

	printer.Successf("Project test passed! Logged in: %v\n", state.LoggedIn)
	if state.Organization != nil {
		printer.Infof("Organization: %s (%s)\n", state.Organization.Name, state.Organization.Slug)
	}
	if state.Project != nil {
		printer.Infof("Project: %s (%s)\n", state.Project.Name, state.Project.Slug)
	}
	return nil
}

func (c *TestSetupCmd) testFullSetup(controller interfaces.CommandSetupController) error {
	printer.Infof("Testing full setup flow...\n")

	req := models.SetupRequest{
		LoginRequired:        models.NeedLogin,
		OrganizationRequired: models.NeedData,
		ProjectRequired:      models.NeedData,
	}

	ctx := context.Background()
	state, err := controller.SetupCommandState(ctx, req)
	if err != nil {
		return eris.Wrap(err, "full setup test failed")
	}

	printer.Successf("Full setup test passed!\n")
	printer.Infof("Logged in: %v\n", state.LoggedIn)
	if state.User != nil {
		printer.Infof("User: %s (%s)\n", state.User.Name, state.User.Email)
	}
	if state.Organization != nil {
		printer.Infof("Organization: %s (%s)\n", state.Organization.Name, state.Organization.Slug)
	}
	if state.Project != nil {
		printer.Infof("Project: %s (%s)\n", state.Project.Name, state.Project.Slug)
	}
	return nil
}

func (c *TestSetupCmd) testRepoLookup(controller interfaces.CommandSetupController) error {
	printer.Infof("Testing repository lookup...\n")

	req := models.SetupRequest{
		LoginRequired:        models.NeedLogin,
		OrganizationRequired: models.NeedRepoLookup,
		ProjectRequired:      models.NeedRepoLookup,
	}

	ctx := context.Background()
	state, err := controller.SetupCommandState(ctx, req)
	if err != nil {
		return eris.Wrap(err, "repo lookup test failed")
	}

	printer.Successf("Repo lookup test passed!\n")
	printer.Infof("Current repo known: %v\n", state.CurrRepoKnown)
	if state.Organization != nil {
		printer.Infof("Organization: %s (%s)\n", state.Organization.Name, state.Organization.Slug)
	}
	if state.Project != nil {
		printer.Infof("Project: %s (%s)\n", state.Project.Name, state.Project.Slug)
	}
	return nil
}

func (c *TestSetupCmd) testExistingOrgOnly(controller interfaces.CommandSetupController) error {
	printer.Infof("Testing existing organization requirement only...\n")

	req := models.SetupRequest{
		LoginRequired:        models.NeedLogin,
		OrganizationRequired: models.NeedExistingData,
		ProjectRequired:      models.Ignore,
	}

	ctx := context.Background()
	state, err := controller.SetupCommandState(ctx, req)
	if err != nil {
		return eris.Wrap(err, "existing organization test failed")
	}

	printer.Successf("Existing organization test passed!\n")
	if state.Organization != nil {
		printer.Infof("Organization: %s (%s)\n", state.Organization.Name, state.Organization.Slug)
	}
	return nil
}

func (c *TestSetupCmd) testExistingProjectOnly(controller interfaces.CommandSetupController) error {
	printer.Infof("Testing existing project requirement only...\n")

	req := models.SetupRequest{
		LoginRequired:        models.NeedLogin,
		OrganizationRequired: models.NeedExistingData,
		ProjectRequired:      models.NeedExistingData,
	}

	ctx := context.Background()
	state, err := controller.SetupCommandState(ctx, req)
	if err != nil {
		return eris.Wrap(err, "existing project test failed")
	}

	printer.Successf("Existing project test passed!\n")
	if state.Organization != nil {
		printer.Infof("Organization: %s (%s)\n", state.Organization.Name, state.Organization.Slug)
	}
	if state.Project != nil {
		printer.Infof("Project: %s (%s)\n", state.Project.Name, state.Project.Slug)
	}
	return nil
}

func (c *TestSetupCmd) runAllTests(controller interfaces.CommandSetupController) error {
	printer.Infof("Running all cmd_setup tests...\n")
	printer.NewLine(1)

	tests := []struct {
		name string
		fn   func(interfaces.CommandSetupController) error
	}{
		{"Login Only", c.testLoginOnly},
		{"Organization Only", c.testOrgOnly},
		{"Project Only", c.testProjectOnly},
		{"Full Setup", c.testFullSetup},
		{"Repo Lookup", c.testRepoLookup},
		{"Existing Organization Only", c.testExistingOrgOnly},
		{"Existing Project Only", c.testExistingProjectOnly},
	}

	passed := 0
	failed := 0

	for _, test := range tests {
		printer.Infof("Running test: %s\n", test.name)
		err := test.fn(controller)
		if err != nil {
			printer.Errorf("Test '%s' failed: %s\n", test.name, err)
			failed++
		} else {
			printer.Successf("Test '%s' passed!\n", test.name)
			passed++
		}
		printer.NewLine(1)
	}

	printer.Headerln("  Test Results  ")
	printer.Successf("Passed: %d\n", passed)
	printer.Errorf("Failed: %d\n", failed)
	printer.Infof("Total: %d\n", passed+failed)

	if failed > 0 {
		return eris.New(fmt.Sprintf("Some tests failed: %d passed, %d failed", passed, failed))
	}

	return nil
}
