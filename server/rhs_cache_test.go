// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

type countingRHSStatusClient struct {
	mu            sync.Mutex
	statusCalls   int
	catCalls      int
	sleep         time.Duration
	statuses      []*JiraStatus
	categories    []*JiraStatusCategory
	statusErr     error
	catErr        error
	statusStarted chan struct{}
	catStarted    chan struct{}
	block         chan struct{}
}

func (c *countingRHSStatusClient) ListStatuses() ([]*JiraStatus, error) {
	c.mu.Lock()
	c.statusCalls++
	c.mu.Unlock()
	c.signalStart(c.statusStarted)
	c.waitBlock()
	if c.sleep > 0 {
		time.Sleep(c.sleep)
	}
	if c.statusErr != nil {
		return nil, c.statusErr
	}
	return c.statuses, nil
}

func (c *countingRHSStatusClient) ListStatusCategories() ([]*JiraStatusCategory, error) {
	c.mu.Lock()
	c.catCalls++
	c.mu.Unlock()
	c.signalStart(c.catStarted)
	c.waitBlock()
	if c.catErr != nil {
		return nil, c.catErr
	}
	return c.categories, nil
}

func (c *countingRHSStatusClient) signalStart(ch chan struct{}) {
	if ch == nil {
		return
	}
	select {
	case <-ch:
	default:
		close(ch)
	}
}

func (c *countingRHSStatusClient) waitBlock() {
	if c.block == nil {
		return
	}
	<-c.block
}

func (c *countingRHSStatusClient) counts() (int, int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.statusCalls, c.catCalls
}

func fixtureStatusesAndCategories(t *testing.T) ([]*JiraStatus, []*JiraStatusCategory) {
	t.Helper()
	var statuses []*JiraStatus
	require.NoError(t, json.Unmarshal(loadRHSTestdata(t, "rhs-status.json"), &statuses))
	var categories []*JiraStatusCategory
	require.NoError(t, json.Unmarshal(loadRHSTestdata(t, "rhs-statuscategory.json"), &categories))
	return statuses, categories
}

const testRHSCacheUserID types.ID = "rhs-cache-user"

func testRHSKey(instanceID types.ID) rhsStatusCacheKey {
	return rhsStatusCacheKey{instanceID: instanceID, userID: testRHSCacheUserID}
}

func getCached(p *Plugin, instanceID types.ID, client rhsStatusLister) (*rhsStatusCacheEntry, error) {
	return p.getInstanceStatuses(instanceID, testRHSCacheUserID, client)
}

func TestRHSCacheTTLIsOneHour(t *testing.T) {
	assert.Equal(t, time.Hour, rhsStatusCacheTTL)
}

func TestRHSCacheFreshHitDoesNotRefetch(t *testing.T) {
	api := &plugintest.API{}
	p := setupTestPlugin(api)
	statuses, categories := fixtureStatusesAndCategories(t)
	client := &countingRHSStatusClient{statuses: statuses, categories: categories}
	id := types.ID("https://a.example.atlassian.net")

	got, err := getCached(p, id, client)
	require.NoError(t, err)
	require.NotNil(t, got)
	statusCalls, catCalls := client.counts()
	assert.Equal(t, 1, statusCalls)
	assert.Equal(t, 1, catCalls)
	require.Len(t, got.statuses, 2)
	require.Len(t, got.categories, 4)
	assert.Equal(t, "3", got.statuses[0].ID)
	keys := make([]string, 0, len(got.categories))
	for _, c := range got.categories {
		keys = append(keys, c.Key)
	}
	assert.Contains(t, keys, statusCategoryKeyDone)

	got2, err := getCached(p, id, client)
	require.NoError(t, err)
	require.NotNil(t, got2)
	statusCalls, catCalls = client.counts()
	assert.Equal(t, 1, statusCalls)
	assert.Equal(t, 1, catCalls)
	require.Len(t, got2.statuses, 2)
	require.Len(t, got2.categories, 4)
}

func TestRHSCacheExpiredEntryRefetches(t *testing.T) {
	api := &plugintest.API{}
	p := setupTestPlugin(api)
	statuses, categories := fixtureStatusesAndCategories(t)
	client := &countingRHSStatusClient{statuses: statuses, categories: categories}
	id := types.ID("https://a.example.atlassian.net")

	p.rhsStatusCacheLock.Lock()
	p.rhsStatusCache = map[rhsStatusCacheKey]*rhsStatusCacheEntry{
		testRHSKey(id): {
			statuses:   statuses,
			categories: categories,
			fetchedAt:  time.Now().Add(-2 * time.Hour),
		},
	}
	p.rhsStatusCacheLock.Unlock()

	got, err := getCached(p, id, client)
	require.NoError(t, err)
	require.NotNil(t, got)
	statusCalls, catCalls := client.counts()
	assert.Equal(t, 1, statusCalls)
	assert.Equal(t, 1, catCalls)

	p.rhsStatusCacheLock.RLock()
	entry := p.rhsStatusCache[testRHSKey(id)]
	p.rhsStatusCacheLock.RUnlock()
	require.NotNil(t, entry)
	assert.Less(t, time.Since(entry.fetchedAt), time.Minute)
}

func TestRHSCacheInvalidateEmptiesMap(t *testing.T) {
	api := &plugintest.API{}
	p := setupTestPlugin(api)
	statuses, categories := fixtureStatusesAndCategories(t)
	idA := types.ID("https://a.example.atlassian.net")
	idB := types.ID("https://b.example.atlassian.net")

	p.rhsStatusCacheLock.Lock()
	p.rhsStatusCache = map[rhsStatusCacheKey]*rhsStatusCacheEntry{
		testRHSKey(idA): {fetchedAt: time.Now()},
		testRHSKey(idB): {fetchedAt: time.Now()},
	}
	p.rhsStatusCacheLock.Unlock()

	p.invalidateRHSStatusCache()
	assert.Empty(t, p.rhsStatusCache)

	client := &countingRHSStatusClient{statuses: statuses, categories: categories}
	got, err := getCached(p, idA, client)
	require.NoError(t, err)
	require.NotNil(t, got)
	statusCalls, catCalls := client.counts()
	assert.Equal(t, 1, statusCalls)
	assert.Equal(t, 1, catCalls)
}

func TestRHSCacheConcurrentWarmDoesNotDoubleFetch(t *testing.T) {
	api := &plugintest.API{}
	p := setupTestPlugin(api)
	statuses, categories := fixtureStatusesAndCategories(t)
	client := &countingRHSStatusClient{
		statuses:   statuses,
		categories: categories,
		sleep:      50 * time.Millisecond,
	}
	id := types.ID("https://a.example.atlassian.net")

	const waiters = 8
	var wg sync.WaitGroup
	wg.Add(waiters)
	gots := make([]*rhsStatusCacheEntry, waiters)
	errs := make([]error, waiters)
	for i := 0; i < waiters; i++ {
		go func(i int) {
			defer wg.Done()
			gots[i], errs[i] = getCached(p, id, client)
		}(i)
	}
	wg.Wait()

	for i := 0; i < waiters; i++ {
		require.NoError(t, errs[i])
		require.NotNil(t, gots[i])
		require.Len(t, gots[i].statuses, 2)
	}
	statusCalls, catCalls := client.counts()
	assert.Equal(t, 1, statusCalls)
	assert.Equal(t, 1, catCalls)
}

func TestRHSCacheSlowMissDoesNotBlockOtherInstanceHit(t *testing.T) {
	api := &plugintest.API{}
	p := setupTestPlugin(api)
	statuses, categories := fixtureStatusesAndCategories(t)
	idA := types.ID("https://a.example.atlassian.net")
	idB := types.ID("https://b.example.atlassian.net")

	p.rhsStatusCacheLock.Lock()
	p.rhsStatusCache = map[rhsStatusCacheKey]*rhsStatusCacheEntry{
		testRHSKey(idB): {
			statuses:   statuses,
			categories: categories,
			fetchedAt:  time.Now(),
		},
	}
	p.rhsStatusCacheLock.Unlock()

	slowA := &countingRHSStatusClient{
		statuses:   statuses,
		categories: categories,
		sleep:      250 * time.Millisecond,
	}
	clientB := &countingRHSStatusClient{statuses: statuses, categories: categories}

	aDone := make(chan struct{})
	go func() {
		defer close(aDone)
		_, _ = getCached(p, idA, slowA)
	}()

	waitUntil := time.Now().Add(time.Second)
	for {
		n, _ := slowA.counts()
		if n >= 1 {
			break
		}
		if time.Now().After(waitUntil) {
			t.Fatal("timed out waiting for instance A ListStatuses")
		}
		time.Sleep(5 * time.Millisecond)
	}

	gotB, err := getCached(p, idB, clientB)

	select {
	case <-aDone:
		t.Fatal("instance A fetch finished before instance B cache hit returned")
	default:
	}

	require.NoError(t, err)
	require.NotNil(t, gotB)
	statusB, catB := clientB.counts()
	assert.Equal(t, 0, statusB)
	assert.Equal(t, 0, catB)
	require.Len(t, gotB.statuses, 2)

	<-aDone
}

func TestRHSCacheFetchErrorDoesNotStore(t *testing.T) {
	api := &plugintest.API{}
	p := setupTestPlugin(api)
	statuses, categories := fixtureStatusesAndCategories(t)
	id := types.ID("https://a.example.atlassian.net")

	t.Run("status error", func(t *testing.T) {
		statusErr := errors.New("status boom")
		client := &countingRHSStatusClient{statusErr: statusErr}
		got, err := getCached(p, id, client)
		assert.ErrorIs(t, err, statusErr)
		assert.Nil(t, got)
		assert.Empty(t, p.rhsStatusCache)
		assert.Empty(t, p.rhsStatusFlights)
	})

	t.Run("category error", func(t *testing.T) {
		catErr := errors.New("category boom")
		client := &countingRHSStatusClient{statuses: statuses, catErr: catErr}
		got, err := getCached(p, id, client)
		assert.ErrorIs(t, err, catErr)
		assert.Nil(t, got)
		assert.Empty(t, p.rhsStatusCache)
		assert.Empty(t, p.rhsStatusFlights)
	})

	healthy := &countingRHSStatusClient{statuses: statuses, categories: categories}
	got, err := getCached(p, id, healthy)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Len(t, p.rhsStatusCache, 1)
}

func TestRHSCacheOnConfigurationChangeEmptiesAllInstances(t *testing.T) {
	api := &plugintest.API{}
	p := setupTestPlugin(api)

	maxFileSize := int64(100 * 1024 * 1024)
	api.On("GetConfig").Return(&model.Config{
		FileSettings: model.FileSettings{MaxFileSize: &maxFileSize},
	})
	api.On("LoadPluginConfiguration", mock.Anything).Return(nil)
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe().Return()

	idA := types.ID("https://a.example.atlassian.net")
	idB := types.ID("https://b.example.atlassian.net")
	p.rhsStatusCacheLock.Lock()
	p.rhsStatusCache = map[rhsStatusCacheKey]*rhsStatusCacheEntry{
		testRHSKey(idA): {fetchedAt: time.Now()},
		testRHSKey(idB): {fetchedAt: time.Now()},
	}
	p.rhsStatusCacheLock.Unlock()

	require.NoError(t, p.OnConfigurationChange())
	assert.Empty(t, p.rhsStatusCache)
}

type lookupRHSStatusClient struct {
	countingRHSStatusClient
	projects    map[string]JiraStatusProject
	lookupErr   error
	lookupCalls int
}

func (c *lookupRHSStatusClient) lookupStatusProjects(ids []string) (map[string]JiraStatusProject, error) {
	c.mu.Lock()
	c.lookupCalls++
	c.mu.Unlock()
	if c.lookupErr != nil {
		return nil, c.lookupErr
	}
	return c.projects, nil
}

func TestRHSCacheDoesNotLookupProjectsOnGetInstanceStatuses(t *testing.T) {
	api := &plugintest.API{}
	p := setupTestPlugin(api)
	statuses := []*JiraStatus{
		{ID: "3", Name: "In Progress"},
		{
			ID:   "10042",
			Name: "In Progress",
			Scope: &JiraStatusScope{
				Type:    "PROJECT",
				Project: &JiraStatusScopeProject{ID: "10000"},
			},
		},
	}
	categories := []*JiraStatusCategory{{ID: 4, Key: statusCategoryKeyIndeterminate, Name: "In Progress"}}
	client := &lookupRHSStatusClient{
		countingRHSStatusClient: countingRHSStatusClient{statuses: statuses, categories: categories},
		projects: map[string]JiraStatusProject{
			"10000": {ID: "10000", Key: "PLAY", Name: "Playbooks"},
		},
	}

	got, err := getCached(p, types.ID("https://a.example.atlassian.net"), client)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, 0, client.lookupCalls)
	assert.Nil(t, got.statuses[0].Project)
	assert.Nil(t, got.statuses[1].Project)
}

func TestRHSCacheEnrichesScopedStatusesWithProjectNames(t *testing.T) {
	api := &plugintest.API{}
	p := setupTestPlugin(api)
	statuses := []*JiraStatus{
		{ID: "3", Name: "In Progress"},
		{
			ID:   "10042",
			Name: "In Progress",
			Scope: &JiraStatusScope{
				Type:    "PROJECT",
				Project: &JiraStatusScopeProject{ID: "10000"},
			},
		},
	}
	client := &lookupRHSStatusClient{
		projects: map[string]JiraStatusProject{
			"10000": {ID: "10000", Key: "PLAY", Name: "Playbooks"},
		},
	}

	p.enrichStatusesWithProjects(client, statuses)
	assert.Equal(t, 1, client.lookupCalls)
	assert.Nil(t, statuses[0].Project)
	require.NotNil(t, statuses[1].Project)
	assert.Equal(t, "Playbooks", statuses[1].Project.Name)
	assert.Equal(t, "PLAY", statuses[1].Project.Key)
}

func TestRHSCacheSkipsProjectLookupWhenStatusesAreGlobal(t *testing.T) {
	api := &plugintest.API{}
	p := setupTestPlugin(api)
	statuses, _ := fixtureStatusesAndCategories(t)
	client := &lookupRHSStatusClient{
		projects: map[string]JiraStatusProject{"10000": {Name: "unused"}},
	}

	p.enrichStatusesWithProjects(client, statuses)
	assert.Equal(t, 0, client.lookupCalls)
}

func TestRHSCacheProjectLookupErrorStillReturnsStatuses(t *testing.T) {
	api := &plugintest.API{}
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe().Return()
	p := setupTestPlugin(api)
	statuses := []*JiraStatus{
		{
			ID:   "10042",
			Name: "In Progress",
			Scope: &JiraStatusScope{
				Type:    "PROJECT",
				Project: &JiraStatusScopeProject{ID: "10000"},
			},
		},
	}
	client := &lookupRHSStatusClient{
		lookupErr: errors.New("project search failed"),
	}

	p.enrichStatusesWithProjects(client, statuses)
	require.Len(t, statuses, 1)
	assert.Nil(t, statuses[0].Project)
	assert.Equal(t, 1, client.lookupCalls)
}

func TestRHSAdminStatusCacheUserID(t *testing.T) {
	admin := types.ID("admin-user")
	assert.Equal(t, rhsStatusCacheBotUser, rhsAdminStatusCacheUserID(&cloudInstance{}, admin))
	assert.Equal(t, admin, rhsAdminStatusCacheUserID(&cloudOAuthInstance{}, admin))
	assert.Equal(t, admin, rhsAdminStatusCacheUserID(nil, admin))
}

func TestRHSCacheUsersDoNotShareStatuses(t *testing.T) {
	api := &plugintest.API{}
	p := setupTestPlugin(api)
	id := types.ID("https://a.example.atlassian.net")
	_, categories := fixtureStatusesAndCategories(t)
	scoped := &JiraStatus{
		ID:   "10042",
		Name: "In Progress",
		Scope: &JiraStatusScope{
			Type:    "PROJECT",
			Project: &JiraStatusScopeProject{ID: "10000"},
		},
	}
	global := []*JiraStatus{{ID: "3", Name: "In Progress"}}
	userA := types.ID("user-a")
	userB := types.ID("user-b")
	clientA := &countingRHSStatusClient{statuses: []*JiraStatus{scoped}, categories: categories}
	clientB := &countingRHSStatusClient{statuses: global, categories: categories}

	gotA, err := p.getInstanceStatuses(id, userA, clientA)
	require.NoError(t, err)
	gotB, err := p.getInstanceStatuses(id, userB, clientB)
	require.NoError(t, err)

	configured := []RHSTabEntry{{Kind: RHSTabKindStatus, ID: "10042", Name: "Playbooks In Progress"}}
	tabsA := resolveTabs(configured, gotA.statuses, gotA.categories)
	tabsB := resolveTabs(configured, gotB.statuses, gotB.categories)
	require.Len(t, tabsA, 2)
	assert.Equal(t, "10042", tabsA[1].ID)
	require.Len(t, tabsB, 1)
	assert.Equal(t, RHSTabKindAssigned, tabsB[0].Kind)

	gotB2, err := p.getInstanceStatuses(id, userB, clientB)
	require.NoError(t, err)
	tabsB2 := resolveTabs(configured, gotB2.statuses, gotB2.categories)
	require.Len(t, tabsB2, 1)

	statusA, _ := clientA.counts()
	statusB, _ := clientB.counts()
	assert.Equal(t, 1, statusA)
	assert.Equal(t, 1, statusB)
	assert.Len(t, p.rhsStatusCache, 2)
}

func TestRHSCacheBotKeyDoesNotCollideWithUser(t *testing.T) {
	api := &plugintest.API{}
	p := setupTestPlugin(api)
	id := types.ID("https://a.example.atlassian.net")
	_, categories := fixtureStatusesAndCategories(t)
	botStatuses := []*JiraStatus{{ID: "10042", Name: "Team In Progress"}}
	userStatuses := []*JiraStatus{{ID: "3", Name: "In Progress"}}
	botClient := &countingRHSStatusClient{statuses: botStatuses, categories: categories}
	userClient := &countingRHSStatusClient{statuses: userStatuses, categories: categories}

	botEntry, err := p.getInstanceStatuses(id, rhsStatusCacheBotUser, botClient)
	require.NoError(t, err)
	userEntry, err := p.getInstanceStatuses(id, testRHSCacheUserID, userClient)
	require.NoError(t, err)

	require.Len(t, botEntry.statuses, 1)
	assert.Equal(t, "10042", botEntry.statuses[0].ID)
	require.Len(t, userEntry.statuses, 1)
	assert.Equal(t, "3", userEntry.statuses[0].ID)

	botAgain, err := p.getInstanceStatuses(id, rhsStatusCacheBotUser, botClient)
	require.NoError(t, err)
	assert.Equal(t, "10042", botAgain.statuses[0].ID)
	statusCalls, _ := botClient.counts()
	assert.Equal(t, 1, statusCalls)
}

func TestRHSCacheSlowMissDoesNotBlockOtherUserHit(t *testing.T) {
	api := &plugintest.API{}
	p := setupTestPlugin(api)
	statuses, categories := fixtureStatusesAndCategories(t)
	id := types.ID("https://a.example.atlassian.net")
	userA := types.ID("user-a")
	userB := types.ID("user-b")

	p.rhsStatusCacheLock.Lock()
	p.rhsStatusCache = map[rhsStatusCacheKey]*rhsStatusCacheEntry{
		{instanceID: id, userID: userB}: {
			statuses:   statuses,
			categories: categories,
			fetchedAt:  time.Now(),
		},
	}
	p.rhsStatusCacheLock.Unlock()

	slowA := &countingRHSStatusClient{
		statuses:   statuses,
		categories: categories,
		sleep:      250 * time.Millisecond,
	}
	clientB := &countingRHSStatusClient{statuses: statuses, categories: categories}

	aDone := make(chan struct{})
	go func() {
		defer close(aDone)
		_, _ = p.getInstanceStatuses(id, userA, slowA)
	}()

	waitUntil := time.Now().Add(time.Second)
	for {
		n, _ := slowA.counts()
		if n >= 1 {
			break
		}
		if time.Now().After(waitUntil) {
			t.Fatal("timed out waiting for user A ListStatuses")
		}
		time.Sleep(5 * time.Millisecond)
	}

	gotB, err := p.getInstanceStatuses(id, userB, clientB)

	select {
	case <-aDone:
		t.Fatal("user A fetch finished before user B cache hit returned")
	default:
	}

	require.NoError(t, err)
	require.NotNil(t, gotB)
	statusB, catB := clientB.counts()
	assert.Equal(t, 0, statusB)
	assert.Equal(t, 0, catB)

	<-aDone
}

func TestRHSCacheConcurrentWarmSharesFetchError(t *testing.T) {
	api := &plugintest.API{}
	p := setupTestPlugin(api)
	statusErr := errors.New("status boom")
	client := &countingRHSStatusClient{
		statusErr: statusErr,
		sleep:     50 * time.Millisecond,
	}
	id := types.ID("https://a.example.atlassian.net")

	const waiters = 8
	var wg sync.WaitGroup
	wg.Add(waiters)
	errs := make([]error, waiters)
	for i := 0; i < waiters; i++ {
		go func(i int) {
			defer wg.Done()
			_, errs[i] = getCached(p, id, client)
		}(i)
	}
	wg.Wait()

	for i := 0; i < waiters; i++ {
		assert.ErrorIs(t, errs[i], statusErr)
	}
	statusCalls, _ := client.counts()
	assert.Equal(t, 1, statusCalls)
	assert.Empty(t, p.rhsStatusCache)
	assert.Empty(t, p.rhsStatusFlights)
}

func TestRHSCacheConcurrentWarmDifferentUsersEachFetch(t *testing.T) {
	api := &plugintest.API{}
	p := setupTestPlugin(api)
	statuses, categories := fixtureStatusesAndCategories(t)
	id := types.ID("https://a.example.atlassian.net")
	clientA := &countingRHSStatusClient{
		statuses:   statuses,
		categories: categories,
		sleep:      50 * time.Millisecond,
	}
	clientB := &countingRHSStatusClient{
		statuses:   statuses,
		categories: categories,
		sleep:      50 * time.Millisecond,
	}

	var wg sync.WaitGroup
	wg.Add(2)
	var errA, errB error
	go func() {
		defer wg.Done()
		_, errA = p.getInstanceStatuses(id, "user-a", clientA)
	}()
	go func() {
		defer wg.Done()
		_, errB = p.getInstanceStatuses(id, "user-b", clientB)
	}()
	wg.Wait()

	require.NoError(t, errA)
	require.NoError(t, errB)
	statusA, _ := clientA.counts()
	statusB, _ := clientB.counts()
	assert.Equal(t, 1, statusA)
	assert.Equal(t, 1, statusB)
	assert.Len(t, p.rhsStatusCache, 2)
}

func TestRHSCacheFetchesStatusesAndCategoriesInParallel(t *testing.T) {
	api := &plugintest.API{}
	p := setupTestPlugin(api)
	statuses, categories := fixtureStatusesAndCategories(t)
	statusStarted := make(chan struct{})
	catStarted := make(chan struct{})
	block := make(chan struct{})
	t.Cleanup(func() {
		select {
		case <-block:
		default:
			close(block)
		}
	})
	client := &countingRHSStatusClient{
		statuses:      statuses,
		categories:    categories,
		statusStarted: statusStarted,
		catStarted:    catStarted,
		block:         block,
	}

	done := make(chan struct{})
	var got *rhsStatusCacheEntry
	var err error
	go func() {
		defer close(done)
		got, err = p.fetchRHSStatuses(client)
	}()

	wait := func(ch chan struct{}, name string) {
		t.Helper()
		select {
		case <-ch:
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for %s to start; fetches were not in flight together", name)
		}
	}
	wait(statusStarted, "ListStatuses")
	wait(catStarted, "ListStatusCategories")
	close(block)
	<-done

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Len(t, got.statuses, 2)
	require.Len(t, got.categories, 4)
	statusCalls, catCalls := client.counts()
	assert.Equal(t, 1, statusCalls)
	assert.Equal(t, 1, catCalls)
}

func TestRHSCacheEnrichDoesNotMutateCachedStatuses(t *testing.T) {
	api := &plugintest.API{}
	p := setupTestPlugin(api)
	statuses := []*JiraStatus{
		{
			ID:   "10042",
			Name: "In Progress",
			Scope: &JiraStatusScope{
				Type:    "PROJECT",
				Project: &JiraStatusScopeProject{ID: "10000"},
			},
		},
	}
	categories := []*JiraStatusCategory{{ID: 4, Key: statusCategoryKeyIndeterminate, Name: "In Progress"}}
	client := &lookupRHSStatusClient{
		countingRHSStatusClient: countingRHSStatusClient{statuses: statuses, categories: categories},
		projects: map[string]JiraStatusProject{
			"10000": {ID: "10000", Key: "PLAY", Name: "Playbooks"},
		},
	}
	id := types.ID("https://a.example.atlassian.net")

	adminCopy, err := getCached(p, id, client)
	require.NoError(t, err)
	require.Len(t, adminCopy.statuses, 1)
	assert.Nil(t, adminCopy.statuses[0].Project)

	p.enrichStatusesWithProjects(client, adminCopy.statuses)
	require.NotNil(t, adminCopy.statuses[0].Project)
	assert.Equal(t, "Playbooks", adminCopy.statuses[0].Project.Name)

	userCopy, err := getCached(p, id, client)
	require.NoError(t, err)
	require.Len(t, userCopy.statuses, 1)
	assert.Nil(t, userCopy.statuses[0].Project, "later fetch must not see Project written by admin enrichment")

	p.rhsStatusCacheLock.RLock()
	cached := p.rhsStatusCache[testRHSKey(id)]
	p.rhsStatusCacheLock.RUnlock()
	require.NotNil(t, cached)
	require.Len(t, cached.statuses, 1)
	assert.Nil(t, cached.statuses[0].Project)

	statusCalls, _ := client.counts()
	assert.Equal(t, 1, statusCalls)
}
