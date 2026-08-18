// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"encoding/json"
	"testing"

	jira "github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

type teamFieldTestClient struct {
	Client
	fields []JiraField
	err    error
}

func (c teamFieldTestClient) ListFields() ([]JiraField, error) {
	return c.fields, c.err
}

func setupTeamFieldPlugin(t *testing.T) (*Plugin, *plugintest.API) {
	t.Helper()

	api := &plugintest.API{}
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()

	p := &Plugin{}
	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)
	p.teamFieldCache = make(map[types.ID]map[string]struct{})

	return p, api
}

func TestGetTeamFieldKeysNoHardcodedFallback(t *testing.T) {
	p, api := setupTeamFieldPlugin(t)
	api.On("KVGet", mock.AnythingOfType("string")).Return(nil, (*model.AppError)(nil))

	keys := p.getTeamFieldKeys(testInstance1.InstanceID)

	assert.Empty(t, keys, "an instance with no discovered team field must not fall back to a hardcoded key")
}

func TestGetTeamFieldKeysFromKV(t *testing.T) {
	p, api := setupTeamFieldPlugin(t)

	stored, err := json.Marshal([]string{"customfield_10800"})
	require.NoError(t, err)
	api.On("KVGet", mock.AnythingOfType("string")).Return(stored, (*model.AppError)(nil))

	keys := p.getTeamFieldKeys(testInstance1.InstanceID)
	assert.Equal(t, map[string]struct{}{"customfield_10800": {}}, keys)

	// A second read is served from memory, so the KV round trip happens once per process.
	keys = p.getTeamFieldKeys(testInstance1.InstanceID)
	assert.Equal(t, map[string]struct{}{"customfield_10800": {}}, keys)
	api.AssertNumberOfCalls(t, "KVGet", 1)
}

func TestGetTeamFieldKeysKVError(t *testing.T) {
	p, api := setupTeamFieldPlugin(t)
	api.On("KVGet", mock.AnythingOfType("string")).Return(nil, &model.AppError{Message: "kv unavailable"})

	assert.Empty(t, p.getTeamFieldKeys(testInstance1.InstanceID))

	// A failed load must not be cached, otherwise a transient KV error would
	// disable team filters until the next restart.
	assert.Empty(t, p.getTeamFieldKeys(testInstance1.InstanceID))
	api.AssertNumberOfCalls(t, "KVGet", 2)
}

func TestCacheTeamFieldKeysPersists(t *testing.T) {
	p, api := setupTeamFieldPlugin(t)

	var persisted []byte
	api.On("KVSetWithOptions", mock.AnythingOfType("string"), mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			persisted, _ = args.Get(1).([]byte)
		}).Return(true, (*model.AppError)(nil))

	p.cacheTeamFieldKeys(testInstance1.InstanceID, []string{"CustomField_10800", " ", ""})

	assert.Equal(t, map[string]struct{}{"customfield_10800": {}}, p.getTeamFieldKeys(testInstance1.InstanceID))

	var stored []string
	require.NoError(t, json.Unmarshal(persisted, &stored))
	assert.Equal(t, []string{"customfield_10800"}, stored)

	// Re-caching a known key must not write to KV again.
	p.cacheTeamFieldKeys(testInstance1.InstanceID, []string{"customfield_10800"})
	api.AssertNumberOfCalls(t, "KVSetWithOptions", 1)
}

func TestDiscoverTeamFieldKeys(t *testing.T) {
	for name, tc := range map[string]struct {
		client   teamFieldTestClient
		expected map[string]struct{}
	}{
		"atlassian team field": {
			client: teamFieldTestClient{fields: []JiraField{
				{ID: "customfield_10800", Name: "Team", Schema: schemaWithCustom(teamFieldSchema)},
				{ID: "customfield_10001", Name: "Sprint", Schema: schemaWithCustom("com.pyxis.greenhopper.jira:gh-sprint")},
			}},
			expected: map[string]struct{}{"customfield_10800": {}},
		},
		"advanced roadmaps cloud and dc": {
			client: teamFieldTestClient{fields: []JiraField{
				{ID: "customfield_10500", Schema: schemaWithCustom(teamAdvancedRoadmapsSchema)},
				{ID: "customfield_10600", Schema: schemaWithCustom(teamAdvancedRoadmapsDC)},
			}},
			expected: map[string]struct{}{"customfield_10500": {}, "customfield_10600": {}},
		},
		"no team field": {
			client: teamFieldTestClient{fields: []JiraField{
				{ID: "customfield_10001", Schema: schemaWithCustom("com.pyxis.greenhopper.jira:gh-sprint")},
			}},
			expected: map[string]struct{}{},
		},
		"jira error": {
			client:   teamFieldTestClient{err: errors.New("jira unavailable")},
			expected: map[string]struct{}{},
		},
	} {
		t.Run(name, func(t *testing.T) {
			p, api := setupTeamFieldPlugin(t)
			api.On("KVSetWithOptions", mock.AnythingOfType("string"), mock.Anything, mock.Anything).
				Return(true, (*model.AppError)(nil)).Maybe()
			api.On("KVGet", mock.AnythingOfType("string")).Return(nil, (*model.AppError)(nil)).Maybe()

			p.discoverTeamFieldKeys(testInstance1.InstanceID, tc.client)

			assert.Equal(t, tc.expected, p.getTeamFieldKeys(testInstance1.InstanceID))
		})
	}
}

func TestGetTeamFieldKeysKeepsConcurrentDiscovery(t *testing.T) {
	p, api := setupTeamFieldPlugin(t)
	api.On("KVSetWithOptions", mock.AnythingOfType("string"), mock.Anything, mock.Anything).
		Return(true, (*model.AppError)(nil))

	// Simulate a discovery landing while this instance's KV read is in flight.
	api.On("KVGet", mock.AnythingOfType("string")).
		Run(func(_ mock.Arguments) {
			p.cacheTeamFieldKeys(testInstance1.InstanceID, []string{"customfield_10800"})
		}).Return(nil, (*model.AppError)(nil))

	assert.Equal(t, map[string]struct{}{"customfield_10800": {}}, p.getTeamFieldKeys(testInstance1.InstanceID))
	assert.Equal(t, map[string]struct{}{"customfield_10800": {}}, p.getTeamFieldKeys(testInstance1.InstanceID))
}

const testTeamID = "d885d551-c24d-45d5-a8a3-5be1808be30f"

func teamFilterWebhook(teamFieldKey string) *webhook {
	fields := &jira.IssueFields{
		Type:     jira.IssueType{ID: "10001"},
		Project:  jira.Project{Key: mockProjectKey},
		Unknowns: map[string]interface{}{},
	}
	if teamFieldKey != "" {
		fields.Unknowns[teamFieldKey] = map[string]interface{}{"id": testTeamID}
	}

	return &webhook{
		JiraWebhook: &JiraWebhook{Issue: jira.Issue{Fields: fields}},
		eventTypes:  NewStringSet(eventCreated),
	}
}

func teamFilters(inclusion string) SubscriptionFilters {
	return SubscriptionFilters{
		Events:     NewStringSet(eventCreated),
		IssueTypes: NewStringSet("10001"),
		Projects:   NewStringSet(mockProjectKey),
		Fields: []FieldFilter{{
			Key:       TeamFilter,
			Inclusion: inclusion,
			Values:    NewStringSet(testTeamID),
		}},
	}
}

func TestMatchesSubscriptionFiltersResolvedTeamField(t *testing.T) {
	p, api := setupTeamFieldPlugin(t)

	stored, err := json.Marshal([]string{"customfield_10800"})
	require.NoError(t, err)
	api.On("KVGet", mock.AnythingOfType("string")).Return(stored, (*model.AppError)(nil))

	assert.True(t, p.matchesSubscriptionFilters(
		teamFilterWebhook("customfield_10800"), testInstance1.InstanceID, teamFilters(FilterIncludeAny)))

	assert.False(t, p.matchesSubscriptionFilters(
		teamFilterWebhook("customfield_10800"), testInstance1.InstanceID, teamFilters(FilterExcludeAny)))
}

func TestMatchesSubscriptionFiltersUnresolvedTeamFieldFailsClosed(t *testing.T) {
	for _, inclusion := range []string{FilterIncludeAny, FilterIncludeAll, FilterExcludeAny, FilterEmpty, FilterIncludeOrEmpty} {
		t.Run(inclusion, func(t *testing.T) {
			p, api := setupTeamFieldPlugin(t)
			api.On("KVGet", mock.AnythingOfType("string")).Return(nil, (*model.AppError)(nil))

			matched := p.matchesSubscriptionFilters(
				teamFilterWebhook("customfield_10800"), testInstance1.InstanceID, teamFilters(inclusion))

			assert.False(t, matched, "an unresolved team field must not select the channel")
		})
	}
}

func TestHasTeamFilter(t *testing.T) {
	assert.True(t, hasTeamFilter(SubscriptionFilters{Fields: []FieldFilter{
		{Key: securityLevelField},
		{Key: TeamFilter},
	}}))
	assert.False(t, hasTeamFilter(SubscriptionFilters{Fields: []FieldFilter{
		{Key: securityLevelField},
	}}))
}

func schemaWithCustom(custom string) struct {
	Custom string `json:"custom"`
} {
	return struct {
		Custom string `json:"custom"`
	}{Custom: custom}
}
