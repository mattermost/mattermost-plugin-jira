// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	jira "github.com/andygrunwald/go-jira"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

type testRHSCloudClient struct {
	testClient
	mu          sync.Mutex
	statuses    []*JiraStatus
	categories  []*JiraStatusCategory
	search      *CloudSearchResult
	searchErr   error
	searchCalls int
	lastSearch  CloudSearchParams
	statusCalls int
	catCalls    int
}

func (c *testRHSCloudClient) ListStatuses() ([]*JiraStatus, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.statusCalls++
	return c.statuses, nil
}

func (c *testRHSCloudClient) ListStatusCategories() ([]*JiraStatusCategory, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.catCalls++
	return c.categories, nil
}

func (c *testRHSCloudClient) SearchJQL(params CloudSearchParams) (*CloudSearchResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.searchCalls++
	c.lastSearch = params
	if c.searchErr != nil {
		return nil, c.searchErr
	}
	if c.search == nil {
		return &CloudSearchResult{Issues: nil, IsLast: true}, nil
	}
	return c.search, nil
}

type rhsTestCloudInstance struct {
	testInstance
	client      Client
	jiraBaseURL string
	apiURL      string
}

func (i rhsTestCloudInstance) GetURL() string {
	if i.apiURL != "" {
		return i.apiURL
	}
	return i.testInstance.GetURL()
}

func (i rhsTestCloudInstance) GetJiraBaseURL() string {
	if i.jiraBaseURL != "" {
		return i.jiraBaseURL
	}
	return i.testInstance.GetJiraBaseURL()
}

func (i rhsTestCloudInstance) GetClient(*Connection) (Client, error) {
	return i.client, nil
}

func (i rhsTestCloudInstance) Common() *InstanceCommon {
	return i.InstanceCommon.Common()
}

type rhsErrNotFoundUserStore struct {
	mockUserStore
}

func (rhsErrNotFoundUserStore) LoadConnection(types.ID, types.ID) (*Connection, error) {
	return nil, kvstore.ErrNotFound
}

// Production LoadConnection often yields a zero Connection (nil OAuth2Token)
// instead of kvstore.ErrNotFound when the user is not connected.
type rhsNilOAuthTokenUserStore struct {
	mockUserStore
}

func (rhsNilOAuthTokenUserStore) LoadConnection(types.ID, types.ID) (*Connection, error) {
	return &Connection{}, nil
}

func installRHSCloudOAuth(t *testing.T, p *Plugin) *cloudOAuthInstance {
	t.Helper()
	oi := &cloudOAuthInstance{
		InstanceCommon: newInstanceCommon(p, CloudOAuthInstanceType, types.ID("https://oauth.example.atlassian.net")),
		JiraBaseURL:    "https://oauth.example.atlassian.net",
	}
	storeRHSInstance(t, p, oi)
	return oi
}

func setupRHSHTTPPlugin(t *testing.T, api *plugintest.API) *Plugin {
	t.Helper()
	api.On("LogWarn", mockAnythingOfTypeBatch("string", 11)...).Maybe().Return()
	api.On("LogDebug", mockAnythingOfTypeBatch("string", 9)...).Maybe().Return()
	p := setupTestPlugin(api)
	p.updateConfig(func(conf *config) {
		conf.maxAttachmentSize = defaultMaxAttachmentSize
	})
	return p
}

func storeRHSInstance(t *testing.T, p *Plugin, instance Instance) {
	t.Helper()
	store, ok := p.instanceStore.(*mockInstanceStoreKV)
	require.True(t, ok)
	store.kv.Store(instance.GetID(), instance)
}

func decodeRHSError(t *testing.T, w *httptest.ResponseRecorder) rhsJSONError {
	t.Helper()
	require.Equal(t, "application/json", w.Result().Header.Get("Content-Type"))
	var body rhsJSONError
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body
}

func fixtureCategoriesOmitDone() []*JiraStatusCategory {
	return []*JiraStatusCategory{
		{ID: 1, Key: statusCategoryKeyUndefined, Name: "No Category"},
		{ID: 2, Key: statusCategoryKeyNew, Name: "To Do"},
		{ID: 4, Key: statusCategoryKeyIndeterminate, Name: "In Progress"},
		// done deliberately omitted
	}
}

func fixtureCanonicalCategories(t *testing.T) []*JiraStatusCategory {
	t.Helper()
	var cats []*JiraStatusCategory
	require.NoError(t, json.Unmarshal(loadRHSTestdata(t, "rhs-statuscategory.json"), &cats))
	return cats
}

func fixtureStatuses(t *testing.T) []*JiraStatus {
	t.Helper()
	var statuses []*JiraStatus
	require.NoError(t, json.Unmarshal(loadRHSTestdata(t, "rhs-status.json"), &statuses))
	return statuses
}

func installRHSUserCloud(t *testing.T, p *Plugin, client *testRHSCloudClient) rhsTestCloudInstance {
	t.Helper()
	inst := rhsTestCloudInstance{
		testInstance: testInstance{
			InstanceCommon: InstanceCommon{
				InstanceID: types.ID("https://cloud.example.atlassian.net"),
				Type:       CloudInstanceType,
				Plugin:     p,
			},
		},
		client:      client,
		jiraBaseURL: "https://cloud.example.atlassian.net",
		apiURL:      "https://api.atlassian.com/ex/jira/CLOUDID",
	}
	storeRHSInstance(t, p, inst)
	p.userStore = mockUserStore{}
	return inst
}

func doRHSHTTPGet(t *testing.T, p *Plugin, path, userID string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	if userID != "" {
		request.Header.Set(HeaderMattermostUserID, userID)
	}
	w := httptest.NewRecorder()
	p.ServeHTTP(&plugin.Context{}, w, request)
	return w
}

func tes41Issue() jira.Issue {
	return jira.Issue{
		Key: "TES-41",
		Fields: &jira.IssueFields{
			Summary: "Fix login redirect",
			Status: &jira.Status{
				Name:           "In Progress",
				StatusCategory: jira.StatusCategory{Key: statusCategoryKeyIndeterminate},
			},
			Priority: &jira.Priority{Name: "High"},
			Type: jira.IssueType{
				Name:    "Bug",
				IconURL: "https://example.atlassian.net/images/icons/issuetypes/bug.svg",
			},
			Project:  jira.Project{Key: "TES"},
			Assignee: &jira.User{DisplayName: "Ada"},
			Reporter: &jira.User{DisplayName: "Bea"},
			Labels:   []string{"rhs"},
		},
	}
}

func TestRHSHTTPGetIssuesHappyPathDTO(t *testing.T) {
	api := &plugintest.API{}
	p := setupRHSHTTPPlugin(t, api)
	client := &testRHSCloudClient{
		statuses:   fixtureStatuses(t),
		categories: fixtureCanonicalCategories(t),
		search: &CloudSearchResult{
			Issues:        []jira.Issue{tes41Issue()},
			NextPageToken: "",
			IsLast:        true,
		},
	}
	inst := installRHSUserCloud(t, p, client)

	w := doRHSHTTPGet(t, p, makeAPIRoute(routeAPIRHSIssues)+"?instance_id="+string(inst.GetID()), "connected_user")
	require.Equal(t, http.StatusOK, w.Result().StatusCode)
	require.Equal(t, "application/json", w.Result().Header.Get("Content-Type"))

	var body rhsIssuesResult
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Issues, 1)
	assert.Equal(t, "TES-41", body.Issues[0].Key)
	assert.Equal(t, "Fix login redirect", body.Issues[0].Summary)
	assert.Equal(t, "https://cloud.example.atlassian.net/browse/TES-41", body.Issues[0].BrowseURL)
	assert.Equal(t, "In Progress", body.Issues[0].Status.Name)
	assert.Equal(t, "indeterminate", body.Issues[0].Status.CategoryKey)
	assert.Equal(t, "Bug", body.Issues[0].IssueType)
	assert.Equal(t, "https://example.atlassian.net/images/icons/issuetypes/bug.svg", body.Issues[0].IssueTypeIconURL)
	assert.Equal(t, "TES", body.Issues[0].Project)
	require.GreaterOrEqual(t, len(body.Tabs), 1)
	assert.Equal(t, RHSTabKindAssigned, body.Tabs[0].Kind)
	assert.Equal(t, rhsAssignedTabName, body.Tabs[0].Name)

	client.mu.Lock()
	defer client.mu.Unlock()
	assert.Equal(t, 1, client.searchCalls)
	assert.Equal(t, 20, client.lastSearch.MaxResults)
	assert.Equal(t, rhsSearchFields, client.lastSearch.Fields)
	assert.Equal(t, "assignee = currentUser() AND statusCategory != done ORDER BY updated DESC, key ASC", client.lastSearch.JQL)
	assert.Equal(t, "", client.lastSearch.NextPageToken)
}

func TestRHSHTTPGetIssuesCursorAndIsLast(t *testing.T) {
	api := &plugintest.API{}
	p := setupRHSHTTPPlugin(t, api)
	client := &testRHSCloudClient{
		statuses:   fixtureStatuses(t),
		categories: fixtureCanonicalCategories(t),
		search: &CloudSearchResult{
			Issues:        []jira.Issue{tes41Issue()},
			NextPageToken: "",
			IsLast:        true,
		},
	}
	inst := installRHSUserCloud(t, p, client)

	w := doRHSHTTPGet(t, p, makeAPIRoute(routeAPIRHSIssues)+"?instance_id="+string(inst.GetID())+"&next_page_token=opaque-token-page-2", "connected_user")
	require.Equal(t, http.StatusOK, w.Result().StatusCode)

	var body rhsIssuesResult
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.True(t, body.IsLast)
	assert.Equal(t, "", body.NextPageToken)

	client.mu.Lock()
	defer client.mu.Unlock()
	assert.Equal(t, "opaque-token-page-2", client.lastSearch.NextPageToken)
}

func TestRHSHTTPGetIssuesNotConnectedJSON(t *testing.T) {
	api := &plugintest.API{}
	p := setupRHSHTTPPlugin(t, api)
	client := &testRHSCloudClient{
		statuses:   fixtureStatuses(t),
		categories: fixtureCanonicalCategories(t),
	}
	inst := installRHSUserCloud(t, p, client)
	p.userStore = rhsErrNotFoundUserStore{}

	w := doRHSHTTPGet(t, p, makeAPIRoute(routeAPIRHSIssues)+"?instance_id="+string(inst.GetID()), "connected_user")
	require.Equal(t, http.StatusUnauthorized, w.Result().StatusCode)
	assert.NotEqual(t, "text/plain", w.Result().Header.Get("Content-Type"))
	body := decodeRHSError(t, w)
	assert.Equal(t, rhsErrNotConnected, body.Error)
	assert.NotEmpty(t, body.Message)

	client.mu.Lock()
	defer client.mu.Unlock()
	assert.Equal(t, 0, client.searchCalls)
}

func TestRHSHTTPGetIssuesCloudOAuthMissingTokenJSON(t *testing.T) {
	api := &plugintest.API{}
	p := setupRHSHTTPPlugin(t, api)
	oi := installRHSCloudOAuth(t, p)
	p.userStore = rhsNilOAuthTokenUserStore{}

	w := doRHSHTTPGet(t, p, makeAPIRoute(routeAPIRHSIssues)+"?instance_id="+string(oi.GetID()), "connected_user")
	require.Equal(t, http.StatusUnauthorized, w.Result().StatusCode)
	require.NotEqual(t, http.StatusInternalServerError, w.Result().StatusCode)
	assert.NotEqual(t, "text/plain", w.Result().Header.Get("Content-Type"))
	body := decodeRHSError(t, w)
	assert.Equal(t, rhsErrNotConnected, body.Error)
	assert.NotEqual(t, rhsErrInternal, body.Error)
	assert.NotEmpty(t, body.Message)
}

func TestRHSHTTPGetIssuesRateLimitedJSON(t *testing.T) {
	api := &plugintest.API{}
	p := setupRHSHTTPPlugin(t, api)
	client := &testRHSCloudClient{
		statuses:   fixtureStatuses(t),
		categories: fixtureCanonicalCategories(t),
		searchErr:  ErrRateLimited,
	}
	inst := installRHSUserCloud(t, p, client)

	w := doRHSHTTPGet(t, p, makeAPIRoute(routeAPIRHSIssues)+"?instance_id="+string(inst.GetID()), "connected_user")
	require.Equal(t, http.StatusTooManyRequests, w.Result().StatusCode)
	body := decodeRHSError(t, w)
	assert.Equal(t, rhsErrRateLimited, body.Error)
	assert.Contains(t, body.Message, "rate limit")
}

func TestRHSHTTPGetIssuesNotCloudJSON(t *testing.T) {
	api := &plugintest.API{}
	p := setupRHSHTTPPlugin(t, api)
	p.userStore = mockUserStore{}

	w := doRHSHTTPGet(t, p, makeAPIRoute(routeAPIRHSIssues)+"?instance_id="+mockInstance1URL, "connected_user")
	require.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	body := decodeRHSError(t, w)
	assert.Equal(t, rhsErrNotCloud, body.Error)
}

func TestRHSHTTPGetIssuesUsesCachedCategoryKeys(t *testing.T) {
	api := &plugintest.API{}
	p := setupRHSHTTPPlugin(t, api)
	client := &testRHSCloudClient{
		statuses:   fixtureStatuses(t),
		categories: fixtureCategoriesOmitDone(),
	}
	inst := installRHSUserCloud(t, p, client)

	w := doRHSHTTPGet(t, p, makeAPIRoute(routeAPIRHSIssues)+"?instance_id="+string(inst.GetID()), "connected_user")
	require.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	body := decodeRHSError(t, w)
	assert.Equal(t, rhsErrInvalidRequest, body.Error)

	client.mu.Lock()
	defer client.mu.Unlock()
	assert.Equal(t, 0, client.searchCalls)
}

func TestRHSHTTPGetIssuesBrowseURLUsesJiraBaseURL(t *testing.T) {
	api := &plugintest.API{}
	p := setupRHSHTTPPlugin(t, api)
	client := &testRHSCloudClient{
		statuses:   fixtureStatuses(t),
		categories: fixtureCanonicalCategories(t),
		search: &CloudSearchResult{
			Issues: []jira.Issue{tes41Issue()},
			IsLast: true,
		},
	}
	inst := installRHSUserCloud(t, p, client)

	w := doRHSHTTPGet(t, p, makeAPIRoute(routeAPIRHSIssues)+"?instance_id="+string(inst.GetID()), "connected_user")
	require.Equal(t, http.StatusOK, w.Result().StatusCode)

	var body rhsIssuesResult
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Issues, 1)
	assert.Equal(t, "https://cloud.example.atlassian.net/browse/TES-41", body.Issues[0].BrowseURL)
	assert.NotContains(t, body.Issues[0].BrowseURL, "api.atlassian.com")
}

func TestRHSHTTPGetIssuesInvalidSortJSON(t *testing.T) {
	api := &plugintest.API{}
	p := setupRHSHTTPPlugin(t, api)
	client := &testRHSCloudClient{
		statuses:   fixtureStatuses(t),
		categories: fixtureCanonicalCategories(t),
	}
	inst := installRHSUserCloud(t, p, client)

	w := doRHSHTTPGet(t, p, makeAPIRoute(routeAPIRHSIssues)+"?instance_id="+string(inst.GetID())+"&sort=priority", "connected_user")
	require.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	body := decodeRHSError(t, w)
	assert.Equal(t, rhsErrInvalidRequest, body.Error)

	client.mu.Lock()
	defer client.mu.Unlock()
	assert.Equal(t, 0, client.searchCalls)
}

func TestRHSHTTPGetIssuesUnknownStatusTabFallsBackToAssigned(t *testing.T) {
	api := &plugintest.API{}
	p := setupRHSHTTPPlugin(t, api)
	client := &testRHSCloudClient{
		statuses:   fixtureStatuses(t),
		categories: fixtureCanonicalCategories(t),
		search: &CloudSearchResult{
			Issues: []jira.Issue{tes41Issue()},
			IsLast: true,
		},
	}
	inst := installRHSUserCloud(t, p, client)

	w := doRHSHTTPGet(t, p, makeAPIRoute(routeAPIRHSIssues)+"?instance_id="+string(inst.GetID())+"&tab=status:does-not-exist", "connected_user")
	require.Equal(t, http.StatusOK, w.Result().StatusCode)

	var body rhsIssuesResult
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Issues, 1)
	assert.Equal(t, "TES-41", body.Issues[0].Key)
	require.GreaterOrEqual(t, len(body.Tabs), 2)
	assert.Equal(t, RHSTabKindAssigned, body.Tabs[0].Kind)
	assert.Equal(t, rhsAssignedTabName, body.Tabs[0].Name)
	assert.Equal(t, RHSTabKindCategory, body.Tabs[1].Kind)
	assert.Equal(t, statusCategoryKeyIndeterminate, body.Tabs[1].Key)

	client.mu.Lock()
	defer client.mu.Unlock()
	assert.Equal(t, 1, client.searchCalls)
	assert.Equal(t, "assignee = currentUser() AND statusCategory != done ORDER BY updated DESC, key ASC", client.lastSearch.JQL)
}

func TestRHSHTTPGetIssuesCategoryTabIdentity(t *testing.T) {
	api := &plugintest.API{}
	p := setupRHSHTTPPlugin(t, api)
	client := &testRHSCloudClient{
		statuses:   fixtureStatuses(t),
		categories: fixtureCanonicalCategories(t),
		search:     &CloudSearchResult{IsLast: true},
	}
	inst := installRHSUserCloud(t, p, client)

	w := doRHSHTTPGet(t, p, makeAPIRoute(routeAPIRHSIssues)+"?instance_id="+string(inst.GetID())+"&tab=category:indeterminate", "connected_user")
	require.Equal(t, http.StatusOK, w.Result().StatusCode)

	client.mu.Lock()
	defer client.mu.Unlock()
	assert.Equal(t, "assignee = currentUser() AND statusCategory = indeterminate ORDER BY updated DESC, key ASC", client.lastSearch.JQL)
}

func TestRHSHTTPGetIssuesMissingInstanceIDJSON(t *testing.T) {
	api := &plugintest.API{}
	p := setupRHSHTTPPlugin(t, api)

	w := doRHSHTTPGet(t, p, makeAPIRoute(routeAPIRHSIssues), "connected_user")
	require.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	body := decodeRHSError(t, w)
	assert.Equal(t, rhsErrInvalidRequest, body.Error)
}

func TestRHSHTTPListStatusesNotAuthorizedJSON(t *testing.T) {
	api := &plugintest.API{}
	api.On("HasPermissionTo", mock.AnythingOfType("string"), mock.Anything).Return(false)
	p := setupRHSHTTPPlugin(t, api)

	w := doRHSHTTPGet(t, p, makeAPIRoute(routeAPIRHSStatuses)+"?instance_id="+mockInstance1URL, "connected_user")
	require.Equal(t, http.StatusForbidden, w.Result().StatusCode)
	body := decodeRHSError(t, w)
	assert.Equal(t, rhsErrNotAuthorized, body.Error)
}

func TestRHSHTTPListStatusesConnectJWTNoPersonalConnection(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/3/status":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(loadRHSTestdata(t, "rhs-status.json"))
		case "/rest/api/3/statuscategory":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(loadRHSTestdata(t, "rhs-statuscategory.json"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(ts.Close)

	api := &plugintest.API{}
	api.On("HasPermissionTo", mock.AnythingOfType("string"), mock.Anything).Return(true)
	p := setupRHSHTTPPlugin(t, api)
	p.userStore = rhsErrNotFoundUserStore{}

	ci := newCloudInstance(p, types.ID("https://connect.example.atlassian.net"), true, "", &AtlassianSecurityContext{
		Key:          "test-key",
		ClientKey:    "test-client-key",
		SharedSecret: "test-shared-secret",
		BaseURL:      ts.URL,
	})
	storeRHSInstance(t, p, ci)

	w := doRHSHTTPGet(t, p, makeAPIRoute(routeAPIRHSStatuses)+"?instance_id=https://connect.example.atlassian.net", "connected_user")
	require.Equal(t, http.StatusOK, w.Result().StatusCode)
	require.Equal(t, "application/json", w.Result().Header.Get("Content-Type"))

	var body rhsStatusesResult
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Statuses, 2)
	require.Len(t, body.Categories, 4)
	keys := make([]string, 0, len(body.Categories))
	for _, c := range body.Categories {
		require.NotNil(t, c)
		keys = append(keys, c.Key)
	}
	assert.Contains(t, keys, statusCategoryKeyDone)
	require.NotNil(t, body.Statuses[0])
	assert.Equal(t, "3", body.Statuses[0].ID)
	assert.Equal(t, statusCategoryKeyIndeterminate, body.Statuses[0].StatusCategory.Key)
}

func TestRHSHTTPListStatusesEnrichesScopedProject(t *testing.T) {
	var projectSearchCalls int
	var projectSearchIDs []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/3/status":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(loadRHSTestdata(t, "rhs-status-scoped.json"))
		case "/rest/api/3/statuscategory":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(loadRHSTestdata(t, "rhs-statuscategory.json"))
		case "/rest/api/3/project/search":
			projectSearchCalls++
			projectSearchIDs = r.URL.Query()["id"]
			writeJSON(w, http.StatusOK, []byte(`{
				"values": [{"id": "10000", "key": "PLAY", "name": "Playbooks"}],
				"isLast": true
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(ts.Close)

	api := &plugintest.API{}
	api.On("HasPermissionTo", mock.AnythingOfType("string"), mock.Anything).Return(true)
	p := setupRHSHTTPPlugin(t, api)
	p.userStore = rhsErrNotFoundUserStore{}

	ci := newCloudInstance(p, types.ID("https://connect.example.atlassian.net"), true, "", &AtlassianSecurityContext{
		Key:          "test-key",
		ClientKey:    "test-client-key",
		SharedSecret: "test-shared-secret",
		BaseURL:      ts.URL,
	})
	storeRHSInstance(t, p, ci)

	w := doRHSHTTPGet(t, p, makeAPIRoute(routeAPIRHSStatuses)+"?instance_id=https://connect.example.atlassian.net", "connected_user")
	require.Equal(t, http.StatusOK, w.Result().StatusCode)

	var body rhsStatusesResult
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Statuses, 1)
	require.NotNil(t, body.Statuses[0].Project)
	assert.Equal(t, "Playbooks", body.Statuses[0].Project.Name)
	assert.Equal(t, "PLAY", body.Statuses[0].Project.Key)
	assert.Equal(t, 1, projectSearchCalls)
	assert.Equal(t, []string{"10000"}, projectSearchIDs)
}

func TestRHSHTTPGetIssuesCachesStatusesPerUser(t *testing.T) {
	api := &plugintest.API{}
	p := setupRHSHTTPPlugin(t, api)
	client := &testRHSCloudClient{
		statuses:   fixtureStatuses(t),
		categories: fixtureCanonicalCategories(t),
		search:     &CloudSearchResult{IsLast: true},
	}
	inst := installRHSUserCloud(t, p, client)

	w := doRHSHTTPGet(t, p, makeAPIRoute(routeAPIRHSIssues)+"?instance_id="+string(inst.GetID()), "connected_user")
	require.Equal(t, http.StatusOK, w.Result().StatusCode)

	p.rhsStatusCacheLock.RLock()
	defer p.rhsStatusCacheLock.RUnlock()
	_, ok := p.rhsStatusCache[rhsStatusCacheKey{instanceID: inst.GetID(), userID: "connected_user"}]
	assert.True(t, ok)
	_, shared := p.rhsStatusCache[rhsStatusCacheKey{instanceID: inst.GetID()}]
	assert.False(t, shared)
	_, bot := p.rhsStatusCache[rhsStatusCacheKey{instanceID: inst.GetID(), userID: rhsStatusCacheBotUser}]
	assert.False(t, bot)
}

func TestRHSHTTPListStatusesCachesUnderBotUser(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/3/status":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(loadRHSTestdata(t, "rhs-status.json"))
		case "/rest/api/3/statuscategory":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(loadRHSTestdata(t, "rhs-statuscategory.json"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(ts.Close)

	api := &plugintest.API{}
	api.On("HasPermissionTo", mock.AnythingOfType("string"), mock.Anything).Return(true)
	p := setupRHSHTTPPlugin(t, api)
	p.userStore = rhsErrNotFoundUserStore{}

	ci := newCloudInstance(p, types.ID("https://connect.example.atlassian.net"), true, "", &AtlassianSecurityContext{
		Key:          "test-key",
		ClientKey:    "test-client-key",
		SharedSecret: "test-shared-secret",
		BaseURL:      ts.URL,
	})
	storeRHSInstance(t, p, ci)
	instanceID := ci.GetID()

	w := doRHSHTTPGet(t, p, makeAPIRoute(routeAPIRHSStatuses)+"?instance_id="+string(instanceID), "connected_user")
	require.Equal(t, http.StatusOK, w.Result().StatusCode)

	p.rhsStatusCacheLock.RLock()
	defer p.rhsStatusCacheLock.RUnlock()
	_, bot := p.rhsStatusCache[rhsStatusCacheKey{instanceID: instanceID, userID: rhsStatusCacheBotUser}]
	assert.True(t, bot)
	_, asAdmin := p.rhsStatusCache[rhsStatusCacheKey{instanceID: instanceID, userID: "connected_user"}]
	assert.False(t, asAdmin)
}

func TestRHSHTTPGetIssuesDoesNotSearchProjects(t *testing.T) {
	var projectSearchCalls int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/3/status":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(loadRHSTestdata(t, "rhs-status-scoped.json"))
		case "/rest/api/3/statuscategory":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(loadRHSTestdata(t, "rhs-statuscategory.json"))
		case "/rest/api/3/search/jql":
			writeJSON(w, http.StatusOK, loadRHSTestdata(t, "rhs-search-jql-page1.json"))
		case "/rest/api/3/project/search":
			projectSearchCalls++
			t.Errorf("GET /rhs/issues must not call /project/search")
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(ts.Close)

	jc, err := jira.NewClient(ts.Client(), ts.URL)
	require.NoError(t, err)
	api := &plugintest.API{}
	p := setupRHSHTTPPlugin(t, api)
	inst := rhsTestCloudInstance{
		testInstance: testInstance{
			InstanceCommon: InstanceCommon{
				InstanceID: types.ID("https://cloud.example.atlassian.net"),
				Type:       CloudInstanceType,
				Plugin:     p,
			},
		},
		client:      newCloudClient(jc),
		jiraBaseURL: "https://cloud.example.atlassian.net",
		apiURL:      ts.URL,
	}
	storeRHSInstance(t, p, inst)
	p.userStore = mockUserStore{}

	w := doRHSHTTPGet(t, p, makeAPIRoute(routeAPIRHSIssues)+"?instance_id="+string(inst.GetID()), "connected_user")
	require.Equal(t, http.StatusOK, w.Result().StatusCode)
	assert.Equal(t, 0, projectSearchCalls)
}

func TestRHSHTTPListStatusesOAuth2NotConnectedJSON(t *testing.T) {
	api := &plugintest.API{}
	api.On("HasPermissionTo", mock.AnythingOfType("string"), mock.Anything).Return(true)
	p := setupRHSHTTPPlugin(t, api)
	p.userStore = rhsErrNotFoundUserStore{}

	oi := &cloudOAuthInstance{
		InstanceCommon: newInstanceCommon(p, CloudOAuthInstanceType, types.ID("https://oauth.example.atlassian.net")),
		JiraBaseURL:    "https://oauth.example.atlassian.net",
	}
	storeRHSInstance(t, p, oi)

	w := doRHSHTTPGet(t, p, makeAPIRoute(routeAPIRHSStatuses)+"?instance_id=https://oauth.example.atlassian.net", "connected_user")
	require.Equal(t, http.StatusUnauthorized, w.Result().StatusCode)
	body := decodeRHSError(t, w)
	assert.Equal(t, rhsErrNotConnected, body.Error)
}

func TestRHSHTTPListStatusesCloudOAuthMissingTokenJSON(t *testing.T) {
	api := &plugintest.API{}
	api.On("HasPermissionTo", mock.AnythingOfType("string"), mock.Anything).Return(true)
	p := setupRHSHTTPPlugin(t, api)
	oi := installRHSCloudOAuth(t, p)
	p.userStore = rhsNilOAuthTokenUserStore{}

	w := doRHSHTTPGet(t, p, makeAPIRoute(routeAPIRHSStatuses)+"?instance_id="+string(oi.GetID()), "connected_user")
	require.Equal(t, http.StatusUnauthorized, w.Result().StatusCode)
	require.NotEqual(t, http.StatusInternalServerError, w.Result().StatusCode)
	body := decodeRHSError(t, w)
	assert.Equal(t, rhsErrNotConnected, body.Error)
	assert.NotEqual(t, rhsErrInternal, body.Error)
	assert.NotEmpty(t, body.Message)
}

func TestRHSHTTPListStatusesOAuth2IgnoresAdminAPIToken(t *testing.T) {
	api := &plugintest.API{}
	api.On("HasPermissionTo", mock.AnythingOfType("string"), mock.Anything).Return(true)
	p := setupRHSHTTPPlugin(t, api)
	p.userStore = rhsErrNotFoundUserStore{}
	p.updateConfig(func(conf *config) {
		conf.AdminEmail = "admin@example.com"
		conf.AdminAPIToken = "not-a-real-token"
		conf.maxAttachmentSize = defaultMaxAttachmentSize
	})

	oi := &cloudOAuthInstance{
		InstanceCommon: newInstanceCommon(p, CloudOAuthInstanceType, types.ID("https://oauth.example.atlassian.net")),
		JiraBaseURL:    "https://oauth.example.atlassian.net",
	}
	storeRHSInstance(t, p, oi)

	w := doRHSHTTPGet(t, p, makeAPIRoute(routeAPIRHSStatuses)+"?instance_id=https://oauth.example.atlassian.net", "connected_user")
	require.Equal(t, http.StatusUnauthorized, w.Result().StatusCode)
	body := decodeRHSError(t, w)
	assert.Equal(t, rhsErrNotConnected, body.Error)
}

func TestRHSHTTPMissingUserStillPlainText(t *testing.T) {
	api := &plugintest.API{}
	p := setupRHSHTTPPlugin(t, api)

	issues := doRHSHTTPGet(t, p, makeAPIRoute(routeAPIRHSIssues), "")
	require.Equal(t, http.StatusUnauthorized, issues.Result().StatusCode)
	assert.Contains(t, issues.Body.String(), "Not authorized")
	assert.NotEqual(t, "application/json", issues.Result().Header.Get("Content-Type"))

	statuses := doRHSHTTPGet(t, p, makeAPIRoute(routeAPIRHSStatuses), "")
	require.Equal(t, http.StatusUnauthorized, statuses.Result().StatusCode)
	assert.Contains(t, statuses.Body.String(), "Not authorized")
	assert.NotEqual(t, "application/json", statuses.Result().Header.Get("Content-Type"))
}
