package forge

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/suite"

	"pkg.world.dev/world-cli/common/globalconfig"
	"pkg.world.dev/world-cli/tea/component/multiselect"
)

var (
	originalGenerateKey  = generateKey
	originalOpenBrowser  = openBrowser
	originalGetInput     = getInput
	originalGetConfigDir = globalconfig.GetConfigDir
	tempDir              string
)

type ForgeTestSuite struct {
	suite.Suite
	server *httptest.Server
	ctx    context.Context
}

func (s *ForgeTestSuite) SetupTest() {
	s.ctx = context.Background()

	// Create test server on port 8001
	listener, err := net.Listen("tcp", ":8001")
	s.Require().NoError(err)

	// Create test server
	s.server = &httptest.Server{
		Listener: listener,
		Config: &http.Server{
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/organization":
					s.handleOrganizationList(w, r)
				case "/api/organization/empty-org-id":
					s.handleOrganizationGet(w, r)
				case "/api/organization/test-org-id":
					s.handleOrganizationGet(w, r)
				case "/api/organization/invalid-org-id":
					http.Error(w, "Not found", http.StatusNotFound)
				case "/api/organization/test-org-id/project":
					s.handleProjectList(w, r)
				case "/api/organization/test-org-id/project/test-project-id":
					s.handleProjectGet(w, r)
				case "/api/organization/test-org-id/project/invalid-project-id":
					http.Error(w, "Not found", http.StatusNotFound)
				case "/api/organization/test-org-id/project/test-project-id/deploy":
					s.handleDeploy(w, r)
				case "/api/organization/test-org-id/project/test-project-id/destroy":
					s.handleDestroy(w, r)
				case "/api/organization/test-org-id/project/test-project-id/reset":
					s.handleReset(w, r)
				case "/api/organization/invalid-org-id/project/test-project-id/deploy":
					http.Error(w, "Organization not found", http.StatusNotFound)
				case "/api/organization/test-org-id/project/invalid-project-id/deploy":
					http.Error(w, "Project not found", http.StatusNotFound)
				case "/api/organization/test-org-id/project/00000000-0000-0000-0000-000000000000/regions":
					s.handleGetRegions(w, r)
				case "/api/organization/test-org-id/invite":
					s.handleInvite(w, r)
				case "/api/organization/test-org-id/role":
					s.handleRole(w, r)
				case "/api/organization/invalid-org-id/invite":
					http.Error(w, "Organization not found", http.StatusNotFound)
				case "/api/organization/invalid-org-id/role":
					http.Error(w, "Organization not found", http.StatusNotFound)
				case "/api/user/login":
					s.handleLogin(w, r)
				case "/api/user/login/get-token":
					s.handleGetToken(w, r)
				case "/api/organization/empty-org-id/project":
					s.writeJSON(w, map[string]interface{}{"data": []project{}})
				case "/api/deployment/test-project-id":
					s.handleStatusDeployed(w, r)
				case "/api/deployment/failedbuild-project-id":
					s.handleStatusFailedBuild(w, r)
				case "/api/deployment/undeployed-project-id":
					s.handleStatusUndeployed(w, r)
				case "/api/deployment/destroyed-project-id":
					s.handleStatusDestroyed(w, r)
				case "/api/deployment/reset-project-id":
					s.handleStatusReset(w, r)
				case "/api/health/test-project-id":
					s.handleHealth(w, r)
				case "/api/health/reset-project-id":
					s.handleHealth(w, r)
				case "/api/health/destroyed-project-id":
					s.handleHealth(w, r)
				default:
					http.Error(w, "Not found", http.StatusNotFound)
				}
			}),
		},
	}
	s.server.Start()

	// Set max attempts to 3 for login tests
	maxAttempts = 3

	// Create temp config dir
	tempDir = filepath.Join(os.TempDir(), "worldcli")
	globalconfig.GetConfigDir = func() (string, error) {
		return tempDir, nil
	}
	err = globalconfig.SetupConfigDir()
	s.Require().NoError(err)

	// Create config file
	err = globalconfig.SaveGlobalConfig(globalconfig.GlobalConfig{
		OrganizationID: "test-org-id",
		ProjectID:      "test-project-id",
		Credential: globalconfig.Credential{
			Token: "test-token",
		},
	})
	s.Require().NoError(err)
}

func (s *ForgeTestSuite) TearDownTest() {
	s.server.Close()

	// Remove temp config dir
	os.RemoveAll(tempDir)

	// Restore original functions
	globalconfig.GetConfigDir = originalGetConfigDir
}

func (s *ForgeTestSuite) handleInvite(w http.ResponseWriter, _ *http.Request) {
	s.writeJSON(w, map[string]interface{}{"data": ""})
}

func (s *ForgeTestSuite) handleRole(w http.ResponseWriter, _ *http.Request) {
	s.writeJSON(w, map[string]interface{}{"data": ""})
}

func (s *ForgeTestSuite) handleOrganizationList(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		orgs := []organization{
			{
				ID:   "test-org-id",
				Name: "Test Org",
				Slug: "testo",
			},
		}
		s.writeJSON(w, map[string]interface{}{"data": orgs})
	case http.MethodPost:
		org := organization{
			ID:   "test-org-id",
			Name: "Test Organization",
			Slug: "testo",
		}
		s.writeJSON(w, map[string]interface{}{"data": org})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *ForgeTestSuite) handleOrganizationGet(w http.ResponseWriter, r *http.Request) {
	// Get last path segment
	parts := strings.Split(r.URL.Path, "/")
	orgID := parts[len(parts)-1]

	org := organization{
		ID:   "test-org-id",
		Name: "Test Org",
		Slug: "testo",
	}

	if orgID == "empty-org-id" {
		org = organization{
			ID:   "empty-org-id",
			Name: "Empty Org",
			Slug: "empty",
		}
	}

	s.writeJSON(w, map[string]interface{}{"data": org})
}

func (s *ForgeTestSuite) handleProjectList(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		parsedBody, err := io.ReadAll(r.Body)
		s.Require().NoError(err)
		defer r.Body.Close()

		body := map[string]interface{}{}
		err = json.Unmarshal(parsedBody, &body)
		s.Require().NoError(err)

		proj := project{
			ID:      "test-project-id",
			OrgID:   "test-org-id",
			Name:    body["name"].(string),
			Slug:    body["slug"].(string),
			RepoURL: body["repo_url"].(string),
		}
		s.writeJSON(w, map[string]interface{}{"data": proj})
		return
	}

	projects := []project{
		{
			ID:      "test-project-id",
			OrgID:   "test-org-id",
			Name:    "Test Project",
			Slug:    "testp",
			RepoURL: "https://github.com/test/repo",
		},
	}
	s.writeJSON(w, map[string]interface{}{"data": projects})
}

func (s *ForgeTestSuite) handleProjectGet(w http.ResponseWriter, _ *http.Request) {
	proj := project{
		ID:      "test-project-id",
		OrgID:   "test-org-id",
		Name:    "Test Project",
		Slug:    "testp",
		RepoURL: "https://github.com/test/repo",
	}
	s.writeJSON(w, map[string]interface{}{"data": proj})
}

func (s *ForgeTestSuite) handleGetRegions(w http.ResponseWriter, _ *http.Request) {
	result := map[string]string{
		"38f46cb3-63a3-4955-ae5f-6c31595fd970": "ap-southeast-1",
		"4ee8a580-879f-47c8-a183-de6d50329dc1": "us-east-1",
		"71d61857-f803-4135-80a7-68b3e6f55443": "eu-central-1",
		"f80a422c-eb8d-4d6d-8244-0f065773cb20": "us-west-2",
	}
	s.writeJSON(w, map[string]interface{}{"data": result})
}

func (s *ForgeTestSuite) handleDeploy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// check if preview flag is set
	preview := r.URL.Query().Get("preview")
	if preview == "true" {
		deploymentPreview := deploymentPreview{
			ProjectName:    "Test Project",
			ProjectSlug:    "testp",
			OrgName:        "Test Org",
			OrgSlug:        "testo",
			ExecutorName:   "Test Executor",
			DeploymentType: "deploy",
			TickRate:       10,
			Regions:        []string{"ap-southeast-1", "us-east-1", "eu-central-1", "us-west-2"},
		}

		s.writeJSON(w, map[string]interface{}{"data": deploymentPreview})
		return
	}

	s.writeJSON(w, map[string]interface{}{"data": "deployment started"})
}

func (s *ForgeTestSuite) handleStatusDeployed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.writeJSONString(w, `{"data":{"dev":{
		"project_id":"test-project-id",
		"type":"deploy",
		"executor_id":"test-executor-id",
		"execution_time":"2001-01-01T01:02:00Z",
		"build_number":1,
		"build_start_time":"2001-01-01T01:01:00Z",
		"build_state":"passed"
	}}}`)
}

func (s *ForgeTestSuite) handleStatusFailedBuild(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.writeJSONString(w, `{"data":{"dev":{
		"project_id":"failedbuild-project-id",
		"type":"deploy",
		"executor_id":"test-executor-id",
		"execution_time":"2001-01-01T01:02:00Z",
		"build_number":1,
		"build_start_time":"2001-01-01T01:01:00Z",
		"build_state":"failed"
	}}}`)
}

func (s *ForgeTestSuite) handleStatusDestroyed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.writeJSONString(w, `{"data":{"dev":{
		"project_id":"destroyed-project-id",
		"type":"destroy",
		"executor_id":"test-executor-id",
		"execution_time":"2001-01-01T01:02:00Z",
		"build_state":"passed"
	}}}`)
}

func (s *ForgeTestSuite) handleStatusReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.writeJSONString(w, `{"data":{"dev":{
		"project_id":"reset-project-id",
		"type":"reset",
		"executor_id":"test-executor-id",
		"execution_time":"2001-01-01T01:02:00Z",
		"build_state":"passed"
	}}}`)
}

func (s *ForgeTestSuite) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.writeJSONString(w, `{"data":{"dev":{"ok":false,"offline":false,"deployed_instances":[
	{
		"region":"ap-southeast-1",
		"instance":1,
		"cardinal":{
			"url":"https://cardinal.apse-1.test.com/health",
			"ok":true,
            "result_code":200,
			"result_str":"{}"
		},
		"nakama":{
			"url":"https://nakama.apse-1.test.com/healthcheck",
			"ok":true,
            "result_code":200,
			"result_str":"{}"
		}
    },
    {
		"region":"us-east-1",
		"instance":1,
		"cardinal":{
			"url":"https://cardinal01.use-1.test.com/health",
			"ok":false,
            "result_code":404,
			"result_str":"Not Found"
		},
		"nakama":{
			"url":"https://nakama01.use-1.test.com/healthcheck",
			"ok":false,
            "result_code":0,
			"result_str":""
		}
    },
    {
		"region":"us-east-1",
		"instance":2,
		"cardinal":{
			"url":"https://cardinal02.use-1.test.com/health",
			"ok":false,
            "result_code":0,
			"result_str":""
		},
		"nakama":{
			"url":"https://nakama02.use1-1.test.com/healthcheck",
			"ok":false,
            "result_code":502,
			"result_str":"Bad Gateway"
		}
    }
	]}}}`)
}

func (s *ForgeTestSuite) handleStatusUndeployed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.writeJSONString(w, `{}`)
}

func (s *ForgeTestSuite) handleDestroy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// check if preview flag is set
	preview := r.URL.Query().Get("preview")
	if preview == "true" {
		deploymentPreview := deploymentPreview{
			ProjectName:    "Test Project",
			ProjectSlug:    "testp",
			OrgName:        "Test Org",
			OrgSlug:        "testo",
			ExecutorName:   "Test Executor",
			DeploymentType: "deploy",
			TickRate:       10,
			Regions:        []string{"ap-southeast-1", "us-east-1", "eu-central-1", "us-west-2"},
		}

		s.writeJSON(w, map[string]interface{}{"data": deploymentPreview})
		return
	}

	s.writeJSON(w, map[string]interface{}{"data": "destroy started"})
}

func (s *ForgeTestSuite) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Just return success as the actual endpoint opens a browser
	w.WriteHeader(http.StatusOK)
}

func (s *ForgeTestSuite) handleGetToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	key := r.URL.Query().Get("key")
	if key == "valid-key" {
		token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9." +
			"eyJ1c2VyX21ldGFkYXRhIjp7Im5hbWUiOiJUZXN0IFVzZXIiLCJzdWIiOiJ0ZXN0LXVzZXItaWQifX0.sign"
		s.writeJSON(w, map[string]interface{}{
			"data": token,
		})
	} else {
		http.Error(w, "Invalid key", http.StatusBadRequest)
	}
}

func (s *ForgeTestSuite) handleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// check if preview flag is set
	preview := r.URL.Query().Get("preview")
	if preview == "true" {
		deploymentPreview := deploymentPreview{
			ProjectName:    "Test Project",
			ProjectSlug:    "testp",
			OrgName:        "Test Org",
			OrgSlug:        "testo",
			ExecutorName:   "Test Executor",
			DeploymentType: "deploy",
			TickRate:       10,
			Regions:        []string{"ap-southeast-1", "us-east-1", "eu-central-1", "us-west-2"},
		}

		s.writeJSON(w, map[string]interface{}{"data": deploymentPreview})
		return
	}

	s.writeJSON(w, map[string]interface{}{"data": "reset started"})
}

func (s *ForgeTestSuite) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(data)
	s.Require().NoError(err)
}

func (s *ForgeTestSuite) writeJSONString(w http.ResponseWriter, data string) {
	w.Header().Set("Content-Type", "application/json")
	_, err := w.Write([]byte(data))
	s.Require().NoError(err)
}

func (s *ForgeTestSuite) TestGetSelectedOrganization() {
	testCases := []struct {
		name          string
		config        globalconfig.GlobalConfig
		expectedError bool
		expectedOrg   *organization
	}{
		{
			name: "Success - Valid organization",
			config: globalconfig.GlobalConfig{
				OrganizationID: "test-org-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			expectedError: false,
			expectedOrg: &organization{
				ID:   "test-org-id",
				Name: "Test Org",
				Slug: "testo",
			},
		},
		{
			name: "Error - Invalid organization ID",
			config: globalconfig.GlobalConfig{
				OrganizationID: "invalid-org-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			expectedError: true,
			expectedOrg:   nil,
		},
		{
			name:          "Error - No organization selected",
			config:        globalconfig.GlobalConfig{},
			expectedError: false,
			expectedOrg:   nil,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := globalconfig.SaveGlobalConfig(tc.config)
			s.Require().NoError(err)

			org, err := getSelectedOrganization(s.ctx)
			switch {
			case tc.expectedError:
				s.Require().Error(err)
				s.Empty(org)
			case tc.expectedOrg == nil:
				s.Require().NoError(err)
				s.Empty(org)
			default:
				s.Require().NoError(err)
				s.Equal(tc.expectedOrg.ID, org.ID)
				s.Equal(tc.expectedOrg.Name, org.Name)
				s.Equal(tc.expectedOrg.Slug, org.Slug)
			}
		})
	}
}

func (s *ForgeTestSuite) TestGetSelectedProject() {
	testCases := []struct {
		name          string
		config        globalconfig.GlobalConfig
		expectedError bool
		expectedProj  *project
	}{
		{
			name: "Success - Valid project",
			config: globalconfig.GlobalConfig{
				OrganizationID: "test-org-id",
				ProjectID:      "test-project-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			expectedError: false,
			expectedProj: &project{
				ID:      "test-project-id",
				OrgID:   "test-org-id",
				Name:    "Test Project",
				Slug:    "testp",
				RepoURL: "https://github.com/test/repo",
			},
		},
		{
			name: "Error - Invalid project ID",
			config: globalconfig.GlobalConfig{
				OrganizationID: "test-org-id",
				ProjectID:      "invalid-project-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			expectedError: true,
			expectedProj:  nil,
		},
		{
			name: "Error - No organization selected",
			config: globalconfig.GlobalConfig{
				ProjectID: "test-project-id",
			},
			expectedError: false,
			expectedProj:  nil,
		},
		{
			name: "Error - No project selected",
			config: globalconfig.GlobalConfig{
				OrganizationID: "test-org-id",
			},
			expectedError: false,
			expectedProj:  nil,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := globalconfig.SaveGlobalConfig(tc.config)
			s.Require().NoError(err)

			proj, err := getSelectedProject(s.ctx)
			switch {
			case tc.expectedError:
				s.Require().Error(err)
				s.Empty(proj)
			case tc.expectedProj == nil:
				s.Require().NoError(err)
				s.Empty(proj)
			default:
				s.Require().NoError(err)
				s.Equal(tc.expectedProj.ID, proj.ID)
				s.Equal(tc.expectedProj.Name, proj.Name)
				s.Equal(tc.expectedProj.Slug, proj.Slug)
				s.Equal(tc.expectedProj.RepoURL, proj.RepoURL)
			}
		})
	}
}

func (s *ForgeTestSuite) TestIsAlphanumeric() {
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Success - Lowercase alphanumeric",
			input:    "abc123",
			expected: true,
		},
		{
			name:     "Success - Uppercase alphanumeric",
			input:    "ABC123",
			expected: true,
		},
		{
			name:     "Success - Mixed case alphanumeric",
			input:    "aBc123",
			expected: true,
		},
		{
			name:     "Error - Contains hyphen",
			input:    "abc-123",
			expected: false,
		},
		{
			name:     "Error - Contains underscore",
			input:    "abc_123",
			expected: false,
		},
		{
			name:     "Error - Contains space",
			input:    "abc 123",
			expected: false,
		},
		{
			name:     "Error - Contains special character",
			input:    "abc@123",
			expected: false,
		},
		{
			name:     "Error - Empty string",
			input:    "",
			expected: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := isAlphanumeric(tc.input)
			s.Equal(tc.expected, result)
		})
	}
}

func (s *ForgeTestSuite) TestDeploy() {
	testCases := []struct {
		name          string
		config        globalconfig.GlobalConfig
		input         string
		expectedError bool
	}{
		{
			name: "Success - Valid deployment",
			config: globalconfig.GlobalConfig{
				OrganizationID: "test-org-id",
				ProjectID:      "test-project-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			input:         "Y",
			expectedError: false,
		},
		{
			name: "Error - Invalid organization ID",
			config: globalconfig.GlobalConfig{
				OrganizationID: "invalid-org-id",
				ProjectID:      "test-project-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			input:         "Y",
			expectedError: true,
		},
		{
			name: "Error - Invalid project ID",
			config: globalconfig.GlobalConfig{
				OrganizationID: "test-org-id",
				ProjectID:      "invalid-project-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			input:         "Y",
			expectedError: true,
		},
		{
			name: "Error - No organization selected",
			config: globalconfig.GlobalConfig{
				ProjectID: "test-project-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			expectedError: false,
		},
		{
			name: "Error - No project selected",
			config: globalconfig.GlobalConfig{
				OrganizationID: "test-org-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			input:         "Y",
			expectedError: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := globalconfig.SaveGlobalConfig(tc.config)
			s.Require().NoError(err)

			getInput = func() (string, error) {
				return tc.input, nil
			}
			defer func() { getInput = originalGetInput }()

			err = deployment(s.ctx, "deploy")
			if tc.expectedError {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *ForgeTestSuite) TestStatus() {
	testCases := []struct {
		name          string
		config        globalconfig.GlobalConfig
		expectedError bool
	}{
		{
			name: "Success - Valid deployment",
			config: globalconfig.GlobalConfig{
				ProjectID: "test-project-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			expectedError: false,
		},
		{
			name: "Success - Valid undeployed project",
			config: globalconfig.GlobalConfig{
				ProjectID: "undeployed-project-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			expectedError: false,
		},
		{
			name: "Success - Valid failed build project",
			config: globalconfig.GlobalConfig{
				ProjectID: "failedbuild-project-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			expectedError: false,
		},
		{
			name: "Success - Valid destroyed project",
			config: globalconfig.GlobalConfig{
				ProjectID: "destroyed-project-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			expectedError: false,
		},
		{
			name: "Success - Valid reset project",
			config: globalconfig.GlobalConfig{
				ProjectID: "reset-project-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			expectedError: false,
		},
		{
			name: "Error - Invalid project ID",
			config: globalconfig.GlobalConfig{
				OrganizationID: "test-org-id",
				ProjectID:      "invalid-project-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			expectedError: true,
		},
		{
			name: "Error - No organization selected",
			config: globalconfig.GlobalConfig{
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			expectedError: false,
		},
		{
			name: "Error - No project selected",
			config: globalconfig.GlobalConfig{
				OrganizationID: "test-org-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			expectedError: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := globalconfig.SaveGlobalConfig(tc.config)
			s.Require().NoError(err)

			err = status(s.ctx)
			if tc.expectedError {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *ForgeTestSuite) TestDestroy() {
	testCases := []struct {
		name          string
		config        globalconfig.GlobalConfig
		input         string // Simulated user input for confirmation
		expectedError bool
	}{
		{
			name: "Success - Valid destroy with confirmation",
			config: globalconfig.GlobalConfig{
				OrganizationID: "test-org-id",
				ProjectID:      "test-project-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			input:         "y",
			expectedError: false,
		},
		{
			name: "Success - Cancelled destroy",
			config: globalconfig.GlobalConfig{
				OrganizationID: "test-org-id",
				ProjectID:      "test-project-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			input:         "n",
			expectedError: false,
		},
		{
			name: "Error - Invalid organization ID",
			config: globalconfig.GlobalConfig{
				OrganizationID: "invalid-org-id",
				ProjectID:      "test-project-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			input:         "y",
			expectedError: true,
		},
		{
			name: "Error - No organization selected",
			config: globalconfig.GlobalConfig{
				ProjectID: "test-project-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			input:         "y",
			expectedError: false,
		},
		{
			name: "Error - No project selected",
			config: globalconfig.GlobalConfig{
				OrganizationID: "test-org-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			input:         "y",
			expectedError: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Setup test config
			err := globalconfig.SaveGlobalConfig(tc.config)
			s.Require().NoError(err)

			getInput = func() (string, error) {
				return tc.input, nil
			}
			defer func() { getInput = originalGetInput }()

			err = deployment(s.ctx, "destroy")
			if tc.expectedError {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *ForgeTestSuite) TestReset() {
	testCases := []struct {
		name          string
		config        globalconfig.GlobalConfig
		input         string
		expectedError bool
	}{
		{
			name: "Success",
			config: globalconfig.GlobalConfig{
				OrganizationID: "test-org-id",
				ProjectID:      "test-project-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			input:         "y",
			expectedError: false,
		},
		{
			name: "Error - Invalid organization ID",
			config: globalconfig.GlobalConfig{
				OrganizationID: "invalid-org-id",
				ProjectID:      "test-project-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			input:         "y",
			expectedError: true,
		},
		{
			name: "Error - Invalid project ID",
			config: globalconfig.GlobalConfig{
				OrganizationID: "test-org-id",
				ProjectID:      "invalid-project-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			input:         "y",
			expectedError: true,
		},
		{
			name: "Error - No organization selected",
			config: globalconfig.GlobalConfig{
				ProjectID: "test-project-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			input:         "y",
			expectedError: false,
		},
		{
			name: "Error - No project selected",
			config: globalconfig.GlobalConfig{
				OrganizationID: "test-org-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			input:         "y",
			expectedError: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Setup test config
			err := globalconfig.SaveGlobalConfig(tc.config)
			s.Require().NoError(err)

			getInput = func() (string, error) {
				return tc.input, nil
			}
			defer func() { getInput = originalGetInput }()

			err = deployment(s.ctx, "reset")
			if tc.expectedError {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *ForgeTestSuite) TestParseResponse() {
	testCases := []struct {
		name          string
		input         []byte
		expectedError bool
		expectedData  string
	}{
		{
			name:          "Success - Valid JSON response",
			input:         []byte(`{"data": "test data"}`),
			expectedError: false,
			expectedData:  "test data",
		},
		{
			name:          "Error - Invalid JSON",
			input:         []byte(`{"data": invalid}`),
			expectedError: true,
			expectedData:  "",
		},
		{
			name:          "Error - Missing data field",
			input:         []byte(`{"other": "value"}`),
			expectedError: true,
			expectedData:  "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result, err := parseResponse[string](tc.input)
			if tc.expectedError {
				s.Require().Error(err)
				s.Nil(result)
			} else {
				s.Require().NoError(err)
				s.Equal(tc.expectedData, *result)
			}
		})
	}
}

func (s *ForgeTestSuite) TestValidateRepoPath() {
	testCases := []struct {
		name          string
		path          string
		expectedError bool
	}{
		{name: "Good Path",
			path:          "rampage",
			expectedError: false,
		},
		{name: "Bad Path",
			path:          "spaces not allowed",
			expectedError: true,
		},
		{name: "Empty Path",
			path:          "",
			expectedError: false,
		},
		{name: "Multilevel Path",
			path:          "/this/path/is/fine",
			expectedError: false,
		},
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := validateRepoPath(s.ctx, "fake_repo_url", "fake_token", tc.path)
			if tc.expectedError {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *ForgeTestSuite) TestValidateRepoToken() {
	testCases := []struct {
		name          string
		repoURL       string
		token         string
		expectedError bool
	}{
		{
			name:          "Success - Valid GitHub repo and token",
			repoURL:       "https://github.com/Argus-Labs/starter-game-template",
			token:         "",
			expectedError: false,
		},
		{
			name:          "Success - Valid GitLab repo and token",
			repoURL:       "https://gitlab.com/gitlab-org/gitlab.git",
			token:         "",
			expectedError: false,
		},
		{
			name:          "Success - Valid Bitbucket repo and token",
			repoURL:       "https://bitbucket.org/fargo3d/public.git",
			token:         "",
			expectedError: false,
		},
		{
			name:          "Error - Invalid repo URL",
			repoURL:       "invalid-url",
			token:         "valid-token",
			expectedError: true,
		},
		{
			name:          "Error - Unsupported provider",
			repoURL:       "https://unknown.com/test/repo",
			token:         "valid-token",
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := validateRepoToken(s.ctx, tc.repoURL, tc.token)
			if tc.expectedError {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *ForgeTestSuite) TestLogin() {
	testCases := []struct {
		name          string
		key           string
		expectedError bool
	}{
		{
			name:          "Success - Valid login flow",
			key:           "valid-key",
			expectedError: false,
		},
		{
			name:          "Error - Invalid key",
			key:           "invalid-key",
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Mock key generation
			generateKey = func() string { return tc.key }
			defer func() { generateKey = originalGenerateKey }()

			// Mock browser opening
			openBrowser = func(_ string) error { return nil }
			defer func() { openBrowser = originalOpenBrowser }()

			err := login(s.ctx)
			if tc.expectedError {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *ForgeTestSuite) TestGetListOfProjects() {
	testCases := []struct {
		name          string
		config        globalconfig.GlobalConfig
		expectedError bool
		expectedLen   int
	}{
		{
			name: "Success - Valid organization with projects",
			config: globalconfig.GlobalConfig{
				OrganizationID: "test-org-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			expectedError: false,
			expectedLen:   1,
		},
		{
			name: "Error - Invalid organization ID",
			config: globalconfig.GlobalConfig{
				OrganizationID: "invalid-org-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			expectedError: true,
			expectedLen:   0,
		},
		{
			name:          "Error - No organization selected",
			config:        globalconfig.GlobalConfig{},
			expectedError: false,
			expectedLen:   0,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := globalconfig.SaveGlobalConfig(tc.config)
			s.Require().NoError(err)

			projects, err := getListOfProjects(s.ctx)
			if tc.expectedError {
				s.Require().Error(err)
				s.Empty(projects)
			} else {
				s.Require().NoError(err)
				s.Len(projects, tc.expectedLen)
			}
		})
	}
}

func (s *ForgeTestSuite) TestOrganizationOperations() {
	testCases := []struct {
		name          string
		operation     string // "list", "get", "select"
		config        globalconfig.GlobalConfig
		input         string // for select operation
		expectedError bool
		expectedOrgs  int // for list operation
	}{
		{
			name:      "Success - List organizations",
			operation: "list",
			config: globalconfig.GlobalConfig{
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			expectedError: false,
			expectedOrgs:  1,
		},
		{
			name:      "Success - Get selected organization",
			operation: "get",
			config: globalconfig.GlobalConfig{
				OrganizationID: "test-org-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			expectedError: false,
		},
		{
			name:      "Success - Select organization",
			operation: "select",
			config: globalconfig.GlobalConfig{
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			input:         "1",
			expectedError: false,
		},
		{
			name:      "Error - Get invalid organization",
			operation: "get",
			config: globalconfig.GlobalConfig{
				OrganizationID: "invalid-org-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			expectedError: true,
		},
		{
			name:      "Error - Select invalid option",
			operation: "select",
			config: globalconfig.GlobalConfig{
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			input:         "99",
			expectedError: true,
		},
		{
			name:      "Error - Select cancelled",
			operation: "select",
			config: globalconfig.GlobalConfig{
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			input:         "q",
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Setup test config
			err := globalconfig.SaveGlobalConfig(tc.config)
			s.Require().NoError(err)

			switch tc.operation {
			case "list":
				orgs, err := getListOfOrganizations(s.ctx)
				if tc.expectedError {
					s.Require().Error(err)
				} else {
					s.Require().NoError(err)
					s.Len(orgs, tc.expectedOrgs)
				}

			case "get":
				org, err := getSelectedOrganization(s.ctx)
				if tc.expectedError {
					s.Require().Error(err)
				} else {
					s.Require().NoError(err)
					s.Equal("test-org-id", org.ID)
					s.Equal("Test Org", org.Name)
					s.Equal("testo", org.Slug)
				}

			case "select":
				getInput = func() (string, error) {
					return tc.input, nil
				}
				defer func() { getInput = originalGetInput }()

				org, err := selectOrganization(s.ctx)
				if tc.expectedError {
					s.Require().Error(err)
					s.Empty(org)
				} else {
					s.Require().NoError(err)
					s.Equal("test-org-id", org.ID)
					s.Equal("Test Org", org.Name)
					s.Equal("testo", org.Slug)
				}
			}
		})
	}
}

func (s *ForgeTestSuite) TestCreateOrganization() {
	testCases := []struct {
		name          string
		input         string
		expectedError bool
		expectedOrg   *organization
	}{
		{
			name:          "Success - Valid organization",
			input:         "testo",
			expectedError: false,
			expectedOrg: &organization{
				ID:   "test-org-id",
				Name: "Test Organization",
				Slug: "testo",
			},
		},
		{
			name:          "Error - Invalid slug length",
			input:         "testorgs12345678910",
			expectedError: true,
			expectedOrg:   nil,
		},
		{
			name:          "Error - Non-alphanumeric dots dash underscore slug",
			input:         "te_st()",
			expectedError: true,
			expectedOrg:   nil,
		},
		{
			name:          "Error - Empty name",
			input:         "",
			expectedError: true,
			expectedOrg:   nil,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			getInput = func() (string, error) {
				return tc.input, nil
			}
			defer func() { getInput = originalGetInput }()

			org, err := createOrganization(s.ctx)
			if tc.expectedError {
				s.Require().Error(err)
				s.Empty(org)
			} else {
				s.Require().NoError(err)
				s.Equal(tc.expectedOrg.Name, org.Name)
				s.Equal(tc.expectedOrg.Slug, org.Slug)
			}
		})
	}
}

func (s *ForgeTestSuite) TestShowOrganizationList() {
	testCases := []struct {
		name          string
		config        globalconfig.GlobalConfig
		expectedError bool
	}{
		{
			name: "Success - Show organization list with selected org",
			config: globalconfig.GlobalConfig{
				OrganizationID: "test-org-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			expectedError: false,
		},
		{
			name: "Success - Show organization list without selected org",
			config: globalconfig.GlobalConfig{
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			expectedError: false,
		},
		{
			name: "Error - Invalid organization ID",
			config: globalconfig.GlobalConfig{
				OrganizationID: "invalid-org-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := globalconfig.SaveGlobalConfig(tc.config)
			s.Require().NoError(err)

			err = showOrganizationList(s.ctx)
			if tc.expectedError {
				s.Error(err)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *ForgeTestSuite) TestShowProjectList() {
	testCases := []struct {
		name          string
		config        globalconfig.GlobalConfig
		expectedError bool
	}{
		{
			name: "Success - Show project list with selected project",
			config: globalconfig.GlobalConfig{
				OrganizationID: "test-org-id",
				ProjectID:      "test-project-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			expectedError: false,
		},
		{
			name: "Success - Show project list without selected project",
			config: globalconfig.GlobalConfig{
				OrganizationID: "test-org-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			expectedError: false,
		},
		{
			name: "Success - Empty project list",
			config: globalconfig.GlobalConfig{
				OrganizationID: "empty-org-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			expectedError: false,
		},
		{
			name: "Error - Invalid organization ID",
			config: globalconfig.GlobalConfig{
				OrganizationID: "invalid-org-id",
				ProjectID:      "test-project-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			expectedError: true,
		},
		{
			name: "Error - Invalid project ID",
			config: globalconfig.GlobalConfig{
				OrganizationID: "test-org-id",
				ProjectID:      "invalid-project-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := globalconfig.SaveGlobalConfig(tc.config)
			s.Require().NoError(err)

			err = showProjectList(s.ctx)
			if tc.expectedError {
				s.Error(err)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *ForgeTestSuite) TestCreateProject() { //nolint:gocognit
	testCases := []struct {
		name                string
		config              globalconfig.GlobalConfig
		inputs              []string     // For name, slug, repoURL, repoToken
		regionSelectActions []tea.KeyMsg // Simulate region selection
		expectedError       bool
		expectedProject     *project
	}{
		{
			name: "Success - Create project with public repo",
			config: globalconfig.GlobalConfig{
				OrganizationID: "test-org-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			inputs: []string{
				"Test Project", // name
				"testp",        // slug
				"https://github.com/argus-labs/starter-game-template", // repoURL
				"",           // repoToken (empty for public repo)
				"",           // repoPath (empty for default root path of repo)
				"10",         // tick rate
				"Y",          // enable discord notifications
				"test-token", // discord token
				"1234567890", // discord channel ID
				"Y",          // enable slack notifications
				"test-token", // slack token
				"1234567890", // slack channel ID
			},
			regionSelectActions: []tea.KeyMsg{
				tea.KeyMsg{Type: tea.KeySpace}, // select region
				tea.KeyMsg{Type: tea.KeyEnter}, // confirm
			},
			expectedError: false,
			expectedProject: &project{
				Name: "Test Project",
				Slug: "testp",
			},
		},
		// {
		// 	name: "Success - Create project with private repo",
		// 	config: globalconfig.GlobalConfig{
		// 		OrganizationID: "test-org-id",
		// 		Credential: globalconfig.Credential{
		// 			Token: "test-token",
		// 		},
		// 	},
		// 	inputs: []string{
		// 		"Test Private",
		// 		"testp",
		// 		"https://github.com/test/private-repo",
		// 		"secret-token",
		// 	},
		// 	expectedError: false,
		// },
		{
			name: "Error - No organization selected",
			config: globalconfig.GlobalConfig{
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			inputs:          []string{},
			expectedError:   true,
			expectedProject: nil,
		},
		{
			name: "Error - Invalid organization ID",
			config: globalconfig.GlobalConfig{
				OrganizationID: "invalid-org-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			inputs: []string{
				"Test Project",
				"testp",
				"https://github.com/test/repo",
				"",
			},
			expectedError:   true,
			expectedProject: nil,
		},
		{
			name: "Error - Invalid project slug",
			config: globalconfig.GlobalConfig{
				OrganizationID: "test-org-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			inputs: []string{
				"Test Project",
				"te",    // invalid slug 1st attempts
				"te.st", // invalid slug 2nd attempts
				"",      // invalid slug 3rd attempts
				"a---a", // invalid slug 4th attempts
				"()@",   // invalid slug 5th attempts
				"https://github.com/test/repo",
				"",
			},
			expectedError:   true,
			expectedProject: nil,
		},
		{
			name: "Error - Invalid repo URL",
			config: globalconfig.GlobalConfig{
				OrganizationID: "test-org-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			inputs: []string{
				"Test Project",
				"testp",
				"invalid-url",   // invalid URL 1st attempts
				"invalid-token", // invalid token 1st attempts
				"invalid-url",   // invalid URL 2nd attempts
				"invalid-token", // invalid token 2nd attempts
				"invalid-url",   // invalid URL 3rd attempts
				"invalid-token", // invalid token 3rd attempts
				"invalid-url",   // invalid URL 4th attempts
				"invalid-token", // invalid token 4th attempts
				"invalid-url",   // invalid URL 5th attempts
				"invalid-token", // invalid token 5th attempts
				"",
			},
			expectedError:   true,
			expectedProject: nil,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Setup test config
			err := globalconfig.SaveGlobalConfig(tc.config)
			s.Require().NoError(err)

			if len(tc.inputs) > 0 {
				inputIndex := 0
				getInput = func() (string, error) {
					input := tc.inputs[inputIndex]
					inputIndex++
					return input, nil
				}
				defer func() { getInput = originalGetInput }()
			}

			// Simulate region selection
			regionSelector = tea.NewProgram(
				multiselect.InitialMultiselectModel(
					s.ctx,
					[]string{"us-east-1", "us-west-1", "eu-west-1"},
				),
				tea.WithInput(nil),
			)
			if regionSelector == nil {
				print("failed to create region selector")
			}
			defer func() { regionSelector = nil }()

			// Send region select actions
			go func() {
				// wait for 1s to make sure the program is initialized
				time.Sleep(1 * time.Second)
				for _, action := range tc.regionSelectActions {
					// send action to region selector
					if regionSelector != nil {
						regionSelector.Send(action)
						// wait for 100ms to make sure the action is processed
						time.Sleep(100 * time.Millisecond)
					} else {
						print("region selector is nil")
					}
				}
			}()

			prj, err := createProject(s.ctx)
			if tc.expectedError {
				s.Error(err)
			} else {
				s.Require().NoError(err)
				s.Require().NotNil(prj)
				if tc.expectedProject != nil && prj != nil {
					s.Equal(tc.expectedProject.Name, prj.Name)
					s.Equal(tc.expectedProject.Slug, prj.Slug)
				}
			}
		})
	}
}

func (s *ForgeTestSuite) TestSelectProject() {
	testCases := []struct {
		name          string
		config        globalconfig.GlobalConfig
		input         string
		expectedError bool
		expectedProj  *project
	}{
		{
			name: "Success - Valid project selection",
			config: globalconfig.GlobalConfig{
				OrganizationID: "test-org-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			input:         "1",
			expectedError: false,
			expectedProj: &project{
				ID:      "test-project-id",
				OrgID:   "test-org-id",
				Name:    "Test Project",
				Slug:    "testp",
				RepoURL: "https://github.com/test/repo",
			},
		},
		{
			name: "Success - Cancel selection with 'q'",
			config: globalconfig.GlobalConfig{
				OrganizationID: "test-org-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			input:         "q",
			expectedError: true,
			expectedProj:  nil,
		},
		{
			name: "Error - Empty project list",
			config: globalconfig.GlobalConfig{
				OrganizationID: "empty-org-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			input:         "1",
			expectedError: false,
			expectedProj:  nil,
		},
		{
			name: "Error - Invalid selection number",
			config: globalconfig.GlobalConfig{
				OrganizationID: "test-org-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			input:         "99",
			expectedError: true,
			expectedProj:  nil,
		},
		{
			name: "Error - Max attempts reached",
			config: globalconfig.GlobalConfig{
				OrganizationID: "test-org-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			input:         "invalid\ninvalid\ninvalid\ninvalid\ninvalid\n",
			expectedError: true,
			expectedProj:  nil,
		},
		{
			name: "Error - Invalid organization ID",
			config: globalconfig.GlobalConfig{
				OrganizationID: "invalid-org-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			input:         "1",
			expectedError: true,
			expectedProj:  nil,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := globalconfig.SaveGlobalConfig(tc.config)
			s.Require().NoError(err)

			getInput = func() (string, error) {
				return tc.input, nil
			}
			defer func() { getInput = originalGetInput }()

			proj, err := selectProject(s.ctx)
			if tc.expectedError {
				s.Require().Error(err)
				s.Empty(proj)
			} else {
				if tc.expectedProj == nil {
					s.Empty(proj)
				} else {
					s.Require().NoError(err)
					s.Equal(tc.expectedProj.ID, proj.ID)
					s.Equal(tc.expectedProj.Name, proj.Name)
					s.Equal(tc.expectedProj.Slug, proj.Slug)
				}
			}
		})
	}
}

func (s *ForgeTestSuite) TestGetRoleInput() {
	testCases := []struct {
		name          string
		input         string
		allowNone     bool
		expectedInput string
		expectError   bool
	}{
		{
			name:          "Default Role is member",
			input:         "\n",
			allowNone:     true,
			expectedInput: "member",
			expectError:   false,
		},
		{
			name:          "Admin",
			input:         "admin\n",
			allowNone:     true,
			expectedInput: "admin",
			expectError:   false,
		},
		{
			name:          "Owner",
			input:         "owner\n",
			allowNone:     true,
			expectedInput: "owner",
			expectError:   false,
		},
		{
			name:          "Member",
			input:         "member\n",
			allowNone:     true,
			expectedInput: "member",
			expectError:   false,
		},
		{
			name:          "None enabled",
			input:         "none\nYes\n",
			allowNone:     true,
			expectedInput: "none",
			expectError:   true, // can't actually test this because it's two separate inputs
		},
		{
			name:          "None disabled",
			input:         "none\n",
			allowNone:     false,
			expectedInput: "",
			expectError:   true,
		},
		{
			name:          "Bad Role",
			input:         "garbage\n",
			allowNone:     true,
			expectedInput: "",
			expectError:   true,
		},
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Create a temporary file for stdin
			tmpfile, err := os.CreateTemp("", "stdin")
			s.Require().NoError(err)
			defer os.Remove(tmpfile.Name())

			_, err = tmpfile.WriteString(tc.input)
			s.Require().NoError(err)

			_, err = tmpfile.Seek(0, 0)
			s.Require().NoError(err)

			// Save original stdin
			oldStdin := os.Stdin
			defer func() { os.Stdin = oldStdin }()

			// Set stdin to our test file
			os.Stdin = tmpfile

			// Test getInput
			result, err := getRoleInput(tc.allowNone)
			if tc.expectError {
				s.Require().Error(err)
				s.Empty(result)
			} else {
				s.Require().NoError(err)
				s.Equal(tc.expectedInput, result)
			}
		})
	}
}

func (s *ForgeTestSuite) TestGetInput() {
	testCases := []struct {
		name          string
		input         string
		expectedInput string
		expectError   bool
	}{
		{
			name:          "Success - Normal input",
			input:         "test input\n",
			expectedInput: "test input",
			expectError:   false,
		},
		{
			name:          "Success - Input with whitespace",
			input:         "  test input  \n",
			expectedInput: "test input",
			expectError:   false,
		},
		{
			name:          "Success - Empty input",
			input:         "\n",
			expectedInput: "",
			expectError:   false,
		},
		{
			name:          "Success - Input with multiple lines",
			input:         "test\ninput\n",
			expectedInput: "test",
			expectError:   false,
		},
		{
			name:          "Error - No newline",
			input:         "test input",
			expectedInput: "",
			expectError:   true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Create a temporary file for stdin
			tmpfile, err := os.CreateTemp("", "stdin")
			s.Require().NoError(err)
			defer os.Remove(tmpfile.Name())

			_, err = tmpfile.WriteString(tc.input)
			s.Require().NoError(err)

			_, err = tmpfile.Seek(0, 0)
			s.Require().NoError(err)

			// Save original stdin
			oldStdin := os.Stdin
			defer func() { os.Stdin = oldStdin }()

			// Set stdin to our test file
			os.Stdin = tmpfile

			// Test getInput
			result, err := getInput()
			if tc.expectError {
				s.Require().Error(err)
				s.Empty(result)
			} else {
				s.Require().NoError(err)
				s.Equal(tc.expectedInput, result)
			}
		})
	}
}

func (s *ForgeTestSuite) TestInviteUserToOrganization() {
	testCases := []struct {
		name          string
		config        globalconfig.GlobalConfig
		inputs        []string // For user id, role
		expectedError bool
	}{
		{
			name: "Success - Default role",
			config: globalconfig.GlobalConfig{
				OrganizationID: "test-org-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			inputs: []string{
				"test-user-id", // user-id
				"",             // role
			},
			expectedError: false,
		},
		{
			name: "Success - admin role",
			config: globalconfig.GlobalConfig{
				OrganizationID: "test-org-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			inputs: []string{
				"test-user-id", // user-id
				"admin",        // role
			},
			expectedError: false,
		},
		{
			name: "Error - No organization selected",
			config: globalconfig.GlobalConfig{
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			inputs:        []string{},
			expectedError: true,
		},
		{
			name: "Error - Invalid organization ID",
			config: globalconfig.GlobalConfig{
				OrganizationID: "invalid-org-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			inputs: []string{
				"test-user-id", // user-id
				"",             // role
			},
			expectedError: true,
		},
		{
			name: "Error - Invalid Role: None",
			config: globalconfig.GlobalConfig{
				OrganizationID: "test-org-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			inputs: []string{
				"test-user-id", // user-id
				"none",         // invalid role
				"none",
				"none",
				"none",
				"none",
			},
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Setup test config
			err := globalconfig.SaveGlobalConfig(tc.config)
			s.Require().NoError(err)

			if len(tc.inputs) > 0 {
				inputIndex := 0
				getInput = func() (string, error) {
					input := tc.inputs[inputIndex]
					inputIndex++
					return input, nil
				}
				defer func() { getInput = originalGetInput }()
			}
			defer func() { regionSelector = nil }()

			err = inviteUserToOrganization(s.ctx)
			if tc.expectedError {
				s.Error(err)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *ForgeTestSuite) TestUpdateRoleInOrganization() {
	testCases := []struct {
		name          string
		config        globalconfig.GlobalConfig
		inputs        []string // For user id, role
		expectedError bool
	}{
		{
			name: "Success - Default role",
			config: globalconfig.GlobalConfig{
				OrganizationID: "test-org-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			inputs: []string{
				"test-user-id", // user-id
				"",             // role
			},
			expectedError: false,
		},
		{
			name: "Success - admin role",
			config: globalconfig.GlobalConfig{
				OrganizationID: "test-org-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			inputs: []string{
				"test-user-id", // user-id
				"admin",        // role
			},
			expectedError: false,
		},
		{
			name: "Success - none with confirm remove",
			config: globalconfig.GlobalConfig{
				OrganizationID: "test-org-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			inputs: []string{
				"test-user-id", // user-id
				"none",         // role
				"Yes",          // confirm removal
			},
			expectedError: false,
		},
		{
			name: "Error - No organization selected",
			config: globalconfig.GlobalConfig{
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			inputs:        []string{},
			expectedError: true,
		},
		{
			name: "Error - Invalid organization ID",
			config: globalconfig.GlobalConfig{
				OrganizationID: "invalid-org-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			inputs: []string{
				"test-user-id", // user-id
				"",             // role
			},
			expectedError: true,
		},
		{
			name: "Error - Role none dont confirm remove",
			config: globalconfig.GlobalConfig{
				OrganizationID: "test-org-id",
				Credential: globalconfig.Credential{
					Token: "test-token",
				},
			},
			inputs: []string{
				"test-user-id", // user-id
				"none",         // invalid role
				"no",
				"none",
				"",
				"none",
				"bah",
				"none",
				"NO",
				"none",
				"y",
			},
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Setup test config
			err := globalconfig.SaveGlobalConfig(tc.config)
			s.Require().NoError(err)

			if len(tc.inputs) > 0 {
				inputIndex := 0
				getInput = func() (string, error) {
					input := tc.inputs[inputIndex]
					inputIndex++
					return input, nil
				}
				defer func() { getInput = originalGetInput }()
			}
			defer func() { regionSelector = nil }()

			err = updateUserRoleInOrganization(s.ctx)
			if tc.expectedError {
				s.Error(err)
			} else {
				s.NoError(err)
			}
		})
	}
}

func TestForgeSuite(t *testing.T) {
	suite.Run(t, new(ForgeTestSuite))
}
