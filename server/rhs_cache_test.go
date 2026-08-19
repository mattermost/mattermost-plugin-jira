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
	mu          sync.Mutex
	statusCalls int
	catCalls    int
	sleep       time.Duration
	statuses    []*JiraStatus
	categories  []*JiraStatusCategory
	statusErr   error
	catErr      error
}

func (c *countingRHSStatusClient) ListStatuses() ([]*JiraStatus, error) {
	c.mu.Lock()
	c.statusCalls++
	c.mu.Unlock()
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
	if c.catErr != nil {
		return nil, c.catErr
	}
	return c.categories, nil
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

func TestRHSCacheTTLIsOneHour(t *testing.T) {
	assert.Equal(t, time.Hour, rhsStatusCacheTTL)
}

func TestRHSCacheFreshHitDoesNotRefetch(t *testing.T) {
	api := &plugintest.API{}
	p := setupTestPlugin(api)
	statuses, categories := fixtureStatusesAndCategories(t)
	client := &countingRHSStatusClient{statuses: statuses, categories: categories}
	id := types.ID("https://a.example.atlassian.net")

	got, err := p.getInstanceStatuses(id, client)
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

	got2, err := p.getInstanceStatuses(id, client)
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
	p.rhsStatusCache = map[types.ID]*rhsStatusCacheEntry{
		id: {
			statuses:   statuses,
			categories: categories,
			fetchedAt:  time.Now().Add(-2 * time.Hour),
		},
	}
	p.rhsStatusCacheLock.Unlock()

	got, err := p.getInstanceStatuses(id, client)
	require.NoError(t, err)
	require.NotNil(t, got)
	statusCalls, catCalls := client.counts()
	assert.Equal(t, 1, statusCalls)
	assert.Equal(t, 1, catCalls)

	p.rhsStatusCacheLock.RLock()
	entry := p.rhsStatusCache[id]
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
	p.rhsStatusCache = map[types.ID]*rhsStatusCacheEntry{
		idA: {fetchedAt: time.Now()},
		idB: {fetchedAt: time.Now()},
	}
	p.rhsStatusCacheLock.Unlock()

	p.invalidateRHSStatusCache()
	assert.Empty(t, p.rhsStatusCache)

	client := &countingRHSStatusClient{statuses: statuses, categories: categories}
	got, err := p.getInstanceStatuses(idA, client)
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

	var wg sync.WaitGroup
	wg.Add(2)
	var gotA, gotB *rhsStatusCacheEntry
	var errA, errB error
	go func() {
		defer wg.Done()
		gotA, errA = p.getInstanceStatuses(id, client)
	}()
	go func() {
		defer wg.Done()
		gotB, errB = p.getInstanceStatuses(id, client)
	}()
	wg.Wait()

	require.NoError(t, errA)
	require.NoError(t, errB)
	require.NotNil(t, gotA)
	require.NotNil(t, gotB)
	statusCalls, catCalls := client.counts()
	assert.Equal(t, 1, statusCalls)
	assert.Equal(t, 1, catCalls)
}

func TestRHSCacheFetchErrorDoesNotStore(t *testing.T) {
	api := &plugintest.API{}
	p := setupTestPlugin(api)
	statuses, categories := fixtureStatusesAndCategories(t)
	id := types.ID("https://a.example.atlassian.net")

	t.Run("status error", func(t *testing.T) {
		statusErr := errors.New("status boom")
		client := &countingRHSStatusClient{statusErr: statusErr}
		got, err := p.getInstanceStatuses(id, client)
		assert.ErrorIs(t, err, statusErr)
		assert.Nil(t, got)
		assert.Empty(t, p.rhsStatusCache)
	})

	t.Run("category error", func(t *testing.T) {
		catErr := errors.New("category boom")
		client := &countingRHSStatusClient{statuses: statuses, catErr: catErr}
		got, err := p.getInstanceStatuses(id, client)
		assert.ErrorIs(t, err, catErr)
		assert.Nil(t, got)
		assert.Empty(t, p.rhsStatusCache)
	})

	healthy := &countingRHSStatusClient{statuses: statuses, categories: categories}
	got, err := p.getInstanceStatuses(id, healthy)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Len(t, p.rhsStatusCache, 1)
}

func TestRHSCacheOnConfigurationChangeEmptiesAllInstances(t *testing.T) {
	api := &plugintest.API{}
	p := setupTestPlugin(api)

	max := int64(100 * 1024 * 1024)
	api.On("GetConfig").Return(&model.Config{
		FileSettings: model.FileSettings{MaxFileSize: &max},
	})
	api.On("LoadPluginConfiguration", mock.Anything).Return(nil)
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe().Return()

	idA := types.ID("https://a.example.atlassian.net")
	idB := types.ID("https://b.example.atlassian.net")
	p.rhsStatusCacheLock.Lock()
	p.rhsStatusCache = map[types.ID]*rhsStatusCacheEntry{
		idA: {fetchedAt: time.Now()},
		idB: {fetchedAt: time.Now()},
	}
	p.rhsStatusCacheLock.Unlock()

	require.NoError(t, p.OnConfigurationChange())
	assert.Empty(t, p.rhsStatusCache)
}
