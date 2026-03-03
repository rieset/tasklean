package plane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/rie/tasklean/internal/config"
	"github.com/rie/tasklean/internal/models"
	"github.com/rie/tasklean/internal/transform"
)

var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// PlaneTracker fetches tasks from a Plane instance (self-hosted or cloud).
type PlaneTracker struct {
	client       *http.Client
	statesCache  map[string][]stateInfo       // projectID → states
	modulesCache map[string]map[string]string // projectID → moduleID→name
}

// NewPlaneTracker creates a new Plane tracker.
func NewPlaneTracker() *PlaneTracker {
	return &PlaneTracker{
		client:       &http.Client{Timeout: 30 * time.Second},
		statesCache:  make(map[string][]stateInfo),
		modulesCache: make(map[string]map[string]string),
	}
}

// FetchOptions holds filters for Plane FetchTasks.
type FetchOptions struct {
	From        string
	Assignee    string
	Debug       bool
	SkipModules bool // skip module resolution to reduce API calls
}

// FetchTasks fetches issues from Plane API.
// Self-hosted: GET /api/v1/workspaces/{slug}/projects/{project_id}/issues/
// Cloud:       GET /api/v1/workspaces/{slug}/projects/{project_id}/work-items/
// No state filter — fetches all issues including backlog (state__group: backlog).
func (p *PlaneTracker) FetchTasks(ctx context.Context, cfg *config.RemoteConfig, opts *FetchOptions) ([]*models.Task, error) {
	if cfg.Workspace == "" || cfg.Project == "" {
		return nil, fmt.Errorf("Plane requires workspace and project in remote config (use: remote config <name> -w <slug> -p <id>)")
	}

	apiBase := strings.TrimSuffix(cfg.URL, "/") + "/api/v1"
	endpoint := "issues"
	authHeader := "X-Api-Key"
	if cfg.Cloud {
		endpoint = "work-items"
		authHeader = "X-API-Key"
	}

	projectID := cfg.Project
	if !uuidRegex.MatchString(projectID) {
		resolved, err := p.resolveProjectID(ctx, apiBase, cfg.Workspace, cfg.Token, authHeader, projectID)
		if err != nil {
			return nil, fmt.Errorf("project %q: %w (hint: use project UUID from Plane URL, e.g. xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)", projectID, err)
		}
		projectID = resolved
	}

	assigneeID := ""
	if opts != nil && opts.Assignee != "" {
		id, err := p.resolveAssigneeByEmail(ctx, apiBase, cfg, authHeader, opts.Assignee)
		if err != nil {
			return nil, fmt.Errorf("resolve assignee %q: %w", opts.Assignee, err)
		}
		assigneeID = id
	}

	var allIssues []issue
	offset := 0
	limit := 100

	for {
		u := fmt.Sprintf("%s/workspaces/%s/projects/%s/%s/",
			apiBase, url.PathEscape(cfg.Workspace), url.PathEscape(projectID), endpoint)
		reqURL, err := url.Parse(u)
		if err != nil {
			return nil, fmt.Errorf("parse URL: %w", err)
		}
		q := reqURL.Query()
		q.Set("limit", fmt.Sprintf("%d", limit))
		q.Set("offset", fmt.Sprintf("%d", offset))
		q.Set("expand", "module,module_details,assignees")
		if assigneeID != "" {
			q.Set("assignees", assigneeID)
		}
		reqURL.RawQuery = q.Encode()

		req, err := http.NewRequestWithContext(ctx, "GET", reqURL.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set(authHeader, cfg.Token)

		resp, err := p.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request: %w", err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read response: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			msg := string(body)
			if resp.StatusCode == http.StatusNotFound && strings.Contains(msg, "Page not found") {
				return nil, fmt.Errorf("API error 404: project ID must be a UUID (found %q). Get it from Plane: open project → URL shows id", cfg.Project)
			}
			return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, msg)
		}

		if offset == 0 && opts != nil && opts.Debug {
			var first interface{}
			var rawList []interface{}
			if err := json.Unmarshal(body, &rawList); err == nil && len(rawList) > 0 {
				first = rawList[0]
			} else {
				var wrapped struct {
					Results []interface{} `json:"results"`
				}
				if err := json.Unmarshal(body, &wrapped); err == nil && len(wrapped.Results) > 0 {
					first = wrapped.Results[0]
				}
			}
			if first != nil {
				b, _ := json.MarshalIndent(first, "", "  ")
				fmt.Fprintf(os.Stderr, "[debug] first issue raw from API:\n%s\n", b)
			}
		}

		items, err := parseIssuesResponse(body)
		if err != nil {
			return nil, fmt.Errorf("parse response: %w", err)
		}

		allIssues = append(allIssues, items...)

		if len(items) < limit {
			break
		}
		offset += limit
	}

	debug := opts != nil && opts.Debug
	skipModules := opts != nil && opts.SkipModules

	memberEmails, _ := p.fetchProjectMemberEmails(ctx, apiBase, cfg, authHeader, projectID)

	var issueToModule map[string]string
	if !skipModules {
		issueToModule, _ = p.fetchIssueToModuleMap(ctx, apiBase, cfg, authHeader, projectID, debug)
	}

	tasks := make([]*models.Task, 0, len(allIssues))
	for _, i := range allIssues {
		tasks = append(tasks, issueToTask(i, memberEmails, issueToModule))
	}
	return tasks, nil
}

// resolveProjectID converts project identifier (e.g. "PROJ") to UUID by listing projects.
func (p *PlaneTracker) resolveProjectID(ctx context.Context, apiBase, workspace, token, authHeader, identifier string) (string, error) {
	u := fmt.Sprintf("%s/workspaces/%s/projects/", apiBase, url.PathEscape(workspace))
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set(authHeader, token)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("list projects: %d %s", resp.StatusCode, string(body))
	}

	var result struct {
		Results []struct {
			ID         string `json:"id"`
			Identifier string `json:"identifier"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		var list []struct {
			ID         string `json:"id"`
			Identifier string `json:"identifier"`
		}
		if err2 := json.Unmarshal(body, &list); err2 != nil {
			return "", fmt.Errorf("parse projects: %w", err)
		}
		for _, prj := range list {
			if strings.EqualFold(prj.Identifier, identifier) {
				return prj.ID, nil
			}
		}
		return "", fmt.Errorf("project %q not found in workspace", identifier)
	}

	identifier = strings.ToUpper(identifier)
	for _, prj := range result.Results {
		if strings.ToUpper(prj.Identifier) == identifier {
			return prj.ID, nil
		}
	}
	return "", fmt.Errorf("project %q not found in workspace", identifier)
}

// fetchProjectMemberEmails returns map of member ID -> email for resolving assignee_ids.
func (p *PlaneTracker) fetchProjectMemberEmails(ctx context.Context, apiBase string, cfg *config.RemoteConfig, authHeader, projectID string) (map[string]string, error) {
	u := fmt.Sprintf("%s/workspaces/%s/projects/%s/members/",
		apiBase, url.PathEscape(cfg.Workspace), url.PathEscape(projectID))
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set(authHeader, cfg.Token)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	type member struct {
		ID     string `json:"id"`
		Member struct {
			Email string `json:"email"`
		} `json:"member"`
	}

	var members []member
	if err := json.Unmarshal(body, &members); err != nil {
		return nil, nil
	}
	m := make(map[string]string, len(members))
	for _, mem := range members {
		if mem.Member.Email != "" {
			m[mem.ID] = mem.Member.Email
		}
	}
	return m, nil
}

// fetchProjectModules returns map of module ID -> name for resolving module_ids.
func (p *PlaneTracker) fetchProjectModules(ctx context.Context, apiBase string, cfg *config.RemoteConfig, authHeader, projectID string, debug bool) (map[string]string, error) {
	if cached, ok := p.modulesCache[projectID]; ok {
		return cached, nil
	}
	m := make(map[string]string)
	offset := 0
	limit := 100

	for {
		u := fmt.Sprintf("%s/workspaces/%s/projects/%s/modules/",
			apiBase, url.PathEscape(cfg.Workspace), url.PathEscape(projectID))
		reqURL, err := url.Parse(u)
		if err != nil {
			return nil, err
		}
		q := reqURL.Query()
		q.Set("limit", fmt.Sprintf("%d", limit))
		q.Set("offset", fmt.Sprintf("%d", offset))
		reqURL.RawQuery = q.Encode()

		req, err := http.NewRequestWithContext(ctx, "GET", reqURL.String(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set(authHeader, cfg.Token)

		resp, err := p.client.Do(req)
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			if offset == 0 && debug {
				fmt.Fprintf(os.Stderr, "[debug] modules API %d: %s\n", resp.StatusCode, string(body))
			}
			if offset == 0 {
				return nil, nil
			}
			break
		}

		var result struct {
			Results []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"results"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			var list []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}
			if err2 := json.Unmarshal(body, &list); err2 != nil {
				if offset == 0 {
					return nil, nil
				}
				break
			}
			for _, mod := range list {
				if mod.Name != "" {
					m[mod.ID] = mod.Name
				}
			}
			if len(list) < limit {
				break
			}
			offset += limit
			continue
		}
		for _, mod := range result.Results {
			if mod.Name != "" {
				m[mod.ID] = mod.Name
			}
		}
		if len(result.Results) < limit {
			break
		}
		offset += limit
	}
	p.modulesCache[projectID] = m
	return m, nil
}

// fetchIssueToModuleMap returns map of issue ID -> module name.
// Self-hosted Plane does not include module in issue response; we get it via module-issues API.
func (p *PlaneTracker) fetchIssueToModuleMap(ctx context.Context, apiBase string, cfg *config.RemoteConfig, authHeader, projectID string, debug bool) (map[string]string, error) {
	modules, err := p.fetchProjectModules(ctx, apiBase, cfg, authHeader, projectID, debug)
	if err != nil {
		if debug {
			fmt.Fprintf(os.Stderr, "[debug] fetchProjectModules error: %v\n", err)
		}
		return nil, nil
	}
	if debug {
		fmt.Fprintf(os.Stderr, "[debug] modules: %d found\n", len(modules))
	}
	if len(modules) == 0 {
		return make(map[string]string), nil
	}
	m := make(map[string]string)
	first := true
	for moduleID, moduleName := range modules {
		dump := debug && first
		issueIDs := p.fetchModuleIssueIDs(ctx, apiBase, cfg, authHeader, projectID, moduleID, dump)
		if debug {
			fmt.Fprintf(os.Stderr, "[debug] module %q (%s): %d issues\n", moduleName, moduleID, len(issueIDs))
		}
		first = false
		for _, issueID := range issueIDs {
			if m[issueID] == "" {
				m[issueID] = moduleName
			}
		}
	}
	if debug {
		fmt.Fprintf(os.Stderr, "[debug] issueToModule map: %d entries\n", len(m))
	}
	return m, nil
}

func (p *PlaneTracker) fetchModuleIssueIDs(ctx context.Context, apiBase string, cfg *config.RemoteConfig, authHeader, projectID, moduleID string, dumpFirst bool) []string {
	var ids []string
	offset := 0
	limit := 100

	for {
		u := fmt.Sprintf("%s/workspaces/%s/projects/%s/modules/%s/module-issues/",
			apiBase, url.PathEscape(cfg.Workspace), url.PathEscape(projectID), url.PathEscape(moduleID))
		reqURL, err := url.Parse(u)
		if err != nil {
			return nil
		}
		q := reqURL.Query()
		q.Set("limit", fmt.Sprintf("%d", limit))
		q.Set("offset", fmt.Sprintf("%d", offset))
		reqURL.RawQuery = q.Encode()

		req, err := http.NewRequestWithContext(ctx, "GET", reqURL.String(), nil)
		if err != nil {
			return nil
		}
		req.Header.Set(authHeader, cfg.Token)

		resp, err := p.client.Do(req)
		if err != nil {
			return nil
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			break
		}
		if resp.StatusCode != http.StatusOK {
			if dumpFirst {
				fmt.Fprintf(os.Stderr, "[debug] module-issues API %d: %s\n", resp.StatusCode, string(body))
			}
			break
		}
		if dumpFirst && offset == 0 {
			fmt.Fprintf(os.Stderr, "[debug] module-issues raw (first 500 chars): %s\n", truncate(string(body), 500))
		}

		// self-hosted returns full issue objects with "id"; cloud returns {issue: uuid}
		var result struct {
			Results []struct {
				ID    string `json:"id"`    // self-hosted: full issue object
				Issue string `json:"issue"` // cloud: reference to issue
			} `json:"results"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			break
		}
		for _, mi := range result.Results {
			id := mi.Issue
			if id == "" {
				id = mi.ID
			}
			if id != "" {
				ids = append(ids, id)
			}
		}
		if len(result.Results) < limit {
			break
		}
		offset += limit
	}
	return ids
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// resolveAssigneeByEmail looks up a member UUID by email from project members.
// Self-hosted v1 API: GET /api/v1/workspaces/{slug}/projects/{id}/members/
func (p *PlaneTracker) resolveAssigneeByEmail(ctx context.Context, apiBase string, cfg *config.RemoteConfig, authHeader, email string) (string, error) {
	projectID := cfg.Project
	if !uuidRegex.MatchString(projectID) {
		resolved, err := p.resolveProjectID(ctx, apiBase, cfg.Workspace, cfg.Token, authHeader, projectID)
		if err != nil {
			return "", err
		}
		projectID = resolved
	}
	u := fmt.Sprintf("%s/workspaces/%s/projects/%s/members/",
		apiBase, url.PathEscape(cfg.Workspace), url.PathEscape(projectID))
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set(authHeader, cfg.Token)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("members API %d: %s", resp.StatusCode, string(body))
	}

	type member struct {
		ID     string `json:"id"`
		Member struct {
			Email string `json:"email"`
		} `json:"member"`
	}

	var members []member
	if err := json.Unmarshal(body, &members); err != nil {
		return "", fmt.Errorf("parse members: %w", err)
	}

	email = strings.ToLower(strings.TrimSpace(email))
	for _, m := range members {
		if strings.ToLower(m.Member.Email) == email {
			return m.ID, nil
		}
	}
	return "", fmt.Errorf("no member with email %q in project", email)
}

func parseIssuesResponse(body []byte) ([]issue, error) {
	var list issuesResponse
	if err := json.Unmarshal(body, &list); err == nil && len(list.Results) > 0 {
		return list.Results, nil
	}
	var direct []issue
	if err := json.Unmarshal(body, &direct); err != nil {
		return nil, err
	}
	return direct, nil
}

func issueToTask(i issue, memberEmails map[string]string, issueToModule map[string]string) *models.Task {
	createdAt, _ := time.Parse(time.RFC3339, i.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, i.UpdatedAt)

	desc := i.DescriptionStripped
	if desc == "" {
		desc = i.DescriptionHTML
	}
	desc = transform.TransformDescription(desc)

	moduleName := ""
	if i.Module != nil && i.Module.Name != "" {
		moduleName = i.Module.Name
	} else if len(i.ModuleDetails) > 0 {
		moduleName = i.ModuleDetails[0].Name
	} else if issueToModule != nil {
		moduleName = issueToModule[i.ID]
	}
	assignees := extractAssignees(i, memberEmails)
	return &models.Task{
		ID:          i.ID,
		Title:       i.Name,
		Description: desc,
		Status:      mapStateGroupToStatus(i.StateGroup),
		Module:      moduleName,
		Assignees:   assignees,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}

func extractAssignees(i issue, memberEmails map[string]string) []string {
	seen := make(map[string]bool)
	var out []string

	for _, a := range i.AssigneeDetails {
		email := a.Email
		if email == "" {
			email = a.Member.Email
		}
		if email != "" && !seen[email] {
			seen[email] = true
			out = append(out, email)
		}
	}
	if len(out) > 0 {
		return out
	}

	for _, a := range i.Assignees {
		email := a.Email
		if email == "" {
			email = a.Member.Email
		}
		if email != "" && !seen[email] {
			seen[email] = true
			out = append(out, email)
		}
	}
	if len(out) > 0 {
		return out
	}

	if memberEmails != nil {
		for _, id := range i.AssigneeIDs {
			if email := memberEmails[id]; email != "" && !seen[email] {
				seen[email] = true
				out = append(out, email)
			}
		}
	}
	return out
}

func mapStateGroupToStatus(group string) models.TaskStatus {
	switch group {
	case "completed":
		return models.StatusDone
	case "cancelled":
		return models.StatusCancelled
	case "started":
		return models.StatusInProgress
	case "unstarted":
		return models.StatusTodo
	case "backlog":
		return models.StatusBacklog
	default:
		return models.StatusTodo
	}
}

type issue struct {
	ID                  string         `json:"id"`
	Name                string         `json:"name"`
	DescriptionHTML     string         `json:"description_html"`
	DescriptionStripped string         `json:"description_stripped"`
	State               string         `json:"state"`
	StateGroup          string         `json:"state__group"`
	Module              *flexModule    `json:"module"` // can be UUID string or {id,name} object
	ModuleDetails       []moduleInfo   `json:"module_details"`
	ModuleIDs           []string       `json:"module_ids"` // self-hosted: array of module UUIDs
	AssigneeDetails     []assigneeInfo `json:"assignee_details"`
	Assignees           []assigneeInfo `json:"assignees"`    // expanded objects
	AssigneeIDs         []string       `json:"assignee_ids"` // self-hosted: array of UUIDs
	CreatedAt           string         `json:"created_at"`
	UpdatedAt           string         `json:"updated_at"`
}

// flexModule unmarshals from either "uuid-string" or {"id":"...","name":"..."}.
type flexModule struct {
	ID   string
	Name string
}

func (f *flexModule) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		f.ID = s
		return nil
	}
	var obj struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	f.ID = obj.ID
	f.Name = obj.Name
	return nil
}

type moduleInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type assigneeInfo struct {
	ID     string `json:"id"`
	Email  string `json:"email"`
	Member struct {
		Email string `json:"email"`
	} `json:"member"`
}

type issuesResponse struct {
	Results []issue `json:"results"`
}

type stateInfo struct {
	ID    string `json:"id"`
	Group string `json:"group"`
}

// UpdateTask updates an existing issue in Plane.
func (p *PlaneTracker) UpdateTask(ctx context.Context, cfg *config.RemoteConfig, task *models.Task) error {
	if cfg.Workspace == "" || cfg.Project == "" {
		return fmt.Errorf("Plane requires workspace and project in remote config")
	}
	if !uuidRegex.MatchString(task.ID) {
		return fmt.Errorf("task ID %q is not a valid UUID (cannot update)", task.ID)
	}

	apiBase, endpoint, authHeader, projectID, err := p.planeRequestSetup(ctx, cfg)
	if err != nil {
		return err
	}

	stateID, err := p.statusToStateID(ctx, apiBase, cfg, authHeader, projectID, task.Status)
	if err != nil {
		return fmt.Errorf("resolve state for %s: %w", task.Status, err)
	}

	body := map[string]interface{}{
		"name":             task.Title,
		"description_html": transform.MarkdownToHTML(task.Description),
		"state":            stateID,
	}
	jsonBody, _ := json.Marshal(body)

	u := fmt.Sprintf("%s/workspaces/%s/projects/%s/%s/%s/",
		apiBase, url.PathEscape(cfg.Workspace), url.PathEscape(projectID), endpoint, task.ID)
	req, err := http.NewRequestWithContext(ctx, "PATCH", u, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set(authHeader, cfg.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(data))
	}
	return nil
}

// CreateTask creates a new issue in Plane. Returns the created task with real ID.
func (p *PlaneTracker) CreateTask(ctx context.Context, cfg *config.RemoteConfig, task *models.Task) (*models.Task, error) {
	if cfg.Workspace == "" || cfg.Project == "" {
		return nil, fmt.Errorf("Plane requires workspace and project in remote config")
	}

	apiBase, endpoint, authHeader, projectID, err := p.planeRequestSetup(ctx, cfg)
	if err != nil {
		return nil, err
	}

	stateID, err := p.statusToStateID(ctx, apiBase, cfg, authHeader, projectID, task.Status)
	if err != nil {
		return nil, fmt.Errorf("resolve state for %s: %w", task.Status, err)
	}

	assigneeIDs := make([]string, 0, len(task.Assignees))
	for _, email := range task.Assignees {
		id, err := p.resolveAssigneeByEmail(ctx, apiBase, cfg, authHeader, email)
		if err != nil {
			continue
		}
		assigneeIDs = append(assigneeIDs, id)
	}

	body := map[string]interface{}{
		"name":             task.Title,
		"description_html": transform.MarkdownToHTML(task.Description),
		"state":            stateID,
	}
	if len(assigneeIDs) > 0 {
		body["assignees"] = assigneeIDs
	}
	jsonBody, _ := json.Marshal(body)

	u := fmt.Sprintf("%s/workspaces/%s/projects/%s/%s/",
		apiBase, url.PathEscape(cfg.Workspace), url.PathEscape(projectID), endpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", u, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set(authHeader, cfg.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(data))
	}

	var created issue
	if err := json.Unmarshal(data, &created); err != nil {
		return nil, fmt.Errorf("parse created issue: %w", err)
	}

	desc := task.Description
	if created.DescriptionStripped != "" {
		desc = transform.TransformDescription(created.DescriptionStripped)
	} else if created.DescriptionHTML != "" {
		desc = transform.TransformDescription(created.DescriptionHTML)
	}

	status := mapStateGroupToStatus(created.StateGroup)
	if status == "" {
		status = task.Status
	}
	now := time.Now().UTC()
	return &models.Task{
		ID:          created.ID,
		Title:       created.Name,
		Description: desc,
		Status:      status,
		Module:      task.Module,
		Assignees:   task.Assignees,
		Remote:      task.Remote,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// SyncTaskModule updates the module assignment of a task in Plane.
// fromModule is the current module name in Plane, toModule is the desired one.
// Modules are matched case-insensitively and with space/dash normalization.
func (p *PlaneTracker) SyncTaskModule(ctx context.Context, cfg *config.RemoteConfig, taskID, fromModule, toModule string) error {
	if normalizeModuleName(fromModule) == normalizeModuleName(toModule) {
		return nil
	}

	apiBase, _, authHeader, projectID, err := p.planeRequestSetup(ctx, cfg)
	if err != nil {
		return err
	}

	modules, err := p.fetchProjectModules(ctx, apiBase, cfg, authHeader, projectID, false)
	if err != nil {
		return fmt.Errorf("fetch modules: %w", err)
	}

	if fromModule != "" {
		if moduleID := findModuleIDByName(modules, fromModule); moduleID != "" {
			if err := p.removeIssueFromModule(ctx, apiBase, cfg, authHeader, projectID, moduleID, taskID); err != nil {
				return fmt.Errorf("remove from module %q: %w", fromModule, err)
			}
		}
	}

	if toModule != "" {
		moduleID := findModuleIDByName(modules, toModule)
		if moduleID == "" {
			return fmt.Errorf("module %q not found in Plane", toModule)
		}
		if err := p.addIssueToModule(ctx, apiBase, cfg, authHeader, projectID, moduleID, taskID); err != nil {
			return fmt.Errorf("add to module %q: %w", toModule, err)
		}
	}

	return nil
}

func (p *PlaneTracker) addIssueToModule(ctx context.Context, apiBase string, cfg *config.RemoteConfig, authHeader, projectID, moduleID, issueID string) error {
	u := fmt.Sprintf("%s/workspaces/%s/projects/%s/modules/%s/module-issues/",
		apiBase, url.PathEscape(cfg.Workspace), url.PathEscape(projectID), url.PathEscape(moduleID))

	jsonBody, _ := json.Marshal(map[string]interface{}{"issues": []string{issueID}})
	req, err := http.NewRequestWithContext(ctx, "POST", u, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set(authHeader, cfg.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(data))
	}
	return nil
}

// removeIssueFromModule removes a task from a module.
// Self-hosted Plane returns full issue objects from module-issues, so the issue UUID
// is used directly as the pk in the DELETE URL.
func (p *PlaneTracker) removeIssueFromModule(ctx context.Context, apiBase string, cfg *config.RemoteConfig, authHeader, projectID, moduleID, issueID string) error {
	u := fmt.Sprintf("%s/workspaces/%s/projects/%s/modules/%s/module-issues/%s/",
		apiBase, url.PathEscape(cfg.Workspace), url.PathEscape(projectID), url.PathEscape(moduleID), url.PathEscape(issueID))

	req, err := http.NewRequestWithContext(ctx, "DELETE", u, nil)
	if err != nil {
		return err
	}
	req.Header.Set(authHeader, cfg.Token)

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(data))
	}
	return nil
}

// findModuleIDByName finds a module ID by name using case-insensitive and space/dash normalization.
func findModuleIDByName(modules map[string]string, name string) string {
	normalized := normalizeModuleName(name)
	for id, n := range modules {
		if normalizeModuleName(n) == normalized {
			return id
		}
	}
	return ""
}

// normalizeModuleName lowercases and replaces spaces with dashes for comparison.
func normalizeModuleName(s string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(s), " ", "-"))
}

func (p *PlaneTracker) planeRequestSetup(ctx context.Context, cfg *config.RemoteConfig) (apiBase, endpoint, authHeader, projectID string, err error) {
	apiBase = strings.TrimSuffix(cfg.URL, "/") + "/api/v1"
	endpoint = "issues"
	authHeader = "X-Api-Key"
	if cfg.Cloud {
		endpoint = "work-items"
		authHeader = "X-API-Key"
	}
	projectID = cfg.Project
	if !uuidRegex.MatchString(projectID) {
		projectID, err = p.resolveProjectID(ctx, apiBase, cfg.Workspace, cfg.Token, authHeader, projectID)
		if err != nil {
			return "", "", "", "", err
		}
	}
	return apiBase, endpoint, authHeader, projectID, nil
}

func (p *PlaneTracker) fetchProjectStates(ctx context.Context, apiBase string, cfg *config.RemoteConfig, authHeader, projectID string) ([]stateInfo, error) {
	if cached, ok := p.statesCache[projectID]; ok {
		return cached, nil
	}

	u := fmt.Sprintf("%s/workspaces/%s/projects/%s/states/",
		apiBase, url.PathEscape(cfg.Workspace), url.PathEscape(projectID))
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set(authHeader, cfg.Token)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("states API %d: %s", resp.StatusCode, string(data))
	}

	var states []stateInfo
	if err := json.Unmarshal(data, &states); err != nil {
		var wrapped struct {
			Results []stateInfo `json:"results"`
		}
		if err2 := json.Unmarshal(data, &wrapped); err2 != nil {
			return nil, fmt.Errorf("parse states: %w", err)
		}
		states = wrapped.Results
	}
	p.statesCache[projectID] = states
	return states, nil
}

func (p *PlaneTracker) statusToStateID(ctx context.Context, apiBase string, cfg *config.RemoteConfig, authHeader, projectID string, status models.TaskStatus) (string, error) {
	states, err := p.fetchProjectStates(ctx, apiBase, cfg, authHeader, projectID)
	if err != nil {
		return "", err
	}
	targetGroup := statusToStateGroup(status)
	for _, s := range states {
		if s.Group == targetGroup {
			return s.ID, nil
		}
	}
	if len(states) > 0 {
		return states[0].ID, nil
	}
	return "", fmt.Errorf("no states in project")
}

func statusToStateGroup(status models.TaskStatus) string {
	switch status {
	case models.StatusBacklog:
		return "backlog"
	case models.StatusTodo:
		return "unstarted"
	case models.StatusInProgress, models.StatusInReview:
		return "started"
	case models.StatusDone:
		return "completed"
	case models.StatusCancelled:
		return "cancelled"
	default:
		return "unstarted"
	}
}
