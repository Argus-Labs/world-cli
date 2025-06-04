package forge

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rotisserie/eris"
	"github.com/stretchr/testify/suite"
	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/printer"
	"pkg.world.dev/world-cli/tea/component/multiselect"
)

var (
	originalGenerateKey = generateKey
	originalOpenBrowser = openBrowser
	originalGetInput    = getInput
	tempDir             string
	knownProjects       = []KnownProject{
		{
			ProjectID:      "test-project-id",
			RepoURL:        "https://github.com/Argus-Labs/world-cli",
			RepoPath:       "cmd/world/forge",
			OrganizationID: "test-org-id",
		},
	}
)

type ForgeTestSuite struct {
	suite.Suite
	server    *httptest.Server
	testToken string
	ctx       context.Context
}

func (s *ForgeTestSuite) SetupTest() { //nolint: cyclop, gocyclo // test, don't care about cylomatic complexity
	s.ctx = context.Background()

	argusIDAuthURL = "http://localhost:8001/api/auth/service-auth-session"

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
				case "/api/user":
					s.handleGetUser(w, r)
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
					s.handleHealth(w, r, "test-project-id")
				case "/api/health/reset-project-id":
					s.handleHealth(w, r, "reset-project-id")
				case "/api/health/destroyed-project-id":
					s.handleHealth(w, r, "destroyed-project-id")
				case "/api/project/":
					s.handleProjectLookup(w, r)
				case "/api/auth/service-auth-session":
					s.handleArgusIDAuthSession(w, r)
				case "/api/organization/test-org-id/project/test-project-id/testp/check_slug":
					s.handleProjectSlugCheck(w, r)
				case "/api/organization/test-org-id/project/00000000-0000-0000-0000-000000000000/testp/check_slug":
					s.handleProjectSlugCheck(w, r)
				default:
					http.Error(w, "Not found", http.StatusNotFound)
				}
			}),
		},
	}
	s.server.Start()

	// Set max attempts to 3 for login tests
	maxLoginAttempts = 3

	// Ensures that config saves are in a temp dir
	tempDir = filepath.Join(os.TempDir(), "worldcli")
	//nolint:reassign // Might cause issues with parallel tests
	config.GetCLIConfigDir = func() (string, error) {
		return tempDir, nil
	}
	err = config.SetupCLIConfigDir()
	s.Require().NoError(err)

	// Create config file
	err = Config{
		OrganizationID: "test-org-id",
		ProjectID:      "test-project-id",
		Credential: Credential{
			Token: "test-token",
		},
	}.Save()
	s.Require().NoError(err)
}

func (s *ForgeTestSuite) TearDownTest() {
	s.server.Close()

	// Remove temp config dir
	os.RemoveAll(tempDir)
}

func (s *ForgeTestSuite) handleGetUser(w http.ResponseWriter, _ *http.Request) {
	s.writeJSON(w, map[string]interface{}{"data": User{
		ID:        "test-user-id",
		Name:      "Test User",
		Email:     "test@example.com",
		AvatarURL: "https://example.com/avatar.png",
	}})
}

func (s *ForgeTestSuite) handleArgusIDAuthSession(w http.ResponseWriter, _ *http.Request) {
	s.writeJSON(w, map[string]string{
		"callbackUrl": "http://localhost:8001/api/user/login/get-token?key=" + generateKey(),
		"clientUrl":   "http://localhost:8001/api/user/login",
	})
}

func (s *ForgeTestSuite) handleInvite(w http.ResponseWriter, _ *http.Request) {
	s.writeJSON(w, map[string]interface{}{"data": ""})
}

func (s *ForgeTestSuite) handleRole(w http.ResponseWriter, _ *http.Request) {
	s.writeJSON(w, map[string]interface{}{"data": ""})
}

func (s *ForgeTestSuite) handleProjectLookup(w http.ResponseWriter, _ *http.Request) {
	s.writeJSON(w, map[string]interface{}{"success": "true"})
}

func (s *ForgeTestSuite) handleOrganizationList(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		switch s.testToken {
		case "empty-list":
			// Return empty list for no orgs test case
			s.writeJSON(w, map[string]interface{}{"data": []organization{}})
		case "multiple-orgs":
			// Return multiple orgs for multiple orgs test case
			orgs := []organization{
				{
					ID:   "test-org-id-1",
					Name: "Test Org 1",
					Slug: "testo1",
				},
				{
					ID:   "test-org-id-2",
					Name: "Test Org 2",
					Slug: "testo2",
				},
			}
			s.writeJSON(w, map[string]interface{}{"data": orgs})
		default:
			// Default case - single org
			orgs := []organization{
				{
					ID:   "test-org-id",
					Name: "Test Org",
					Slug: "testo",
				},
			}
			s.writeJSON(w, map[string]interface{}{"data": orgs})
		}
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
		s.NoError(err)
		defer r.Body.Close()

		body := map[string]interface{}{}
		err = json.Unmarshal(parsedBody, &body)
		s.NoError(err)

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

	switch s.testToken {
	case "empty-list":
		// Return empty list for no projects test case
		s.writeJSON(w, map[string]interface{}{"data": []project{}})
	case "multiple-projects":
		// Return multiple projects for multiple projects test case
		projects := []project{
			{
				ID:      "test-project-id-1",
				OrgID:   "test-org-id",
				Name:    "Test Project 1",
				Slug:    "testp1",
				RepoURL: "https://github.com/test/repo1",
			},
			{
				ID:      "test-project-id-2",
				OrgID:   "test-org-id",
				Name:    "Test Project 2",
				Slug:    "testp2",
				RepoURL: "https://github.com/test/repo2",
			},
		}
		s.writeJSON(w, map[string]interface{}{"data": projects})
	default:
		// Default case - single project
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
		//nolint:govet // test
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

func (s *ForgeTestSuite) handleHealth(w http.ResponseWriter, r *http.Request, projectID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	resultCodeStr := ""
	resultStr := ""
	ok := false
	switch projectID {
	case "test-project-id":
		resultCodeStr = "200"
		resultStr = "{}"
		ok = true
	case "reset-project-id":
		resultCodeStr = "200"
		resultStr = "{}"
		ok = true
	case "destroyed-project-id":
		resultCodeStr = "502"
		resultStr = "Bad Gateway"
		ok = false
	}
	s.writeJSONString(w, `{"data":{"dev":{"ok":`+strconv.FormatBool(ok)+`,
		"offline":`+strconv.FormatBool(!ok)+`,
		"deployed_instances":[
		{
			"region":"ap-southeast-1",
			"instance":1,
		"cardinal":{
			"url":"https://cardinal.apse-1.test.com/health",
			"ok":`+strconv.FormatBool(ok)+`,
            "result_code":`+resultCodeStr+`,
			"result_str":"`+resultStr+`"
		},
		"nakama":{
			"url":"https://nakama.apse-1.test.com/healthcheck",
			"ok":`+strconv.FormatBool(ok)+`,
            "result_code":`+resultCodeStr+`,
			"result_str":"`+resultStr+`"
		}
    },
    {
		"region":"us-east-1",
		"instance":1,
		"cardinal":{
			"url":"https://cardinal01.use-1.test.com/health",
			"ok":`+strconv.FormatBool(ok)+`,
            "result_code":`+resultCodeStr+`,
			"result_str":"`+resultStr+`"
		},
		"nakama":{
			"url":"https://nakama01.use-1.test.com/healthcheck",
			"ok":`+strconv.FormatBool(ok)+`,
            "result_code":`+resultCodeStr+`,
			"result_str":"`+resultStr+`"
		}
    },
    {
		"region":"us-east-1",
		"instance":2,
		"cardinal":{
			"url":"https://cardinal02.use-1.test.com/health",
			"ok":`+strconv.FormatBool(ok)+`,
            "result_code":`+resultCodeStr+`,
			"result_str":"`+resultStr+`"
		},
		"nakama":{
			"url":"https://nakama02.use1-1.test.com/healthcheck",
			"ok":`+strconv.FormatBool(ok)+`,
            "result_code":`+resultCodeStr+`,
			"result_str":"`+resultStr+`"
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
		//nolint:govet // test
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
		// Add exp claim for 1 hour from now
		exp := time.Now().Add(1 * time.Hour).Unix()
		claims := map[string]interface{}{
			"user_metadata": map[string]string{
				"name": "Test User",
				"sub":  "test-user-id",
			},
			"exp": exp,
		}
		claimsJSON, err := json.Marshal(claims)
		s.NoError(err)
		claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)
		token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9." + claimsB64 + ".sign"
		s.writeJSON(w, map[string]string{
			"status": "success",
			"jwt":    token,
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
		deploymentPreview := deploymentPreview{ //nolint:govet // test
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

func (s *ForgeTestSuite) handleProjectSlugCheck(w http.ResponseWriter, r *http.Request) {
	// Extract org ID and project ID from URL
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 6 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	slug := parts[len(parts)-2]
	if slug != "testp" {
		http.Error(w, "Project slug already exists", http.StatusConflict)
		return
	}

	// Return success for any other slug
	s.writeJSON(w, map[string]string{"status": "available"})
}

func (s *ForgeTestSuite) TestGetSelectedOrganization() {
	testCases := []struct {
		name          string
		fCtx          *ForgeContext
		expectedError bool
		expectedOrg   *organization
	}{
		{
			name: "Success - Valid organization",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					Credential: Credential{
						Token: "test-token",
					},
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
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "invalid-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			expectedError: true,
			expectedOrg:   nil,
		},
		{
			name:          "Error - No organization selected",
			fCtx:          &ForgeContext{Config: &Config{}},
			expectedError: false,
			expectedOrg:   nil,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.fCtx.Context = s.ctx

			org, err := getSelectedOrganization(*tc.fCtx)
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
		fCtx          *ForgeContext
		expectedError bool
		expectedProj  *project
	}{
		{
			name: "Success - Valid project",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					ProjectID:      "test-project-id",
					Credential: Credential{
						Token: "test-token",
					},
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
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					ProjectID:      "invalid-project-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			expectedError: true,
			expectedProj:  nil,
		},
		{
			name: "Error - No organization selected",
			fCtx: &ForgeContext{
				Config: &Config{
					ProjectID: "test-project-id",
				},
			},
			expectedError: false,
			expectedProj:  nil,
		},
		{
			name: "Error - No project selected",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
				},
			},
			expectedError: false,
			expectedProj:  nil,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.fCtx.Context = s.ctx

			proj, err := getSelectedProject(*tc.fCtx)
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
			}
		})
	}
}

func (s *ForgeTestSuite) TestDeploy() {
	testCases := []struct {
		name                string
		fCtx                *ForgeContext
		inputs              []string     // For name, slug, repoURL, repoToken
		regionSelectActions []tea.KeyMsg // Simulate region selection
		expectedError       bool
		createProject       bool
	}{
		{
			name: "Success - Valid deployment",
			fCtx: &ForgeContext{
				State: CommandState{
					Organization: &organization{
						ID: "test-org-id",
					},
					Project: &project{
						ID: "test-project-id",
					},
					User: &User{
						ID: "test-user-id",
					},
				},
				Config: &Config{},
			},
			inputs:        []string{"Y"},
			expectedError: false,
		},
		{
			name: "Error - Invalid organization ID",
			fCtx: &ForgeContext{
				State: CommandState{
					Organization: &organization{
						ID: "invalid-org-id",
					},
					Project: &project{
						ID: "test-project-id",
					},
					User: &User{
						ID: "test-user-id",
					},
				},
				Config: &Config{},
			},
			inputs:        []string{"Y"},
			expectedError: true,
		},
		{
			name: "Error - Invalid project ID",
			fCtx: &ForgeContext{
				State: CommandState{
					Organization: &organization{
						ID: "test-org-id",
					},
					Project: &project{
						ID: "invalid-project-id",
					},
					User: &User{
						ID: "test-user-id",
					},
				},
				Config: &Config{},
			},
			inputs:        []string{"Y"},
			expectedError: true,
		},
		{
			name: "Error - No organization selected",
			fCtx: &ForgeContext{
				State: CommandState{
					Project: &project{
						ID: "test-project-id",
					},
					User: &User{
						ID: "test-user-id",
					},
				},
				Config: &Config{},
			},
			expectedError: false,
		},
		{
			name:          "Success - No project selected (creates new project)",
			createProject: true,
			fCtx: &ForgeContext{
				State: CommandState{
					Organization: &organization{
						ID: "test-org-id",
					},
					User: &User{
						ID: "test-user-id",
					},
				},
				Config: &Config{
					OrganizationID: "test-org-id",
				},
			},
			inputs: []string{
				"Test Project", // Project name
				"testp",        // Project slug
				"https://github.com/argus-labs/starter-game-template", // Repository URL
				"",   // No token needed for public repo
				"",   // Default repo path
				"10", // Tick rate
				"n",  // No Discord
				"n",  // No Slack
				"",   // No avatar URL
			},
			regionSelectActions: []tea.KeyMsg{
				{Type: tea.KeySpace}, // select region
				{Type: tea.KeyEnter}, // confirm
			},
			expectedError: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.fCtx.Context = s.ctx

			if tc.createProject {
				// Create a temp directory for this test case
				tmpDir, err := os.MkdirTemp("", "forge-Deploy-create-test")
				s.Require().NoError(err)
				defer os.RemoveAll(tmpDir)

				// Save original working directory
				oldDir, err := os.Getwd()
				s.Require().NoError(err)
				defer os.Chdir(oldDir)

				// Change to temp directory
				err = os.Chdir(tmpDir)
				s.Require().NoError(err)

				// Create minimal World project structure
				worldDir := filepath.Join(tmpDir, ".world")
				err = os.MkdirAll(worldDir, 0755)
				s.Require().NoError(err)

				// Create cardinal directory
				cardinalDir := filepath.Join(tmpDir, "cardinal")
				err = os.MkdirAll(cardinalDir, 0755)
				s.Require().NoError(err)

				// Initialize git repository
				cmd := exec.Command("git", "init")
				err = cmd.Run()
				s.Require().NoError(err)

				// Add a remote for the repository - use the same URL as in the test inputs
				cmd = exec.Command(
					"git",
					"remote",
					"add",
					"origin",
					"https://github.com/argus-labs/starter-game-template",
				)
				err = cmd.Run()
				s.Require().NoError(err)

				// Create empty world.toml
				worldTomlPath := filepath.Join(tmpDir, "world.toml")
				err = os.WriteFile(worldTomlPath, []byte(""), 0644)
				s.Require().NoError(err)
			}

			inputIndex := 0
			getInput = func(prompt string, defaultVal string) string {
				if inputIndex >= len(tc.inputs) {
					return defaultVal
				}
				input := tc.inputs[inputIndex]
				inputIndex++
				printer.Infof("%s [%s]: %s", prompt, defaultVal, input)
				return input
			}
			defer func() { getInput = originalGetInput }()

			// Simulate region selection
			regionSelector = tea.NewProgram(
				multiselect.InitialMultiselectModel(
					s.ctx,
					[]string{"us-east-1", "us-west-1", "eu-west-1"},
				),
				tea.WithInput(nil),
			)
			if regionSelector == nil {
				print("failed to create region selector") //nolint:forbidigo // test
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
						print("region selector is nil") //nolint:forbidigo // test
					}
				}
			}()

			err := deployment(*tc.fCtx, DeploymentTypeDeploy)
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
		fCtx          *ForgeContext
		expectedError bool
	}{
		{
			name: "Success - Valid deployment",
			fCtx: &ForgeContext{
				State: CommandState{
					Project: &project{
						ID: "test-project-id",
					},
					Organization: &organization{
						ID:   "test-org-id",
						Name: "Test Org",
						Slug: "test-org",
					},
					User: &User{
						ID: "test-user-id",
					},
				},
				Config: &Config{},
			},
			expectedError: false,
		},
		{
			name: "Success - Valid undeployed project",
			fCtx: &ForgeContext{
				State: CommandState{
					Project: &project{
						ID: "undeployed-project-id",
					},
					Organization: &organization{
						ID:   "test-org-id",
						Name: "Test Org",
						Slug: "test-org",
					},
					User: &User{
						ID: "test-user-id",
					},
				},
				Config: &Config{},
			},
			expectedError: false,
		},
		{
			name: "Success - Valid failed build project",
			fCtx: &ForgeContext{
				State: CommandState{
					Project: &project{
						ID: "failedbuild-project-id",
					},
					Organization: &organization{
						ID:   "test-org-id",
						Name: "Test Org",
						Slug: "test-org",
					},
					User: &User{
						ID: "test-user-id",
					},
				},
				Config: &Config{},
			},
			expectedError: false,
		},
		{
			name: "Success - Valid destroyed project",
			fCtx: &ForgeContext{
				State: CommandState{
					Project: &project{
						ID: "destroyed-project-id",
					},
					Organization: &organization{
						ID:   "test-org-id",
						Name: "Test Org",
						Slug: "test-org",
					},
					User: &User{
						ID: "test-user-id",
					},
				},
				Config: &Config{},
			},
			expectedError: false,
		},
		{
			name: "Success - Valid reset project",
			fCtx: &ForgeContext{
				State: CommandState{
					Project: &project{
						ID: "reset-project-id",
					},
					Organization: &organization{
						ID:   "test-org-id",
						Name: "Test Org",
						Slug: "test-org",
					},
					User: &User{
						ID: "test-user-id",
					},
				},
				Config: &Config{},
			},
			expectedError: false,
		},
		{
			name: "Error - Invalid project ID",
			fCtx: &ForgeContext{
				State: CommandState{
					Project: &project{
						ID: "invalid-project-id",
					},
					Organization: &organization{
						ID:   "test-org-id",
						Name: "Test Org",
						Slug: "test-org",
					},
					User: &User{
						ID: "test-user-id",
					},
				},
				Config: &Config{},
			},
			expectedError: true,
		},
		{
			name: "Error - No organization selected",
			fCtx: &ForgeContext{
				State: CommandState{
					Project: &project{
						ID: "test-project-id",
					},
					User: &User{
						ID: "test-user-id",
					},
				},
				Config: &Config{},
			},
			expectedError: false,
		},
		{
			name: "Error - No project selected",
			fCtx: &ForgeContext{
				State: CommandState{
					Organization: &organization{
						ID:   "test-org-id",
						Name: "Test Org",
						Slug: "test-org",
					},
					User: &User{
						ID: "test-user-id",
					},
				},
				Config: &Config{},
			},
			expectedError: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.fCtx.Context = s.ctx
			err := status(*tc.fCtx)
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
		fCtx          *ForgeContext
		input         string // Simulated user input for confirmation
		expectedError bool
	}{
		{
			name: "Success - Valid destroy with confirmation",
			fCtx: &ForgeContext{
				State: CommandState{
					Organization: &organization{
						ID: "test-org-id",
					},
					Project: &project{
						ID: "test-project-id",
					},
					User: &User{
						ID: "test-user-id",
					},
				},
				Config: &Config{},
			},
			input:         "Y",
			expectedError: false,
		},
		{
			name: "Success - Cancelled destroy",
			fCtx: &ForgeContext{
				State: CommandState{
					Organization: &organization{
						ID: "test-org-id",
					},
					Project: &project{
						ID: "test-project-id",
					},
					User: &User{
						ID: "test-user-id",
					},
				},
				Config: &Config{},
			},
			input:         "n",
			expectedError: false,
		},
		{
			name: "Error - Invalid organization ID",
			fCtx: &ForgeContext{
				State: CommandState{
					Organization: &organization{
						ID: "invalid-org-id",
					},
					Project: &project{
						ID: "test-project-id",
					},
					User: &User{
						ID: "test-user-id",
					},
				},
				Config: &Config{},
			},
			input:         "Y",
			expectedError: true,
		},
		{
			name: "Error - No organization selected",
			fCtx: &ForgeContext{
				State: CommandState{
					Project: &project{
						ID: "test-project-id",
					},
					User: &User{
						ID: "test-user-id",
					},
				},
				Config: &Config{},
			},
			input:         "Y",
			expectedError: false,
		},
		/* { // FIXME: this test case is not working
			name: "Error - No project selected",
			state: &ForgeCommandState{
				Organization: &organization{
					ID: "test-org-id",
				},
				User: &User{
					ID: "test-user-id",
				},
			},
			input:         "Y",
			expectedError: false,
		},*/
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			getInput = func(prompt string, defaultVal string) string {
				printer.Infof("%s [%s]: %s", prompt, defaultVal, tc.input)
				return tc.input
			}
			defer func() { getInput = originalGetInput }()

			tc.fCtx.Context = s.ctx
			err := deployment(*tc.fCtx, DeploymentTypeDestroy)
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
		fCtx          *ForgeContext
		input         string
		expectedError bool
	}{
		{
			name: "Success",
			fCtx: &ForgeContext{
				State: CommandState{
					Organization: &organization{
						ID: "test-org-id",
					},
					Project: &project{
						ID: "test-project-id",
					},
					User: &User{
						ID: "test-user-id",
					},
				},
				Config: &Config{},
			},
			input:         "Y",
			expectedError: false,
		},
		{
			name: "Error - Invalid organization ID",
			fCtx: &ForgeContext{
				State: CommandState{
					Organization: &organization{
						ID: "invalid-org-id",
					},
					Project: &project{
						ID: "test-project-id",
					},
					User: &User{
						ID: "test-user-id",
					},
				},
				Config: &Config{},
			},
			input:         "Y",
			expectedError: true,
		},
		{
			name: "Error - Invalid project ID",
			fCtx: &ForgeContext{
				State: CommandState{
					Organization: &organization{
						ID: "test-org-id",
					},
					Project: &project{
						ID: "invalid-project-id",
					},
					User: &User{
						ID: "test-user-id",
					},
				},
				Config: &Config{},
			},
			input:         "Y",
			expectedError: true,
		},
		{
			name: "Error - No organization selected",
			fCtx: &ForgeContext{
				State: CommandState{
					Project: &project{
						ID: "test-project-id",
					},
					User: &User{
						ID: "test-user-id",
					},
				},
				Config: &Config{},
			},
			input:         "Y",
			expectedError: false,
		},
		/*{ // FIXME: this test case is not working
			name: "Error - No project selected",
			state: &ForgeCommandState{
				Organization: &organization{
					ID: "test-org-id",
				},
				User: &User{
					ID: "test-user-id",
				},
			},
			input:         "Y",
			expectedError: false,
		},*/
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			getInput = func(prompt string, defaultVal string) string {
				printer.Infof("%s [%s]: %s", prompt, defaultVal, tc.input)
				return tc.input
			}
			defer func() { getInput = originalGetInput }()

			tc.fCtx.Context = s.ctx
			err := deployment(*tc.fCtx, DeploymentTypeReset)
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
		fCtx          *ForgeContext
	}{
		{
			name:          "Success - Has selected org and project",
			key:           "valid-key",
			expectedError: false,
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID:  "test-org-id",
					ProjectID:       "test-project-id",
					CurrProjectName: "test-project",
					Credential: Credential{
						Token:          "test-token",
						TokenExpiresAt: time.Now().Add(1 * time.Hour),
					},
				},
			},
		},
		{
			name:          "Success - Has selected org only",
			key:           "valid-key",
			expectedError: false,
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					Credential: Credential{
						Token:          "test-token",
						TokenExpiresAt: time.Now().Add(1 * time.Hour),
					},
				},
			},
		},
		{
			name:          "Success - No proj or org selected",
			key:           "valid-key",
			expectedError: false,
			fCtx: &ForgeContext{
				Config: &Config{
					Credential: Credential{
						Token:          "test-token",
						TokenExpiresAt: time.Now().Add(1 * time.Hour),
					},
				},
			},
		},
		{
			name: "Error - Invalid key",
			key:  "invalid-key",
			fCtx: &ForgeContext{
				Config: &Config{},
			},
			expectedError: true,
		},
		{
			name:          "Success - Test inKnownRepo",
			key:           "valid-key",
			expectedError: false,
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID:  "test-org-id",
					ProjectID:       "test-project-id",
					CurrProjectName: "test-project",
					CurrRepoURL:     "https://github.com/test/repo",
					CurrRepoPath:    "cmd/world/forge",
					CurrRepoKnown:   true,
					Credential: Credential{
						Token:          "test-token",
						TokenExpiresAt: time.Now().Add(1 * time.Hour),
					},
				},
				State: CommandState{
					LoggedIn:      true,
					CurrRepoKnown: true,
				},
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.fCtx.Context = s.ctx
			// Set the test token for this case
			s.testToken = tc.key
			defer func() { s.testToken = "" }()

			// Mock key generation
			generateKey = func() string { return tc.key }
			defer func() { generateKey = originalGenerateKey }()

			// Mock browser opening
			openBrowser = func(_ string) error { return nil }
			defer func() { openBrowser = originalOpenBrowser }()

			// If this is the inKnownRepo test, run that test
			if tc.name == "Success - Test inKnownRepo" {
				flowTest := &initFlow{}

				result := flowTest.inKnownRepo(tc.fCtx)
				s.True(result)
				s.True(tc.fCtx.State.CurrRepoKnown)
				s.NotNil(tc.fCtx.State.Organization)
				s.Equal("test-org-id", tc.fCtx.State.Organization.ID)
				s.NotNil(tc.fCtx.State.Project)
				s.Equal("test-project-id", tc.fCtx.State.Project.ID)
				return
			}

			err := login(*tc.fCtx)
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
		fCtx          *ForgeContext
		expectedError bool
		expectedLen   int
	}{
		{
			name: "Success - Valid organization with projects",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			expectedError: false,
			expectedLen:   1,
		},
		{
			name: "Error - Invalid organization ID",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "invalid-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			expectedError: true,
			expectedLen:   0,
		},
		{
			name:          "Error - No organization selected",
			fCtx:          &ForgeContext{},
			expectedError: false,
			expectedLen:   0,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.fCtx.Context = s.ctx
			projects, err := getListOfProjects(*tc.fCtx)
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
		operation     string // "list", "get", "select", "selectFromSlug"
		fCtx          *ForgeContext
		input         string // for select operation
		slug          string // for selectFromSlug operation
		expectedError bool
		expectedOrgs  int    // for list operation
		testToken     string // Add test token to control project list response
	}{
		{
			name:      "Success - List organizations",
			operation: "list",
			fCtx: &ForgeContext{
				Config: &Config{
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			expectedError: false,
			expectedOrgs:  1,
		},
		{
			name:      "Success - Get selected organization",
			operation: "get",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			expectedError: false,
		},
		{
			name:      "Success - Select organization",
			operation: "select",
			fCtx: &ForgeContext{
				Config: &Config{
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			input:         "1",
			expectedError: false,
		},
		{
			name:      "Error - Get invalid organization",
			operation: "get",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "invalid-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			expectedError: true,
		},
		{
			name:      "Error - Select cancelled",
			operation: "select",
			fCtx: &ForgeContext{
				Config: &Config{
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			input:         "q",
			expectedError: true,
		},
		// New test cases for project selection scenarios
		{
			name:      "Success - Select organization with single project",
			operation: "select",
			fCtx: &ForgeContext{
				Config: &Config{
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			slug:          "testo",
			input:         "testp",
			expectedError: false,
		},
		{
			name:      "Success - Select organization with multiple projects",
			operation: "select",
			fCtx: &ForgeContext{
				Config: &Config{
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			slug:          "testo",
			input:         "1",
			expectedError: false,
			testToken:     "multiple-projects",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.fCtx.Context = s.ctx

			// Set test token if provided
			if tc.testToken != "" {
				s.testToken = tc.testToken
				// Reset test token after test
				defer func() { s.testToken = "" }()
			}

			switch tc.operation {
			case "list":
				orgs, err := getListOfOrganizations(*tc.fCtx)
				if tc.expectedError {
					s.Require().Error(err)
				} else {
					s.Require().NoError(err)
					s.Len(orgs, tc.expectedOrgs)
				}

			case "get":
				org, err := getSelectedOrganization(*tc.fCtx)
				if tc.expectedError {
					s.Require().Error(err)
				} else {
					s.Require().NoError(err)
					s.Equal("test-org-id", org.ID)
					s.Equal("Test Org", org.Name)
					s.Equal("testo", org.Slug)
				}

			case "select":
				getInput = func(prompt string, defaultVal string) string {
					printer.Infof("%s [%s]: ", prompt, defaultVal)
					printer.Infoln(tc.input)
					return tc.input
				}
				defer func() { getInput = originalGetInput }()

				org, err := selectOrganization(*tc.fCtx, &SwitchOrganizationCmd{Slug: tc.slug})
				if tc.expectedError {
					s.Require().Error(err)
					s.Empty(org)
				} else {
					s.Require().NoError(err)

					// Verify project selection behavior
					if tc.testToken != "" {
						config, err := GetForgeConfig()
						s.Require().NoError(err)

						switch tc.testToken {
						case "empty-list":
							// No projects case
							s.Empty(config.ProjectID)
							s.Empty(config.CurrProjectName)
						case "multiple-projects":
							// Multiple projects case - should have selected the project matching the input
							switch tc.input {
							case "testp1":
								s.Equal("test-project-id-1", config.ProjectID)
								s.Equal("Test Project 1", config.CurrProjectName)
							case "testp2":
								s.Equal("test-project-id-2", config.ProjectID)
								s.Equal("Test Project 2", config.CurrProjectName)
							}
						default:
							// Single project case
							s.Equal("test-project-id", config.ProjectID)
							s.Equal("Test Project", config.CurrProjectName)
						}
					}
				}
			}
		})
	}
}

func (s *ForgeTestSuite) TestCreateOrganization() {
	testCases := []struct {
		name            string
		input           []string
		expectedPrompt  []string
		expectInputFail int
		expectedError   bool
		expectedOrg     *organization
	}{
		{
			name: "Success - Valid organization default slug",
			input: []string{
				"My Great Org",    // name
				"",                // slug
				"http://test.com", // avatar URL
				"Y",               // confirm
			},
			expectInputFail: 0,
			expectedError:   false,
			expectedOrg: &organization{
				ID:   "test-org-id",
				Name: "Test Organization",
				Slug: "testo",
			},
			expectedPrompt: []string{
				"Enter organization name",
				"Enter organization slug",
				"Enter organization avatar URL (Empty Valid)",
				"Create organization with these details? (Y/n)",
			},
		},
		{
			name: "Success - Valid organization custom slug",
			input: []string{
				"testo",           // name
				"testo",           // slug
				"http://test.com", // avatar URL
				"Y",               // confirm
			},
			expectInputFail: 0,
			expectedError:   false,
			expectedOrg: &organization{
				ID:   "test-org-id",
				Name: "Test Organization",
				Slug: "testo",
			},
			expectedPrompt: []string{
				"Enter organization name",
				"Enter organization slug",
				"Enter organization avatar URL (Empty Valid)",
				"Create organization with these details? (Y/n)",
			},
		},
		{
			name: "Bad Input - Non-alphanumeric dots dash underscore slug",
			input: []string{
				"testo",           // name
				"te_st()",         // slug fail
				"testo",           // retry with valid slug
				"http://test.com", // avatar URL
				"Y",               // confirm
			},
			expectedPrompt: []string{
				"Enter organization name",
				"Enter organization slug",
				"Enter organization slug",
				"Enter organization avatar URL (Empty Valid)",
				"Create organization with these details? (Y/n)",
			},
			expectInputFail: 0,
			expectedError:   false,
			expectedOrg: &organization{
				ID:   "test-org-id",
				Name: "Test Organization",
				Slug: "testo",
			},
		},
		{
			name: "Error - Empty name",
			input: []string{
				"",                // name fail
				"testo",           // retry with valid name
				"testo",           // slug
				"http://test.com", // avatar URL
				"Y",               // confirm
			},
			expectedPrompt: []string{
				"Enter organization name",
				"Enter organization name",
				"Enter organization slug",
				"Enter organization avatar URL (Empty Valid)",
				"Create organization with these details? (Y/n)",
			},
			expectInputFail: 0,
			expectedError:   false,
			expectedOrg: &organization{
				ID:   "test-org-id",
				Name: "Test Organization",
				Slug: "testo",
			},
		},
		{
			name: "Success - Redo creation",
			input: []string{
				"testo",            // First attempt - name
				"testo",            // First attempt - slug
				"http://test.com",  // First attempt - avatar URL
				"n",                // First attempt - redo
				"testo2",           // Second attempt - name
				"testo2",           // Second attempt - slug
				"http://test2.com", // Second attempt - avatar URL
				"Y",                // Second attempt - confirm
			},
			expectedPrompt: []string{
				"Enter organization name",
				"Enter organization slug",
				"Enter organization avatar URL (Empty Valid)",
				"Create organization with these details? (Y/n)",
				"Enter organization name",
				"Enter organization slug",
				"Enter organization avatar URL (Empty Valid)",
				"Create organization with these details? (Y/n)",
			},
			expectInputFail: 0,
			expectedError:   false,
			expectedOrg: &organization{
				ID:   "test-org-id",
				Name: "Test Organization",
				Slug: "testo",
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			inputIndex := 0
			getInput = func(prompt string, defaultVal string) string {
				fmt.Printf("%s [%s]: ", prompt, defaultVal) //nolint:forbidigo // test

				// Validate against expected prompts if defined
				if len(tc.expectedPrompt) > 0 {
					if inputIndex >= len(tc.expectedPrompt) {
						panic(eris.Errorf("More prompts than expected. Got: %s", prompt))
					}
					if prompt != tc.expectedPrompt[inputIndex] {
						panic(eris.Errorf("Unexpected prompt at index %d. Expected: %s, Got: %s",
							inputIndex, tc.expectedPrompt[inputIndex], prompt))
					}
				}

				input := tc.input[inputIndex]
				if input == "" {
					input = defaultVal
				}
				printer.Infoln(input)
				inputIndex++
				return input
			}
			defer func() { getInput = originalGetInput }()

			fCtx := &ForgeContext{
				Context: s.ctx,
				Config:  &Config{},
			}

			org, err := createOrganization(*fCtx, &CreateOrganizationCmd{})
			if tc.expectedError {
				s.Require().Error(err)
				s.Nil(org)
			} else {
				s.Require().NoError(err)
				s.Require().NotNil(org)
				s.Equal(tc.expectedOrg.Name, org.Name)
				s.Equal(tc.expectedOrg.Slug, org.Slug)
			}
		})
	}
}

func (s *ForgeTestSuite) TestShowOrganizationList() {
	testCases := []struct {
		name          string
		fCtx          *ForgeContext
		expectedError bool
	}{
		{
			name: "Success - Show organization list with selected org",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			expectedError: false,
		},
		{
			name: "Success - Show organization list without selected org",
			fCtx: &ForgeContext{
				Config: &Config{
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			expectedError: false,
		},
		{
			name: "Error - Invalid organization ID",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "invalid-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.fCtx.Context = s.ctx
			err := showOrganizationList(*tc.fCtx)
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
		fCtx          *ForgeContext
		expectedError bool
	}{
		{
			name: "Success - Show project list with selected project",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					ProjectID:      "test-project-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			expectedError: false,
		},
		{
			name: "Success - Show project list without selected project",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			expectedError: false,
		},
		{
			name: "Success - Empty project list",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "empty-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			expectedError: false,
		},
		{
			name: "Error - Invalid organization ID",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "invalid-org-id",
					ProjectID:      "test-project-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			expectedError: true,
		},
		{
			name: "Error - Invalid project ID",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					ProjectID:      "invalid-project-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.fCtx.Context = s.ctx
			err := showProjectList(*tc.fCtx)
			if tc.expectedError {
				s.Error(err)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *ForgeTestSuite) TestCreateProject() {
	testCases := []struct {
		name                string
		fCtx                *ForgeContext
		inputs              []string     // For name, slug, repoURL, repoToken
		regionSelectActions []tea.KeyMsg // Simulate region selection
		expectInputFail     int
		expectedError       bool
		expectedProject     *project
		setupWorldToml      bool // New field to indicate if we should create world.toml
	}{
		{
			name: "Success - Public repo default slug + avatar URL validation",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					Credential: Credential{
						Token: "test-token",
					},
					KnownProjects: knownProjects,
				},
			},
			inputs: []string{
				"Test Project", // name
				"testp",        // take default
				"https://github.com/argus-labs/starter-game-template", // Repository URL
				"",                 // repoToken (empty for public repo)
				"",                 // repoPath (empty for default root path of repo)
				"10",               // tick rate
				"Y",                // enable discord notifications  NOTE: these won't show up in the console
				"test-token",       // discord token                       because results are mocked
				"1234567890",       // discord channel ID
				"Y",                // enable slack notifications
				"test-token",       // slack token
				"1234567890",       // slack channel ID
				"test.com",         // no http
				"http://test",      // no tld
				"http://.co.uk",    // invalid domain
				"http://localhost", // localhost invalid
				"http://test.com",  // avatar URL
			},
			regionSelectActions: []tea.KeyMsg{
				{Type: tea.KeySpace}, // select region
				{Type: tea.KeyEnter}, // confirm
			},
			expectInputFail: 0,
			expectedError:   false,
			expectedProject: &project{
				Name: "Test Project",
				Slug: "testp",
			},
		},
		{
			name: "Success - public repo custom slug",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					Credential: Credential{
						Token: "test-token",
					},
					KnownProjects: knownProjects,
				},
			},
			inputs: []string{
				"Test Project", // name
				"testp",        // slug
				"https://github.com/argus-labs/starter-game-template", // repoURL
				"",                // repoToken (empty for public repo)
				"",                // repoPath (empty for default root path of repo)
				"10",              // tick rate
				"Y",               // enable discord notifications  NOTE: these won't show up in the console
				"test-token",      // discord token                       because results are mocked
				"1234567890",      // discord channel ID
				"Y",               // enable slack notifications
				"test-token",      // slack token
				"1234567890",      // slack channel ID
				"http://test.com", // avatar URL
			},
			regionSelectActions: []tea.KeyMsg{
				{Type: tea.KeySpace}, // select region
				{Type: tea.KeyEnter}, // confirm
			},
			expectInputFail: 0,
			expectedError:   false,
			expectedProject: &project{
				Name: "Test Project",
				Slug: "testp",
			},
		},
		{
			name: "Abort - user presses q in region selector",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					Credential: Credential{
						Token: "test-token",
					},
					KnownProjects: knownProjects,
				},
			},
			inputs: []string{
				"Test Project", // name
				"testp",        // take default slug
				"https://github.com/argus-labs/starter-game-template", // repoURL
				"",   // repoToken (empty for public repo)
				"",   // repoPath
				"10", // tick rate
			},
			regionSelectActions: []tea.KeyMsg{
				{Type: tea.KeyRunes, Runes: []rune{'q'}, Alt: false}, // simulate pressing 'q'
			},
			expectedError:   true,
			expectedProject: nil,
		},
		{
			name: "Error - private repo bad token",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			inputs: []string{
				"Test Private",
				"testp",
				"https://github.com/test/private-repo",
				"bad-secret-token",
			},
			expectInputFail: 4,
			expectedError:   false,
		},
		{
			name: "Error - No organization selected",
			fCtx: &ForgeContext{
				Config: &Config{
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			inputs:          []string{},
			expectedError:   true,
			expectedProject: nil,
		},
		{
			name: "Error - Invalid organization ID",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "invalid-org-id",
					Credential: Credential{
						Token: "test-token",
					},
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
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			inputs: []string{
				"Test Project",
				"te", // invalid slug 1st attempts
			},
			expectInputFail: 2,
			expectedError:   false,
			expectedProject: nil,
		},
		{
			name: "Error - Invalid repo URL",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			inputs: []string{
				"Test Project",
				"testp",
				"invalid-url", // invalid URL 1st attempts
				"",            // no token
			},
			expectInputFail: 4,
			expectedError:   false,
			expectedProject: nil,
		},
		{
			name: "Success - Project name from world.toml",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					Credential: Credential{
						Token: "test-token",
					},
					KnownProjects: knownProjects,
				},
			},
			inputs: []string{
				"",      // name (should be taken from world.toml)
				"testp", // take default slug
				"https://github.com/argus-labs/starter-game-template", // repoURL
				"",                // repoToken (empty for public repo)
				"",                // repoPath (empty for default root path of repo)
				"10",              // tick rate
				"Y",               // enable discord notifications
				"test-token",      // discord token
				"1234567890",      // discord channel ID
				"Y",               // enable slack notifications
				"test-token",      // slack token
				"1234567890",      // slack channel ID
				"http://test.com", // avatar URL
			},
			regionSelectActions: []tea.KeyMsg{
				{Type: tea.KeySpace}, // select region
				{Type: tea.KeyEnter}, // confirm
			},
			expectInputFail: 0,
			expectedError:   false,
			expectedProject: &project{
				Name: "test-project-from-toml",
				Slug: "testp",
			},
			setupWorldToml: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.fCtx.Context = s.ctx

			// Create a temp directory for this test case
			tmpDir, err := os.MkdirTemp("", "forge-create-test")
			s.Require().NoError(err)
			defer os.RemoveAll(tmpDir)

			// Save original working directory
			oldDir, err := os.Getwd()
			s.Require().NoError(err)
			defer os.Chdir(oldDir)

			// Change to temp directory
			err = os.Chdir(tmpDir)
			s.Require().NoError(err)

			// Create minimal World project structure
			worldDir := filepath.Join(tmpDir, ".world")
			err = os.MkdirAll(worldDir, 0755)
			s.Require().NoError(err)

			// Create cardinal directory
			cardinalDir := filepath.Join(tmpDir, "cardinal")
			err = os.MkdirAll(cardinalDir, 0755)
			s.Require().NoError(err)

			// Initialize git repository
			cmd := exec.Command("git", "init")
			err = cmd.Run()
			s.Require().NoError(err)

			// Add a remote for the repository
			cmd = exec.Command("git", "remote", "add", "origin", "https://github.com/Argus-Labs/starter-game-template")
			err = cmd.Run()
			s.Require().NoError(err)

			// Create world.toml if needed
			if tc.setupWorldToml {
				worldTomlPath := filepath.Join(tmpDir, "world.toml")
				worldTomlContent := `[forge]
PROJECT_NAME = "test-project-from-toml"
`
				err = os.WriteFile(worldTomlPath, []byte(worldTomlContent), 0644)
				s.Require().NoError(err)
			} else {
				// Create empty world.toml for all test cases
				worldTomlPath := filepath.Join(tmpDir, "world.toml")
				err = os.WriteFile(worldTomlPath, []byte(""), 0644)
				s.Require().NoError(err)
			}

			if len(tc.inputs) > 0 {
				inputIndex := 0
				getInput = func(prompt string, defaultVal string) string {
					printer.Infof("%s [%s]: ", prompt, defaultVal)

					if inputIndex >= len(tc.inputs) {
						panic(fmt.Errorf("Input %d Failed", inputIndex))
					}

					input := tc.inputs[inputIndex]
					inputIndex++

					if input == "" {
						input = defaultVal
					}

					printer.Infoln(input)
					return input
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
				printer.Errorln("failed to create region selector")
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
						printer.Errorln("region selector is nil")
					}
				}
			}()

			var prj *project
			if tc.expectInputFail > 0 { //nolint: nestif // it's a test
				s.PanicsWithError(fmt.Sprintf("Input %d Failed", tc.expectInputFail), func() {
					prj, err = createProject(*tc.fCtx, &CreateProjectCmd{})
				})
			} else {
				prj, err = createProject(*tc.fCtx, &CreateProjectCmd{})
				if tc.expectedError {
					s.Require().Error(err)
					s.Nil(prj)
				} else {
					s.Require().NoError(err)
					s.Require().NotNil(prj)
					if tc.expectedProject != nil && prj != nil {
						s.Equal(tc.expectedProject.Name, prj.Name)
						s.Equal(tc.expectedProject.Slug, prj.Slug)
					}
				}
			}
		})
	}
}

func (s *ForgeTestSuite) TestSelectProject() {
	testCases := []struct {
		name          string
		fCtx          *ForgeContext
		input         string
		expectedError bool
		expectedProj  *project
	}{
		{
			name: "Success - Valid project selection",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					Credential: Credential{
						Token: "test-token",
					},
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
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			input:         "q",
			expectedError: false,
			expectedProj:  nil,
		},
		{
			name: "Error - Empty project list",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "empty-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			input:         "1",
			expectedError: false,
			expectedProj:  nil,
		},
		/* { // disabled because this loops forever right now
			name: "Error - Invalid selection number",
			config: ForgeConfig{
				OrganizationID: "test-org-id",
				Credential: Credential{
					Token: "test-token",
				},
			},
			input:         "99",
			expectedError: true,
			expectedProj:  nil,
		}, */
		{
			name: "Error - Invalid organization ID",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "invalid-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			input:         "1",
			expectedError: true,
			expectedProj:  nil,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.fCtx.Context = s.ctx

			getInput = func(prompt string, defaultVal string) string {
				printer.Infof("%s [%s]: ", prompt, defaultVal)
				printer.Infoln(tc.input)
				return tc.input
			}
			defer func() { getInput = originalGetInput }()

			proj, err := selectProject(*tc.fCtx, &SwitchProjectCmd{})
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
			expectedInput: "member",
			expectError:   false, // can't actually test this because it's two separate inputs
		},
		{
			name:          "None disabled",
			input:         "none\n",
			allowNone:     false,
			expectedInput: "member",
			expectError:   false,
		},
		{
			name:          "Bad Role",
			input:         "garbage\n",
			allowNone:     true,
			expectedInput: "member",
			expectError:   false,
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
			//nolint:reassign // Might cause issues with parallel tests
			defer func() { os.Stdin = oldStdin }()

			// Set stdin to our test file
			os.Stdin = tmpfile //nolint:reassign // Might cause issues with parallel tests

			// Test getInput
			result := getRoleInput(tc.allowNone, "")
			s.Equal(tc.expectedInput, result)
		})
	}
}

func (s *ForgeTestSuite) TestGetInput() {
	testCases := []struct {
		name          string
		input         string
		defaultInput  string
		expectedInput string
		expectError   bool
	}{
		{
			name:          "Success - Normal input",
			input:         "test input\n",
			defaultInput:  "bad",
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
			name:          "Success - Input with default value",
			input:         "\n",
			defaultInput:  "default input value",
			expectedInput: "default input value",
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
			expectedInput: "test input",
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
			//nolint:reassign // Might cause issues with parallel tests
			defer func() { os.Stdin = oldStdin }()
			// Set stdin to our test file
			os.Stdin = tmpfile //nolint:reassign // Might cause issues with parallel tests

			// Test getInput
			result := getInput("test-prompt: ", tc.defaultInput)
			s.Equal(tc.expectedInput, result)
		})
	}
}

func (s *ForgeTestSuite) TestInviteUserToOrganization() {
	testCases := []struct {
		name            string
		fCtx            *ForgeContext
		inputs          []string // For user id, role
		expectInputFail int
		expectedError   bool
	}{
		{
			name: "Success - Default role",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			inputs: []string{
				"test-user-id", // user-id
				"",             // role
			},
			expectInputFail: 0,
			expectedError:   false,
		},
		{
			name: "Success - admin role",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			inputs: []string{
				"test-user-id", // user-id
				"admin",        // role
			},
			expectInputFail: 0,
			expectedError:   false,
		},
		{
			name: "Error - No organization selected",
			fCtx: &ForgeContext{
				Config: &Config{
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			inputs:          []string{},
			expectInputFail: 0,
			expectedError:   true,
		},
		{
			name: "Error - Invalid organization ID",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "invalid-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			inputs: []string{
				"test-user-id", // user-id
				"",             // role
			},
			expectInputFail: 0,
			expectedError:   true,
		},
		{
			name: "Error - Invalid Role: None",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			inputs: []string{
				"test-user-id", // user-id
				"none",         // invalid role
			},
			expectInputFail: 2,
			expectedError:   true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Setup test config
			tc.fCtx.Context = s.ctx

			if len(tc.inputs) > 0 {
				inputIndex := 0
				lastPrompt := ""
				getInput = func(prompt string, defaultVal string) string {
					printer.Infof("%s [%s]: ", prompt, defaultVal)
					if prompt == lastPrompt {
						panic(eris.Errorf("Input %d Failed", inputIndex))
					}
					lastPrompt = prompt
					input := tc.inputs[inputIndex]
					if input == "" {
						input = defaultVal
					}
					printer.Infoln(input)
					inputIndex++
					return input
				}
				defer func() { getInput = originalGetInput }()
			}
			defer func() { regionSelector = nil }()

			org := organization{ID: tc.fCtx.Config.OrganizationID}
			if tc.expectInputFail > 0 {
				s.PanicsWithError(fmt.Sprintf("Input %d Failed", tc.expectInputFail), func() {
					err := org.inviteUser(*tc.fCtx, &InviteUserToOrganizationCmd{})
					s.Require().NoError(err)
				})
			} else {
				err := org.inviteUser(*tc.fCtx, &InviteUserToOrganizationCmd{})
				if tc.expectedError {
					s.Error(err)
				} else {
					s.NoError(err)
				}
			}
		})
	}
}

func (s *ForgeTestSuite) TestUpdateRoleInOrganization() {
	testCases := []struct {
		name            string
		fCtx            *ForgeContext
		inputs          []string // For user id, role
		expectInputFail int
		expectedError   bool
	}{
		{
			name: "Success - Default role",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			inputs: []string{
				"test-user-id", // user-id
				"",             // role
			},
			expectInputFail: 0,
			expectedError:   false,
		},
		{
			name: "Success - admin role",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			inputs: []string{
				"test-user-id", // user-id
				"admin",        // role
			},
			expectInputFail: 0,
			expectedError:   false,
		},
		{
			name: "Success - none with confirm remove",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			inputs: []string{
				"test-user-id", // user-id
				"none",         // role
				"Yes",          // confirm removal
			},
			expectInputFail: 0,
			expectedError:   false,
		},
		{
			name: "Error - No organization selected",
			fCtx: &ForgeContext{
				Config: &Config{
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			inputs:          []string{},
			expectInputFail: 0,
			expectedError:   true,
		},
		{
			name: "Error - Invalid organization ID",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "invalid-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			inputs: []string{
				"test-user-id", // user-id
				"",             // role
			},
			expectInputFail: 0,
			expectedError:   true,
		},
		{
			name: "Error - Role none dont confirm remove",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					Credential: Credential{
						Token: "test-token",
					},
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
			expectInputFail: 10,
			expectedError:   true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Setup test config
			tc.fCtx.Context = s.ctx

			if len(tc.inputs) > 0 {
				inputIndex := 0
				lastPrompt := ""
				nextToLastPrompt := ""
				getInput = func(prompt string, defaultVal string) string {
					printer.Infof("%s [%s]: ", prompt, defaultVal)
					if (prompt == lastPrompt || prompt == nextToLastPrompt) && tc.expectInputFail <= inputIndex {
						panic(eris.Errorf("Input %d Failed", inputIndex))
					}
					nextToLastPrompt = lastPrompt
					lastPrompt = prompt
					input := tc.inputs[inputIndex]
					if input == "" {
						input = defaultVal
					}
					printer.Infoln(input)
					inputIndex++
					return input
				}
				defer func() { getInput = originalGetInput }()
			}
			defer func() { regionSelector = nil }()

			org := organization{ID: tc.fCtx.Config.OrganizationID}
			if tc.expectInputFail > 0 {
				s.PanicsWithError(fmt.Sprintf("Input %d Failed", tc.expectInputFail), func() {
					err := org.updateUserRole(*tc.fCtx, &ChangeUserRoleInOrganizationCmd{})
					s.Require().NoError(err)
				})
			} else {
				err := org.updateUserRole(*tc.fCtx, &ChangeUserRoleInOrganizationCmd{})
				if tc.expectedError {
					s.Error(err)
				} else {
					s.NoError(err)
				}
			}
		})
	}
}

func (s *ForgeTestSuite) TestSlugCreateFromName() {
	testCases := []struct {
		name         string
		input        string
		minLen       int
		maxLen       int
		expectedSlug string
	}{
		{
			name:         "Basic conversion",
			input:        "My Project Name",
			minLen:       3,
			maxLen:       25,
			expectedSlug: "my_project_name",
		},
		{
			name:         "With special characters",
			input:        "Project!@#$%^&*()",
			minLen:       3,
			maxLen:       25,
			expectedSlug: "project",
		},
		{
			name:         "With dashes",
			input:        "my-project-name",
			minLen:       3,
			maxLen:       25,
			expectedSlug: "my_project_name",
		},
		{
			name:         "With multiple spaces",
			input:        "   Multiple   Spaces   ",
			minLen:       3,
			maxLen:       25,
			expectedSlug: "multiple_spaces",
		},
		{
			name:         "With numbers",
			input:        "Project 123",
			minLen:       3,
			maxLen:       25,
			expectedSlug: "project_123",
		},
		{
			name:         "Empty string",
			input:        "",
			minLen:       3,
			maxLen:       25,
			expectedSlug: "{hex8}",
		},
		{
			name:         "Only special characters",
			input:        "!@#$%^&*()",
			minLen:       3,
			maxLen:       25,
			expectedSlug: "{hex8}",
		},
		{
			name:         "Too short",
			input:        "AA",
			minLen:       3,
			maxLen:       25,
			expectedSlug: "aa_{hex8}",
		},
		{
			name:         "Reducing underscores",
			input:        "Big___Stretch",
			minLen:       3,
			maxLen:       25,
			expectedSlug: "big_stretch",
		},
		{
			name:         "CamelCase conversion",
			input:        "MyProjectName",
			minLen:       3,
			maxLen:       25,
			expectedSlug: "my_project_name",
		},
		{
			name:         "Mixed CamelCase with spaces",
			input:        "My CamelCase ProjectName",
			minLen:       3,
			maxLen:       25,
			expectedSlug: "my_camel_case_project_nam",
		},
		{
			name:         "Complex mixed case with special chars",
			input:        "MyProject!Name@With#Stuff",
			minLen:       3,
			maxLen:       25,
			expectedSlug: "my_project_name_with_stuf",
		},
		{
			name:         "Leading number",
			input:        "123Project",
			minLen:       3,
			maxLen:       25,
			expectedSlug: "123project",
		},
		{
			name:         "CamelCase with numbers",
			input:        "My2Project3Name",
			minLen:       3,
			maxLen:       25,
			expectedSlug: "my2project3name",
		},
		{
			name:         "Complex CamelCase with nums",
			input:        "Project3With4SmallNums",
			minLen:       3,
			maxLen:       25,
			expectedSlug: "project3with4small_nums",
		},
		{
			name:         "Numbers with special characters",
			input:        "123!Project@456",
			minLen:       3,
			maxLen:       25,
			expectedSlug: "123_project_456",
		},
		{
			name:         "Very long (truncate)",
			input:        "This_is_a_very_long_name_which_should_be_truncated",
			minLen:       3,
			maxLen:       25,
			expectedSlug: "this_is_a_very_long_name",
		},
		{
			name:         "Very long (shorten)",
			input:        "This is a very long name which should be truncated",
			minLen:       3,
			maxLen:       25,
			expectedSlug: "thisisaverylongnamewhichs",
		},
		{
			name:         "Very long CamelCase (shorten)",
			input:        "ThisIsAVeryLongNameWhichShouldBeTruncated",
			minLen:       3,
			maxLen:       25,
			expectedSlug: "thisisaverylongnamewhichs",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := CreateSlugFromName(tc.input, tc.minLen, tc.maxLen)
			if strings.Contains(tc.expectedSlug, "{hex8}") {
				expectedSlug := strings.Replace(tc.expectedSlug, "{hex8}", "", 1)
				if len(result) >= len(expectedSlug) {
					result = result[:len(expectedSlug)]
				}
				s.Equal(expectedSlug, result)
			} else {
				s.Equal(tc.expectedSlug, result)
			}
		})
	}
}

func (s *ForgeTestSuite) TestFindGitPathAndURL() {
	// Get the actual repo URL for the first test case
	actualPath, actualURL, err := FindGitPathAndURL()
	s.Require().NoError(err)

	testCases := []struct {
		name           string
		setupGit       func(string) error
		expectedPath   string
		expectedURL    string
		expectedError  bool
		changeToSubdir bool
	}{
		{
			name: "Success - Normal path with origin remote",
			setupGit: func(dir string) error {
				// No setup needed - uses existing git repo
				return nil
			},
			expectedPath:   actualPath,
			expectedURL:    actualURL,
			expectedError:  false,
			changeToSubdir: false,
		},
		{
			name: "Success - Fallback to non-origin remote",
			setupGit: func(dir string) error {
				// Initialize git repo
				gitInit := exec.Command("git", "init")
				gitInit.Dir = dir
				if err := gitInit.Run(); err != nil {
					return err
				}

				// Add upstream remote
				gitRemote := exec.Command("git", "remote", "add", "upstream", "https://github.com/test/fallback.git")
				gitRemote.Dir = dir
				if err := gitRemote.Run(); err != nil {
					return err
				}

				// Create subdirectory
				subDir := filepath.Join(dir, "subdir")
				if err := os.Mkdir(subDir, 0755); err != nil {
					return err
				}

				return nil
			},
			expectedPath:   "subdir",
			expectedURL:    "https://github.com/test/fallback",
			expectedError:  false,
			changeToSubdir: true,
		},
		{
			name: "Error - No git repository",
			setupGit: func(dir string) error {
				// No setup - empty directory
				return nil
			},
			expectedPath:   "",
			expectedURL:    "",
			expectedError:  true,
			changeToSubdir: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Create temporary directory
			tmpDir, err := os.MkdirTemp("", "git-test-*")
			s.Require().NoError(err)
			defer os.RemoveAll(tmpDir)

			// Setup git repository if needed
			if tc.setupGit != nil {
				err = tc.setupGit(tmpDir)
				s.Require().NoError(err)
			}

			// Change to the test directory
			originalDir, err := os.Getwd()
			s.Require().NoError(err)
			defer os.Chdir(originalDir)

			// Change to subdir if needed, and check existence
			if tc.changeToSubdir {
				subdirPath := filepath.Join(tmpDir, "subdir")
				info, err := os.Stat(subdirPath)
				s.Require().NoError(err, "subdir should exist")
				s.Require().True(info.IsDir(), "subdir should be a directory")
				err = os.Chdir(subdirPath)
				s.Require().NoError(err)
			} else if tc.name != "Success - Normal path with origin remote" {
				err = os.Chdir(tmpDir)
				s.Require().NoError(err)
			}

			// Run the test
			path, url, err := FindGitPathAndURL()
			if tc.expectedError {
				s.Error(err)
			} else {
				s.Require().NoError(err)
				s.Equal(tc.expectedPath, path)
				s.Equal(tc.expectedURL, url)
			}
		})
	}
}

func (s *ForgeTestSuite) TestSetupForgeCommandState() {
	testCases := []struct {
		name          string
		fCtx          *ForgeContext
		loginReq      LoginStepRequirement
		orgReq        StepRequirement
		projectReq    StepRequirement
		expectedError bool
		checkState    func(*CommandState)
	}{
		{
			name: "Success - Ignore all requirements",
			fCtx: &ForgeContext{
				Config: &Config{
					Credential: Credential{
						Token: "",
					},
				},
			},
			loginReq:      IgnoreLogin,
			orgReq:        Ignore,
			projectReq:    Ignore,
			expectedError: false,
			checkState: func(state *CommandState) {
				s.Nil(state.User)
				s.Nil(state.Organization)
				s.Nil(state.Project)
			},
		},
		{
			name: "Success - Need login and have token",
			fCtx: &ForgeContext{
				Config: &Config{
					Credential: Credential{
						Token:          "test-token",
						TokenExpiresAt: time.Now().Add(1 * time.Hour),
					},
				},
			},
			loginReq:      NeedLogin,
			orgReq:        Ignore,
			projectReq:    Ignore,
			expectedError: false,
			checkState: func(state *CommandState) {
				s.NotNil(state.User)
				s.Nil(state.Organization)
				s.Nil(state.Project)
			},
		},
		{
			name: "Error - Need login but expired token",
			fCtx: &ForgeContext{
				Config: &Config{
					Credential: Credential{
						Token:          "test-token",
						TokenExpiresAt: time.Now().Add(-1 * time.Hour),
					},
				},
			},
			loginReq:      NeedLogin,
			orgReq:        Ignore,
			projectReq:    Ignore,
			expectedError: true,
			checkState: func(state *CommandState) {
				s.Nil(state.User)
				s.Nil(state.Organization)
				s.Nil(state.Project)
			},
		},
		{
			name: "Success - Need org ID and have it",
			fCtx: &ForgeContext{
				Config: &Config{
					Credential: Credential{
						Token:          "test-token",
						TokenExpiresAt: time.Now().Add(1 * time.Hour),
					},
					OrganizationID: "test-org-id",
				},
			},
			loginReq:      NeedLogin,
			orgReq:        NeedIDOnly,
			projectReq:    Ignore,
			expectedError: false,
			checkState: func(state *CommandState) {
				s.NotNil(state.User)
				s.NotNil(state.Organization)
				s.Equal("test-org-id", state.Organization.ID)
				s.Nil(state.Project)
			},
		},
		{
			name: "Success - Need project ID and have it",
			fCtx: &ForgeContext{
				Config: &Config{
					Credential: Credential{
						Token:          "test-token",
						TokenExpiresAt: time.Now().Add(1 * time.Hour),
					},
					OrganizationID:  "test-org-id",
					ProjectID:       "test-project-id",
					CurrProjectName: "Test Project",
				},
			},
			loginReq:      NeedLogin,
			orgReq:        NeedIDOnly,
			projectReq:    NeedIDOnly,
			expectedError: false,
			checkState: func(state *CommandState) {
				s.NotNil(state.User)
				s.NotNil(state.Organization)
				s.Equal("test-org-id", state.Organization.ID)
				s.NotNil(state.Project)
				s.Equal("test-project-id", state.Project.ID)
				// s.Equal("Test Project", state.Project.Name)
			},
		},
		{
			name: "Error - Need login but no token",
			fCtx: &ForgeContext{
				Config: &Config{
					Credential: Credential{
						Token: "",
					},
				},
			},
			loginReq:      NeedLogin,
			orgReq:        Ignore,
			projectReq:    Ignore,
			expectedError: true,
			checkState: func(state *CommandState) {
				s.Nil(state.User)
				s.Nil(state.Organization)
				s.Nil(state.Project)
			},
		},
		{
			name: "Error - Must not have org but have org ID",
			fCtx: &ForgeContext{
				Config: &Config{
					Credential: Credential{
						Token:          "test-token",
						TokenExpiresAt: time.Now().Add(1 * time.Hour),
					},
					OrganizationID: "test-org-id",
				},
			},
			loginReq:      NeedLogin,
			orgReq:        MustNotExist,
			projectReq:    Ignore,
			expectedError: true,
			checkState: func(state *CommandState) {
				s.NotNil(state.User)
				s.Nil(state.Organization)
				s.Nil(state.Project)
			},
		},
		{
			name: "Error - Must not have project but have project ID",
			fCtx: &ForgeContext{
				Config: &Config{
					Credential: Credential{
						Token:          "test-token",
						TokenExpiresAt: time.Now().Add(1 * time.Hour),
					},
					OrganizationID: "test-org-id",
					ProjectID:      "test-project-id",
				},
			},
			loginReq:      NeedLogin,
			orgReq:        NeedIDOnly,
			projectReq:    MustNotExist,
			expectedError: true,
			checkState: func(state *CommandState) {
				s.NotNil(state.User)
				s.Nil(state.Organization)
				s.Nil(state.Project)
			},
		},
		{
			name: "Success - Need repo lookup and have URL",
			fCtx: &ForgeContext{
				Config: &Config{
					Credential: Credential{
						Token:          "test-token",
						TokenExpiresAt: time.Now().Add(1 * time.Hour),
					},
					ProjectID:      "test-project-id",
					OrganizationID: "test-org-id",
					CurrRepoURL:    "https://github.com/test/repo",
					CurrRepoPath:   "cmd/world/forge",
				},
			},
			loginReq:      NeedLogin,
			orgReq:        NeedIDOnly,
			projectReq:    NeedIDOnly,
			expectedError: false,
			checkState: func(state *CommandState) {
				s.NotNil(state.User)
				s.NotNil(state.Organization)
				s.NotNil(state.Project)
			},
		},
	}

	// Mock browser opening
	openBrowser = func(_ string) error { return nil }
	defer func() { openBrowser = originalOpenBrowser }()

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Setup test config
			tc.fCtx.Context = s.ctx

			// Run the test
			err := tc.fCtx.SetupForgeCommandState(tc.loginReq, tc.orgReq, tc.projectReq)

			// Check error
			if tc.expectedError {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}

			// Check state
			tc.checkState(&tc.fCtx.State)
		})
	}
}

func (s *ForgeTestSuite) TestAddKnownProject() {
	config := &Config{
		KnownProjects: []KnownProject{},
	}

	proj := &project{
		ID:       "test-project-id",
		OrgID:    "test-org-id",
		RepoURL:  "https://github.com/test/repo",
		RepoPath: "cmd/world/forge",
		Name:     "Test Project",
	}

	proj.AddKnownProject(config)

	s.Len(config.KnownProjects, 1)
	s.Equal(KnownProject{
		ProjectID:      "test-project-id",
		OrganizationID: "test-org-id",
		RepoURL:        "https://github.com/test/repo",
		RepoPath:       "cmd/world/forge",
		ProjectName:    "Test Project",
	}, config.KnownProjects[0])
}

func (s *ForgeTestSuite) TestRemoveKnownProject() {
	testCases := []struct {
		name             string
		fCtx             *ForgeContext
		projectToRemove  *project
		expectedProjects []KnownProject
	}{
		{
			name: "successfully remove project",
			fCtx: &ForgeContext{
				Config: &Config{
					KnownProjects: []KnownProject{
						{
							ProjectID:      "test-project-id",
							OrganizationID: "test-org-id",
							RepoURL:        "https://github.com/test/repo",
							RepoPath:       "cmd/world/forge",
							ProjectName:    "Test Project",
						},
						{
							ProjectID:      "other-project-id",
							OrganizationID: "other-org-id",
							RepoURL:        "https://github.com/other/repo",
							RepoPath:       "other/path",
							ProjectName:    "Other Project",
						},
					},
				},
			},
			projectToRemove: &project{
				ID:       "test-project-id",
				OrgID:    "test-org-id",
				RepoURL:  "https://github.com/test/repo",
				RepoPath: "cmd/world/forge",
				Name:     "Test Project",
			},
			expectedProjects: []KnownProject{
				{
					ProjectID:      "other-project-id",
					OrganizationID: "other-org-id",
					RepoURL:        "https://github.com/other/repo",
					RepoPath:       "other/path",
					ProjectName:    "Other Project",
				},
			},
		},
		{
			name: "remove project with empty config",
			fCtx: &ForgeContext{
				Config: &Config{
					KnownProjects: make([]KnownProject, 0),
				},
			},
			projectToRemove: &project{
				ID:       "test-project-id",
				OrgID:    "test-org-id",
				RepoURL:  "https://github.com/test/repo",
				RepoPath: "cmd/world/forge",
				Name:     "Test Project",
			},
			expectedProjects: make([]KnownProject, 0),
		},
		{
			name: "remove non-existent project",
			fCtx: &ForgeContext{
				Config: &Config{
					KnownProjects: []KnownProject{
						{
							ProjectID:      "other-project-id",
							OrganizationID: "other-org-id",
							RepoURL:        "https://github.com/other/repo",
							RepoPath:       "other/path",
							ProjectName:    "Other Project",
						},
					},
				},
			},
			projectToRemove: &project{
				ID:       "test-project-id",
				OrgID:    "test-org-id",
				RepoURL:  "https://github.com/test/repo",
				RepoPath: "cmd/world/forge",
				Name:     "Test Project",
			},
			expectedProjects: []KnownProject{
				{
					ProjectID:      "other-project-id",
					OrganizationID: "other-org-id",
					RepoURL:        "https://github.com/other/repo",
					RepoPath:       "other/path",
					ProjectName:    "Other Project",
				},
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Setup
			tc.fCtx.Context = s.ctx

			// Execute
			tc.projectToRemove.RemoveKnownProject(tc.fCtx.Config)

			// Verify
			s.Equal(tc.expectedProjects, tc.fCtx.Config.KnownProjects)
		})
	}
}

func (s *ForgeTestSuite) TestHandleNeedOrgData() {
	testCases := []struct {
		name            string
		testToken       string
		inputs          []string
		expectInputFail int
		expectedError   bool
		expectedOrg     *organization
	}{
		{
			name:      "Success - Create org when none exist",
			testToken: "empty-list",
			inputs: []string{
				"Y",               // create org
				"My Great Org",    // name
				"",                // slug
				"http://test.com", // avatar URL
				"Y",               // confirm
			},
			expectInputFail: 0,
			expectedError:   false,
			expectedOrg: &organization{
				ID:   "test-org-id",
				Name: "Test Organization",
				Slug: "testo",
			},
		},
		{
			name:      "Success - Cancel org creation",
			testToken: "empty-list",
			inputs: []string{
				"n", // don't create org
			},
			expectInputFail: 0,
			expectedError:   true,
			expectedOrg:     nil,
		},
		{
			name:      "Success - Create org with custom slug",
			testToken: "empty-list",
			inputs: []string{
				"Y",               // create org
				"testo",           // name
				"testo",           // slug
				"http://test.com", // avatar URL
				"Y",               // confirm
			},
			expectInputFail: 0,
			expectedError:   false,
			expectedOrg: &organization{
				ID:   "test-org-id",
				Name: "Test Organization",
				Slug: "testo",
			},
		},
		{
			name:      "Success - Single org exists",
			testToken: "test-token", // default token returns single org
			inputs: []string{
				"Y", // confirm using the single org
			},
			expectInputFail: 0,
			expectedError:   false,
			expectedOrg: &organization{
				ID:   "test-org-id",
				Name: "Test Org",
				Slug: "testo",
			},
		},
		{
			name:      "Success - Multiple orgs, select first",
			testToken: "multiple-orgs",
			inputs: []string{
				"1", // select first org
			},
			expectInputFail: 0,
			expectedError:   false,
			expectedOrg: &organization{
				ID:   "test-org-id-1",
				Name: "Test Org 1",
				Slug: "testo1",
			},
		},
		{
			name:      "Success - Multiple orgs, select second",
			testToken: "multiple-orgs",
			inputs: []string{
				"2", // select second org
			},
			expectInputFail: 0,
			expectedError:   false,
			expectedOrg: &organization{
				ID:   "test-org-id-2",
				Name: "Test Org 2",
				Slug: "testo2",
			},
		},
		{
			name:      "Error - Multiple orgs, cancel selection",
			testToken: "multiple-orgs",
			inputs: []string{
				"q", // quit selection
			},
			expectInputFail: 0,
			expectedError:   true,
			expectedOrg:     nil,
		},
		{
			name:      "Success - Single org exists",
			testToken: "test-token", // default token returns single org
			inputs: []string{
				"Y", // confirm using the single org
			},
			expectInputFail: 0,
			expectedError:   false,
			expectedOrg: &organization{
				ID:   "test-org-id",
				Name: "Test Org",
				Slug: "testo",
			},
		},
		{
			name:      "Error - Single org exists, cancel selection",
			testToken: "test-token", // default token returns single org
			inputs: []string{
				"n", // cancel using the single org
			},
			expectInputFail: 0,
			expectedError:   true,
			expectedOrg:     nil,
		},
		{
			name:      "Success - Single org exists, create new instead",
			testToken: "test-token", // default token returns single org
			inputs: []string{
				"c",               // create new org instead
				"My New Org",      // name
				"",                // slug
				"http://test.com", // avatar URL
				"Y",               // confirm
			},
			expectInputFail: 0,
			expectedError:   false,
			expectedOrg: &organization{
				ID:   "test-org-id",
				Name: "Test Organization",
				Slug: "testo",
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Set up test to return empty org list
			s.testToken = tc.testToken
			defer func() { s.testToken = "" }()

			inputIndex := 0
			getInput = func(prompt string, defaultVal string) string {
				printer.Infof("%s [%s]: ", prompt, defaultVal)

				input := tc.inputs[inputIndex]
				if input == "" {
					input = defaultVal
				}
				printer.Infoln(input)
				inputIndex++
				return input
			}
			defer func() { getInput = originalGetInput }()

			// Create flow
			flowState := &initFlow{}

			fCtx := &ForgeContext{
				Context: s.ctx,
				State:   CommandState{},
				Config:  &Config{},
			}
			// Run test
			err := flowState.handleNeedOrgData(fCtx)
			if tc.expectedError {
				s.Require().Error(err)
				s.Nil(fCtx.State.Organization)
			} else {
				s.Require().NoError(err)
				s.Require().NotNil(fCtx.State.Organization)
				s.Equal(tc.expectedOrg.Name, fCtx.State.Organization.Name)
				s.Equal(tc.expectedOrg.Slug, fCtx.State.Organization.Slug)
				s.True(flowState.organizationStepDone)
			}
		})
	}
}

func (s *ForgeTestSuite) TestHandleNeedExistingOrgData() {
	testCases := []struct {
		name            string
		testToken       string
		inputs          []string
		expectInputFail int
		expectedError   bool
		expectedOrg     *organization
	}{
		{
			name:            "Success - Single org exists",
			testToken:       "test-token", // default token returns single org
			inputs:          []string{},   // no input needed for single org case
			expectInputFail: 0,
			expectedError:   false,
			expectedOrg: &organization{
				ID:   "test-org-id",
				Name: "Test Org",
				Slug: "testo",
			},
		},
		{
			name:      "Success - Multiple orgs, select first",
			testToken: "multiple-orgs",
			inputs: []string{
				"1", // select first org
			},
			expectInputFail: 0,
			expectedError:   false,
			expectedOrg: &organization{
				ID:   "test-org-id-1",
				Name: "Test Org 1",
				Slug: "testo1",
			},
		},
		{
			name:      "Success - Multiple orgs, select second",
			testToken: "multiple-orgs",
			inputs: []string{
				"2", // select second org
			},
			expectInputFail: 0,
			expectedError:   false,
			expectedOrg: &organization{
				ID:   "test-org-id-2",
				Name: "Test Org 2",
				Slug: "testo2",
			},
		},
		{
			name:      "Error - Multiple orgs, cancel selection",
			testToken: "multiple-orgs",
			inputs: []string{
				"q", // quit selection
			},
			expectInputFail: 0,
			expectedError:   true,
			expectedOrg:     nil,
		},
		{
			name:            "Error - No orgs exist",
			testToken:       "empty-list",
			inputs:          []string{},
			expectInputFail: 0,
			expectedError:   true,
			expectedOrg:     nil,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.testToken = tc.testToken
			defer func() { s.testToken = "" }()

			inputIndex := 0
			getInput = func(prompt string, defaultVal string) string {
				printer.Infof("%s [%s]: ", prompt, defaultVal)

				if inputIndex >= len(tc.inputs) {
					return defaultVal
				}

				input := tc.inputs[inputIndex]
				if input == "" {
					input = defaultVal
				}
				printer.Infoln(input)
				inputIndex++
				return input
			}
			defer func() { getInput = originalGetInput }()

			// Create flow
			flowState := &initFlow{}

			fCtx := &ForgeContext{
				Context: s.ctx,
				State:   CommandState{},
				Config:  &Config{},
			}

			// Run test
			err := flowState.handleNeedExistingOrgData(fCtx)
			if tc.expectedError {
				s.Require().Error(err)
				s.Nil(fCtx.State.Organization)
			} else {
				s.Require().NoError(err)
				s.Require().NotNil(fCtx.State.Organization)
				s.Equal(tc.expectedOrg.Name, fCtx.State.Organization.Name)
				s.Equal(tc.expectedOrg.Slug, fCtx.State.Organization.Slug)
				s.True(flowState.organizationStepDone)
			}
		})
	}
}

func (s *ForgeTestSuite) TestHandleNeedProjectData() {
	testCases := []struct {
		name                string
		fCtx                *ForgeContext
		testToken           string
		inputs              []string
		regionSelectActions []tea.KeyMsg // Add region selection actions
		expectedError       bool
		createProject       bool
	}{
		{
			name:          "Success - Always returns no projects case",
			createProject: true,
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			testToken: "empty-list",
			inputs: []string{
				"Y",                            // Create project
				"Test Project",                 // Project name
				"testp",                        // Project slug
				"https://github.com/test/repo", // Repo URL
				"",                             // No token needed for public repo
				"",                             // Default repo path
				"10",                           // Tick rate
				"n",                            // No Discord
				"n",                            // No Slack
				"",                             // No avatar URL
			},
			regionSelectActions: []tea.KeyMsg{
				{Type: tea.KeySpace}, // select region
				{Type: tea.KeyEnter}, // confirm
			},
			expectedError: false,
		},
		{
			name: "Error - Cancel project creation",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			testToken: "empty-list",
			inputs: []string{
				"n", // Don't create project
			},
			expectedError: true,
		},
		{
			name: "Success - Single project exists",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			testToken: "test-token", // default token returns single project
			inputs: []string{
				"Y", // Confirm using the single project
			},
			expectedError: false,
		},
		{
			name: "Error - Single project exists, cancel selection",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			testToken: "test-token", // default token returns single project
			inputs: []string{
				"n", // Cancel using the single project
			},
			expectedError: true,
		},
		{
			name:          "Success - Single project exists, create new project",
			createProject: true,
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			testToken: "test-token", // default token returns single project
			inputs: []string{
				"c",           // Choose to create new project
				"New Project", // Project name
				"testp",       // Project slug
				"https://github.com/argus-labs/starter-game-template", // Repo URL
				"",   // No token needed for public repo
				"",   // Default repo path
				"10", // Tick rate
				"n",  // No Discord
				"n",  // No Slack
				"",   // No avatar URL
			},
			regionSelectActions: []tea.KeyMsg{
				{Type: tea.KeySpace}, // select region
				{Type: tea.KeyEnter}, // confirm
			},
			expectedError: false,
		},
		{
			name: "Success - Multiple projects, select first project",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			testToken: "multiple-projects",
			inputs: []string{
				"1", // Select first project
			},
			expectedError: false,
		},
		{
			name: "Success - Multiple projects, select second project",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			testToken: "multiple-projects",
			inputs: []string{
				"2", // Select second project
			},
			expectedError: false,
		},
		{
			name: "Error - Multiple projects, cancel selection",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			testToken: "multiple-projects",
			inputs: []string{
				"q", // Cancel selection
			},
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Setup
			tc.fCtx.Context = s.ctx

			if tc.createProject {
				// Create a temp directory for this test case
				tmpDir, err := os.MkdirTemp("", "forge-create-test")
				s.Require().NoError(err)
				defer os.RemoveAll(tmpDir)

				// Save original working directory
				oldDir, err := os.Getwd()
				s.Require().NoError(err)
				defer os.Chdir(oldDir)

				// Change to temp directory
				err = os.Chdir(tmpDir)
				s.Require().NoError(err)

				// Create minimal World project structure
				worldDir := filepath.Join(tmpDir, ".world")
				err = os.MkdirAll(worldDir, 0755)
				s.Require().NoError(err)

				// Create cardinal directory
				cardinalDir := filepath.Join(tmpDir, "cardinal")
				err = os.MkdirAll(cardinalDir, 0755)
				s.Require().NoError(err)

				// Initialize git repository
				cmd := exec.Command("git", "init")
				err = cmd.Run()
				s.Require().NoError(err)

				// Add a remote for the repository - use the same URL as in the test inputs
				cmd = exec.Command(
					"git",
					"remote",
					"add",
					"origin",
					"https://github.com/argus-labs/starter-game-template",
				)
				err = cmd.Run()
				s.Require().NoError(err)

				// Create empty world.toml
				worldTomlPath := filepath.Join(tmpDir, "world.toml")
				err = os.WriteFile(worldTomlPath, []byte(""), 0644)
				s.Require().NoError(err)
			}

			// Set test token
			s.testToken = tc.testToken
			defer func() { s.testToken = "" }()

			// Setup input mocking
			if len(tc.inputs) > 0 {
				inputIndex := 0
				getInput = func(prompt string, defaultVal string) string {
					if inputIndex >= len(tc.inputs) {
						return defaultVal
					}
					input := tc.inputs[inputIndex]
					inputIndex++
					printer.Infof("%s [%s]: %s\n", prompt, defaultVal, input)
					return input
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
				printer.Errorln("failed to create region selector")
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
						printer.Errorln("region selector is nil")
					}
				}
			}()

			// Create flow
			flowState := &initFlow{}

			tc.fCtx.State = CommandState{}

			// Run test
			err := flowState.handleNeedProjectData(tc.fCtx)
			if tc.expectedError {
				s.Require().Error(err)
				if tc.testToken == "empty-list" {
					s.Equal(ErrProjectCreationCanceled, err)
				} else {
					s.Equal(ErrProjectSelectionCanceled, err)
				}
				s.Nil(tc.fCtx.State.Project)
			} else {
				s.Require().NoError(err)
				s.Require().NotNil(tc.fCtx.State.Project)
				s.True(flowState.projectStepDone)
			}
		})
	}
}

func (s *ForgeTestSuite) TestHandleNeedExistingProjectData() {
	testCases := []struct {
		name          string
		fCtx          *ForgeContext
		testToken     string
		inputs        []string
		expectedError bool
		expectedErr   error // Add expected error type
	}{
		{
			name: "Success - Single project exists",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			testToken: "test-token", // default token returns single project
			inputs: []string{
				"Y", // Confirm using the single project
			},
			expectedError: false,
		},
		{
			name: "Success - Multiple projects, select first project",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			testToken: "multiple-projects",
			inputs: []string{
				"1", // Select first project
			},
			expectedError: false,
		},
		{
			name: "Success - Multiple projects, select second project",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			testToken: "multiple-projects",
			inputs: []string{
				"2", // Select second project
			},
			expectedError: false,
		},
		{
			name: "Error - Multiple projects, cancel selection",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			testToken: "multiple-projects",
			inputs: []string{
				"q", // Cancel selection
			},
			expectedError: true,
			expectedErr:   ErrProjectSelectionCanceled,
		},
		{
			name: "Error - No projects exist",
			fCtx: &ForgeContext{
				Config: &Config{
					OrganizationID: "test-org-id",
					Credential: Credential{
						Token: "test-token",
					},
				},
			},
			testToken:     "empty-list",
			inputs:        []string{},
			expectedError: true,
			expectedErr:   ErrProjectSelectionCanceled,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Setup
			tc.fCtx.Context = s.ctx

			// Set test token
			s.testToken = tc.testToken
			defer func() { s.testToken = "" }()

			// Setup input mocking
			inputIndex := 0
			getInput = func(prompt string, defaultVal string) string {
				if inputIndex >= len(tc.inputs) {
					return defaultVal
				}
				input := tc.inputs[inputIndex]
				inputIndex++
				printer.Infof("%s [%s]: %s", prompt, defaultVal, input)
				return input
			}
			defer func() { getInput = originalGetInput }()

			// Create flow
			flowState := &initFlow{}

			tc.fCtx.State = CommandState{}

			// Run test
			err := flowState.handleNeedExistingProjectData(tc.fCtx)
			if tc.expectedError {
				s.Require().Error(err)
				s.Equal(ErrProjectSelectionCanceled, err)
				s.Nil(tc.fCtx.State.Project)
			} else {
				s.Require().NoError(err)
				s.Require().NotNil(tc.fCtx.State.Project)
				s.True(flowState.projectStepDone)
			}
		})
	}
}

func (s *ForgeTestSuite) TestCreateOrganizationCmd() {
	testCases := []struct {
		name            string
		config          Config
		cmd             *CreateOrganizationCmd
		inputs          []string // For confirmations and prompts
		expectedPrompt  []string
		expectInputFail int
		expectedError   bool
		expectedOrg     *organization
	}{
		{
			name: "Success - Create organization with all fields",
			config: Config{
				Credential: Credential{
					Token:          "test-token",
					TokenExpiresAt: time.Now().Add(1 * time.Hour),
				},
			},
			cmd: &CreateOrganizationCmd{
				Name:      "My Great Org",
				Slug:      "",
				AvatarURL: "http://test.com",
			},
			inputs: []string{
				"",  // use default
				"",  // use default
				"",  // use default
				"Y", // Confirm
			},
			expectedPrompt: []string{
				"Enter organization name",
				"Enter organization slug",
				"Enter organization avatar URL (Empty Valid)",
				"Create organization with these details? (Y/n)",
			},
			expectedError: false,
			expectedOrg: &organization{
				ID:   "test-org-id",
				Name: "Test Organization",
				Slug: "testo",
			},
		},
	}

	// Mock browser opening
	openBrowser = func(_ string) error { return nil }
	defer func() { openBrowser = originalOpenBrowser }()

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Save the test config
			err := tc.config.Save()
			s.Require().NoError(err)

			// Setup input mocking
			//nolint:nestif // Test file is ok
			if len(tc.inputs) > 0 {
				inputIndex := 0
				getInput = func(prompt string, defaultVal string) string {
					printer.Infof("%s [%s]: ", prompt, defaultVal)

					// Validate against expected prompts if defined
					if len(tc.expectedPrompt) > 0 {
						if inputIndex >= len(tc.expectedPrompt) {
							panic(eris.Errorf("More prompts than expected. Got: %s", prompt))
						}
						if prompt != tc.expectedPrompt[inputIndex] {
							panic(eris.Errorf("Unexpected prompt at index %d. Expected: %s, Got: %s",
								inputIndex, tc.expectedPrompt[inputIndex], prompt))
						}
					}

					input := tc.inputs[inputIndex]
					if input == "" {
						input = defaultVal
					}
					printer.Infoln(input)
					inputIndex++
					return input
				}
				defer func() { getInput = originalGetInput }()
			}

			// Run the command
			err = tc.cmd.Run()

			if tc.expectedError {
				s.Require().Error(err)
				if tc.config.Credential.Token == "" {
					s.Contains(err.Error(), "not logged in")
				}
			} else {
				s.Require().NoError(err)
				// Verify the organization was created by checking the config
				config, err := GetForgeConfig()
				s.Require().NoError(err)
				s.Equal(tc.expectedOrg.ID, config.OrganizationID)
			}
		})
	}
}

func (s *ForgeTestSuite) TestSwitchOrganizationCmd() {
	testCases := []struct {
		name          string
		config        Config
		cmd           *SwitchOrganizationCmd
		inputs        []string
		expectedError bool
	}{
		{
			name: "Success - Switch by slug",
			config: Config{
				Credential: Credential{
					Token:          "test-token",
					TokenExpiresAt: time.Now().Add(1 * time.Hour),
				},
			},
			cmd: &SwitchOrganizationCmd{
				Slug: "testo",
			},
			expectedError: false,
		},
		{
			name: "Error - Invalid slug",
			config: Config{
				Credential: Credential{
					Token:          "test-token",
					TokenExpiresAt: time.Now().Add(1 * time.Hour),
				},
			},
			cmd: &SwitchOrganizationCmd{
				Slug: "invalid-slug",
			},
			expectedError: true,
		},
		{
			name: "Success - Interactive selection",
			config: Config{
				Credential: Credential{
					Token:          "test-token",
					TokenExpiresAt: time.Now().Add(1 * time.Hour),
				},
			},
			cmd: &SwitchOrganizationCmd{},
			inputs: []string{
				"1", // Select first organization
			},
			expectedError: false,
		},
		{
			name: "Error - Not logged in",
			config: Config{
				Credential: Credential{
					Token: "",
				},
			},
			cmd: &SwitchOrganizationCmd{
				Slug: "testo",
			},
			expectedError: true,
		},
	}

	// Mock browser opening
	openBrowser = func(_ string) error { return nil }
	defer func() { openBrowser = originalOpenBrowser }()

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Save the test config
			err := tc.config.Save()
			s.Require().NoError(err)

			// Setup input mocking
			if len(tc.inputs) > 0 {
				inputIndex := 0
				getInput = func(prompt string, defaultVal string) string {
					if inputIndex >= len(tc.inputs) {
						return defaultVal
					}
					input := tc.inputs[inputIndex]
					inputIndex++
					printer.Infof("%s [%s]: %s", prompt, defaultVal, input)
					return input
				}
				defer func() { getInput = originalGetInput }()
			}

			// Run the command
			err = tc.cmd.Run()

			if tc.expectedError {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *ForgeTestSuite) TestCreateProjectCmd() {
	testCases := []struct {
		name                string
		config              Config
		cmd                 *CreateProjectCmd
		inputs              []string     // For confirmations and prompts
		regionSelectActions []tea.KeyMsg // Simulate region selection
		expectedPrompt      []string
		expectInputFail     int
		expectedError       bool
		expectedProj        *project
	}{
		{
			name: "Success - Create project with all fields",
			config: Config{
				Credential: Credential{
					Token:          "test-token",
					TokenExpiresAt: time.Now().Add(1 * time.Hour),
				},
				OrganizationID: "test-org-id",
			},
			cmd: &CreateProjectCmd{
				Name:      "Test Project",
				Slug:      "Test",
				AvatarURL: "http://test.com",
			},
			inputs: []string{
				"",      // name
				"testp", // take default
				"https://github.com/argus-labs/starter-game-template", // Repository URL
				"",           // repoToken (empty for public repo)
				"",           // repoPath (empty for default root path of repo)
				"10",         // tick rate
				"Y",          // enable discord notifications  NOTE: these won't show up in the console
				"test-token", // discord token                       because results are mocked
				"1234567890", // discord channel ID
				"Y",          // enable slack notifications
				"test-token", // slack token
				"1234567890", // slack channel ID
				"",           // avatar URL
			},
			regionSelectActions: []tea.KeyMsg{
				{Type: tea.KeySpace}, // select region
				{Type: tea.KeyEnter}, // confirm
			},
			expectedError: false,
			expectedProj: &project{
				ID:   "test-project-id",
				Name: "Test Project",
				Slug: "testp",
			},
		},
		{
			name: "Abort - user presses q in region selector",
			config: Config{
				Credential: Credential{
					Token:          "test-token",
					TokenExpiresAt: time.Now().Add(1 * time.Hour),
				},
				OrganizationID: "test-org-id",
			},
			cmd: &CreateProjectCmd{
				Name:      "Test Project",
				Slug:      "Test",
				AvatarURL: "http://test.com",
			},
			inputs: []string{
				"",      // name
				"testp", // take default
				"https://github.com/argus-labs/starter-game-template", // Repository URL
				"",           // repoToken (empty for public repo)
				"",           // repoPath (empty for default root path of repo)
				"10",         // tick rate
				"Y",          // enable discord notifications
				"test-token", // discord token
				"1234567890", // discord channel ID
				"Y",          // enable slack notifications
				"test-token", // slack token
				"1234567890", // slack channel ID
				"",           // avatar URL
			},
			regionSelectActions: []tea.KeyMsg{
				{Type: tea.KeyRunes, Runes: []rune{'q'}}, // abort region selection
			},
			expectedError: true,
			expectedProj:  nil,
		},
		{
			name: "Error - Need login but expired token",
			config: Config{
				Credential: Credential{
					Token:          "test-token",
					TokenExpiresAt: time.Now().Add(-1 * time.Hour),
				},
			},
			cmd: &CreateProjectCmd{
				Name:      "Test Project",
				Slug:      "testp",
				AvatarURL: "http://test.com",
			},
			expectedError: true,
			expectedProj:  nil,
		},
	}

	// Mock browser opening
	openBrowser = func(_ string) error { return nil }
	defer func() { openBrowser = originalOpenBrowser }()

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// --- TEST ISOLATION START ---
			originalDir, err := os.Getwd()
			s.Require().NoError(err)
			tmpDir, err := os.MkdirTemp("", "forge-create-cmd-test")
			s.Require().NoError(err)
			err = os.Chdir(tmpDir)
			s.Require().NoError(err)
			defer func() {
				os.Chdir(originalDir)
				os.RemoveAll(tmpDir)
			}()

			// Create minimal World project structure
			worldDir := filepath.Join(tmpDir, ".world")
			err = os.MkdirAll(worldDir, 0755)
			s.Require().NoError(err)

			// Create cardinal directory
			cardinalDir := filepath.Join(tmpDir, "cardinal")
			err = os.MkdirAll(cardinalDir, 0755)
			s.Require().NoError(err)

			// Initialize git repository
			cmd := exec.Command("git", "init")
			err = cmd.Run()
			s.Require().NoError(err)

			// Add a remote for the repository - use the same URL as in the test inputs
			cmd = exec.Command(
				"git",
				"remote",
				"add",
				"origin",
				"https://github.com/argus-labs/starter-game-template",
			)
			err = cmd.Run()
			s.Require().NoError(err)

			// Create empty world.toml
			worldTomlPath := filepath.Join(tmpDir, "world.toml")
			err = os.WriteFile(worldTomlPath, []byte(""), 0644)
			s.Require().NoError(err)

			defer func() { getInput = originalGetInput }()
			defer func() { openBrowser = originalOpenBrowser }()
			defer func() { regionSelector = nil }()
			defer func() { s.testToken = "" }()
			// --- TEST ISOLATION END ---

			// Setup test config in temp dir
			err = tc.config.Save()
			s.Require().NoError(err)

			if len(tc.inputs) > 0 {
				inputIndex := 0
				getInput = func(prompt string, defaultVal string) string {
					printer.Infof("%s [%s]: ", prompt, defaultVal)
					if inputIndex >= len(tc.inputs) {
						panic(fmt.Errorf("Input %d Failed: prompt was %q", inputIndex, prompt))
					}
					input := tc.inputs[inputIndex]
					inputIndex++
					if input == "" {
						input = defaultVal
					}
					printer.Infoln(input)
					return input
				}
			}

			regionSelector = tea.NewProgram(
				multiselect.InitialMultiselectModel(
					s.ctx,
					[]string{"us-east-1", "us-west-1", "eu-west-1"},
				),
				tea.WithInput(nil),
			)
			if regionSelector == nil {
				printer.Errorln("failed to create region selector")
			}

			go func() {
				time.Sleep(1 * time.Second)
				for _, action := range tc.regionSelectActions {
					if regionSelector != nil {
						regionSelector.Send(action)
						time.Sleep(100 * time.Millisecond)
					} else {
						printer.Errorln("region selector is nil")
					}
				}
			}()

			err = tc.cmd.Run()

			if tc.expectedError {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
				if tc.expectedProj != nil {
					config, err := GetForgeConfig()
					s.Require().NoError(err)
					s.Equal(tc.expectedProj.ID, config.ProjectID)
				}
			}
		})
	}
}

func (s *ForgeTestSuite) TestSwitchProjectCmd() {
	testCases := []struct {
		name           string
		config         Config
		cmd            *SwitchProjectCmd
		inputs         []string // For interactive project selection
		expectedError  bool
		expectedProj   *project
		expectedOutput string
	}{
		{
			name: "Success - Switch project by slug",
			config: Config{
				Credential: Credential{
					Token:          "test-token",
					TokenExpiresAt: time.Now().Add(1 * time.Hour),
				},
				OrganizationID: "test-org-id",
			},
			cmd: &SwitchProjectCmd{
				Slug: "testp",
			},
			expectedError: false,
			expectedProj: &project{
				ID:   "test-project-id",
				Name: "Test Project",
				Slug: "testp",
			},
			expectedOutput: "Switched to project: Test Project",
		},
		{
			name: "Success - Switch project interactively",
			config: Config{
				Credential: Credential{
					Token:          "test-token",
					TokenExpiresAt: time.Now().Add(1 * time.Hour),
				},
				OrganizationID: "test-org-id",
			},
			cmd: &SwitchProjectCmd{},
			inputs: []string{
				"1", // Select first project
			},
			expectedError: false,
			expectedProj: &project{
				ID:   "test-project-id",
				Name: "Test Project",
				Slug: "testp",
			},
			expectedOutput: "Switched to project: Test Project",
		},
		{
			name: "Error - Not logged in",
			config: Config{
				Credential: Credential{
					Token:          "test-token",
					TokenExpiresAt: time.Now().Add(-1 * time.Hour),
				},
			},
			cmd: &SwitchProjectCmd{
				Slug: "test-project",
			},
			expectedError: true,
			expectedProj:  nil,
		},
		{
			name: "Error - No organization selected",
			config: Config{
				Credential: Credential{
					Token:          "test-token",
					TokenExpiresAt: time.Now().Add(1 * time.Hour),
				},
			},
			cmd: &SwitchProjectCmd{
				Slug: "test-project",
			},
			expectedError: true,
			expectedProj:  nil,
		},
		{
			name: "Error - No projects exist",
			config: Config{
				Credential: Credential{
					Token:          "test-token",
					TokenExpiresAt: time.Now().Add(1 * time.Hour),
				},
				OrganizationID: "empty-org-id",
			},
			cmd: &SwitchProjectCmd{
				Slug: "test-project",
			},
			expectedError: true,
			expectedProj:  nil,
		},
		{
			name: "Error - Project selection aborted",
			config: Config{
				Credential: Credential{
					Token:          "test-token",
					TokenExpiresAt: time.Now().Add(1 * time.Hour),
				},
				OrganizationID: "test-org-id",
			},
			cmd: &SwitchProjectCmd{},
			inputs: []string{
				"q", // Abort selection
			},
			expectedError:  false,
			expectedProj:   nil,
			expectedOutput: "No project selected.",
		},
		{
			name: "Error - failed to select project",
			config: Config{
				Credential: Credential{
					Token:          "test-token",
					TokenExpiresAt: time.Now().Add(1 * time.Hour),
				},
				CurrRepoKnown: false,
				CurrRepoURL:   "https://github.com/test/repo",
				CurrRepoPath:  "/",
			},
			cmd: &SwitchProjectCmd{
				Slug: "testp",
			},
			expectedError: false,
			expectedProj:  nil,
		},
	}

	// Mock browser opening
	openBrowser = func(_ string) error { return nil }
	defer func() { openBrowser = originalOpenBrowser }()

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Save the test config
			err := tc.config.Save()
			s.Require().NoError(err)

			// Setup input mocking
			if len(tc.inputs) > 0 {
				inputIndex := 0
				getInput = func(prompt string, defaultVal string) string {
					printer.Infof("%s [%s]: ", prompt, defaultVal)

					input := tc.inputs[inputIndex]
					if input == "" {
						input = defaultVal
					}
					printer.Infoln(input)
					inputIndex++
					return input
				}
				defer func() { getInput = originalGetInput }()
			}

			// Run the command
			err = tc.cmd.Run()

			if tc.expectedError {
				s.Require().Error(err)
				switch tc.name {
				case "Error - SetupForgeCommandState fails":
					s.Contains(err.Error(), "forge command setup failed")
				case "Error - Project selection fails":
					s.Contains(err.Error(), "Failed to select project")
				}
			} else {
				s.Require().NoError(err)
				if tc.expectedProj != nil {
					// Verify the project was selected by checking the config
					config, err := GetForgeConfig()
					s.Require().NoError(err)
					s.Equal(tc.expectedProj.ID, config.ProjectID)
				}
			}
		})
	}
}

func (s *ForgeTestSuite) TestDeleteProjectCmd() {
	tests := []struct {
		name          string
		config        Config
		cmd           *DeleteProjectCmd
		expectedError bool
		expectedProj  *project
		inputs        []string
	}{
		{
			name: "Success - Delete project",
			config: Config{
				Credential: Credential{
					Token:          "test-token",
					TokenExpiresAt: time.Now().Add(1 * time.Hour),
				},
				OrganizationID: "test-org-id",
				ProjectID:      "test-project-id",
				CurrRepoKnown:  true,
				CurrRepoURL:    "https://github.com/test/repo",
				CurrRepoPath:   "/",
			},
			cmd:           &DeleteProjectCmd{},
			expectedError: false,
			expectedProj:  nil,
			inputs:        []string{"Yes"},
		},
		{
			name: "Error - SetupForgeCommandState fails",
			config: Config{
				Credential: Credential{
					Token:          "test-token",
					TokenExpiresAt: time.Now().Add(-1 * time.Hour),
				},
				OrganizationID: "test-org-id",
				ProjectID:      "test-project-id",
				CurrRepoKnown:  true,
				CurrRepoURL:    "https://github.com/test/repo",
				CurrRepoPath:   "/",
			},
			cmd:           &DeleteProjectCmd{},
			expectedError: true,
			expectedProj:  nil,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			// Set up test config
			err := tc.config.Save()
			s.Require().NoError(err)

			// Set up input mocking
			if len(tc.inputs) > 0 {
				originalGetInput = getInput
				defer func() { getInput = originalGetInput }()
				inputIndex := 0
				getInput = func(prompt string, defaultStr string) string {
					if inputIndex < len(tc.inputs) {
						input := tc.inputs[inputIndex]
						inputIndex++
						return input
					}
					return defaultStr
				}
			}

			// Run the command
			err = tc.cmd.Run()

			// Check error
			if tc.expectedError {
				s.Require().Error(err)
				s.Contains(err.Error(), "forge command setup failed")
			} else {
				s.Require().NoError(err)
			}

			// Only check project state if we didn't get a login error
			if !tc.config.Credential.TokenExpiresAt.Before(time.Now()) {
				// Get the current config after command runs
				currentConfig, err := GetForgeConfig()
				s.Require().NoError(err)

				// Check project state
				if tc.expectedProj == nil {
					s.Empty(currentConfig.ProjectID)
				} else {
					s.Equal(tc.expectedProj.ID, currentConfig.ProjectID)
				}
			}
		})
	}
}

func (s *ForgeTestSuite) TestInviteUserToOrganizationCmd() {
	tests := []struct {
		name          string
		config        Config
		cmd           *InviteUserToOrganizationCmd
		expectedError bool
	}{
		{
			name: "Success - Invite user",
			config: Config{
				Credential: Credential{
					Token:          "test-token",
					TokenExpiresAt: time.Now().Add(1 * time.Hour),
				},
				OrganizationID: "test-org-id",
				ProjectID:      "test-project-id",
				CurrRepoKnown:  true,
				CurrRepoURL:    "https://github.com/test/repo",
				CurrRepoPath:   "/",
			},
			cmd: &InviteUserToOrganizationCmd{
				ID:   "test-user-id",
				Role: "member",
			},
			expectedError: false,
		},
		{
			name: "Error - SetupForgeCommandState fails",
			config: Config{
				Credential: Credential{
					Token:          "test-token",
					TokenExpiresAt: time.Now().Add(-1 * time.Hour),
				},
				OrganizationID: "test-org-id",
				ProjectID:      "test-project-id",
				CurrRepoKnown:  true,
				CurrRepoURL:    "https://github.com/test/repo",
				CurrRepoPath:   "/",
			},
			cmd: &InviteUserToOrganizationCmd{
				ID:   "test-user-id",
				Role: "member",
			},
			expectedError: true,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			// Set up test config
			err := tc.config.Save()
			s.Require().NoError(err)

			// Run the command
			err = tc.cmd.Run()

			// Check error
			if tc.expectedError {
				s.Require().Error(err)
				s.Contains(err.Error(), "forge command setup failed")
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *ForgeTestSuite) TestChangeUserRoleInOrganizationCmd() {
	tests := []struct {
		name          string
		config        Config
		cmd           *ChangeUserRoleInOrganizationCmd
		expectedError bool
	}{
		{
			name: "Success - Change user role",
			config: Config{
				Credential: Credential{
					Token:          "test-token",
					TokenExpiresAt: time.Now().Add(1 * time.Hour),
				},
				OrganizationID: "test-org-id",
				ProjectID:      "test-project-id",
				CurrRepoKnown:  true,
				CurrRepoURL:    "https://github.com/test/repo",
				CurrRepoPath:   "/",
			},
			cmd: &ChangeUserRoleInOrganizationCmd{
				ID:   "test-user-id",
				Role: "admin",
			},
			expectedError: false,
		},
		{
			name: "Error - SetupForgeCommandState fails",
			config: Config{
				Credential: Credential{
					Token:          "test-token",
					TokenExpiresAt: time.Now().Add(-1 * time.Hour),
				},
				OrganizationID: "test-org-id",
				ProjectID:      "test-project-id",
				CurrRepoKnown:  true,
				CurrRepoURL:    "https://github.com/test/repo",
				CurrRepoPath:   "/",
			},
			cmd: &ChangeUserRoleInOrganizationCmd{
				ID:   "test-user-id",
				Role: "admin",
			},
			expectedError: true,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			// Set up test config
			err := tc.config.Save()
			s.Require().NoError(err)

			// Run the command
			err = tc.cmd.Run()

			// Check error
			if tc.expectedError {
				s.Require().Error(err)
				s.Contains(err.Error(), "forge command setup failed")
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *ForgeTestSuite) TestUpdateUserCmd() {
	tests := []struct {
		name          string
		config        Config
		cmd           *UpdateUserCmd
		expectedError bool
	}{
		{
			name: "Success - Update user with flags",
			config: Config{
				Credential: Credential{
					Token:          "test-token",
					TokenExpiresAt: time.Now().Add(1 * time.Hour),
				},
				OrganizationID: "test-org-id",
				ProjectID:      "test-project-id",
				CurrRepoKnown:  true,
				CurrRepoURL:    "https://github.com/test/repo",
				CurrRepoPath:   "/",
			},
			cmd: &UpdateUserCmd{
				Email:     "test@example.com",
				Name:      "admin",
				AvatarURL: "https://github.com/test/avatar.png",
			},
			expectedError: false,
		},
		{
			name: "Success - Update user without flags",
			config: Config{
				Credential: Credential{
					Token:          "test-token",
					TokenExpiresAt: time.Now().Add(1 * time.Hour),
				},
				OrganizationID: "test-org-id",
				ProjectID:      "test-project-id",
				CurrRepoKnown:  true,
				CurrRepoURL:    "https://github.com/test/repo",
				CurrRepoPath:   "/",
			},
			cmd:           &UpdateUserCmd{},
			expectedError: false,
		},
		{
			name: "Error - SetupForgeCommandState fails",
			config: Config{
				Credential: Credential{
					Token:          "test-token",
					TokenExpiresAt: time.Now().Add(-1 * time.Hour),
				},
				OrganizationID: "test-org-id",
				ProjectID:      "test-project-id",
				CurrRepoKnown:  true,
				CurrRepoURL:    "https://github.com/test/repo",
				CurrRepoPath:   "/",
			},
			expectedError: true,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			// Set up test config
			err := tc.config.Save()
			s.Require().NoError(err)

			// Run the command
			err = tc.cmd.Run()

			// Check error
			if tc.expectedError {
				s.Require().Error(err)
				s.Contains(err.Error(), "forge command setup failed")
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func TestForgeSuite(t *testing.T) {
	InitForgeBase("LOCAL")
	suite.Run(t, new(ForgeTestSuite))
}
