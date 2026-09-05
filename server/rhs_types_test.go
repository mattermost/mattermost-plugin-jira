// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loadConfigLikeMattermost reproduces LoadPluginConfiguration's round-trip:
// lowercase every top-level key, json.Marshal, json.Unmarshal into externalConfig.
func loadConfigLikeMattermost(t *testing.T, stored map[string]any) externalConfig {
	t.Helper()
	finalConfig := make(map[string]any, len(stored))
	for key, value := range stored {
		finalConfig[strings.ToLower(key)] = value
	}
	b, err := json.Marshal(finalConfig)
	require.NoError(t, err)
	var dest externalConfig
	require.NoError(t, json.Unmarshal(b, &dest))
	return dest
}

func TestRHSConfigLoadFromLowercaseKeys(t *testing.T) {
	const instanceA = "https://a.example.atlassian.net"
	const instanceB = "https://b.example.atlassian.net"

	categoryEntry := map[string]any{
		"kind": "category",
		"key":  "indeterminate",
		"name": "In Progress",
	}
	statusEntry := map[string]any{
		"kind": "status",
		"id":   "10001",
		"name": "Submitted",
	}

	tests := map[string]struct {
		stored         map[string]any
		wantRHSEnabled bool
		wantTabs       map[string][]RHSTabEntry
		wantTabsNil    bool
	}{
		"two instances, keys already lowercased": {
			stored: map[string]any{
				"enablejirarhs": true,
				"rhsstatustabs": map[string]any{
					instanceA: []any{categoryEntry},
					instanceB: []any{statusEntry},
				},
			},
			wantRHSEnabled: true,
			wantTabs: map[string][]RHSTabEntry{
				instanceA: {{Kind: RHSTabKindCategory, Key: "indeterminate", Name: "In Progress"}},
				instanceB: {{Kind: RHSTabKindStatus, ID: "10001", Name: "Submitted"}},
			},
		},
		"schema-cased keys are lowercased before unmarshal": {
			stored: map[string]any{
				"EnableJiraRHS": true,
				"RHSStatusTabs": map[string]any{
					instanceA: []any{categoryEntry},
				},
			},
			wantRHSEnabled: true,
			wantTabs: map[string][]RHSTabEntry{
				instanceA: {{Kind: RHSTabKindCategory, Key: "indeterminate", Name: "In Progress"}},
			},
		},
		"absent key yields nil map and false flag": {
			stored:         map[string]any{},
			wantRHSEnabled: false,
			wantTabsNil:    true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := loadConfigLikeMattermost(t, tc.stored)
			assert.Equal(t, tc.wantRHSEnabled, got.EnableJiraRHS)
			if tc.wantTabsNil {
				assert.Nil(t, got.RHSStatusTabs)
				return
			}
			require.NotNil(t, got.RHSStatusTabs)
			assert.Equal(t, tc.wantTabs, got.RHSStatusTabs)
		})
	}
}

func TestRHSConfigCamelCaseTagWouldFail(t *testing.T) {
	payload := []byte(`{"rhsstatustabs":{"https://a.example.atlassian.net":[{"kind":"category","key":"indeterminate","name":"In Progress"}]}}`)

	var camel struct {
		RHSStatusTabs map[string][]RHSTabEntry `json:"rhsStatusTabs"`
	}
	require.NoError(t, json.Unmarshal(payload, &camel))
	// encoding/json matches tags case-insensitively; camelCase still populates.
	require.NotNil(t, camel.RHSStatusTabs)
	assert.Equal(t, "indeterminate", camel.RHSStatusTabs["https://a.example.atlassian.net"][0].Key)

	var lower struct {
		RHSStatusTabs map[string][]RHSTabEntry `json:"rhsstatustabs"`
	}
	require.NoError(t, json.Unmarshal(payload, &lower))
	require.NotNil(t, lower.RHSStatusTabs)
	assert.Equal(t, "indeterminate", lower.RHSStatusTabs["https://a.example.atlassian.net"][0].Key)

	var snake struct {
		RHSStatusTabs map[string][]RHSTabEntry `json:"rhs_status_tabs"`
	}
	require.NoError(t, json.Unmarshal(payload, &snake))
	assert.Nil(t, snake.RHSStatusTabs, "a json tag that is not a case-variant of rhsstatustabs must not populate")
}

func TestRHSConfigExternalConfigTagsAreLowercase(t *testing.T) {
	typ := reflect.TypeOf(externalConfig{})

	enable, ok := typ.FieldByName("EnableJiraRHS")
	require.True(t, ok)
	assert.Equal(t, "enablejirarhs", enable.Tag.Get("json"))

	tabs, ok := typ.FieldByName("RHSStatusTabs")
	require.True(t, ok)
	assert.Equal(t, "rhsstatustabs", tabs.Tag.Get("json"))
}

func TestRHSConfigDefaultTabs(t *testing.T) {
	got := rhsDefaultTabs()
	require.Len(t, got, 2)

	assert.Equal(t, RHSTabKindAssigned, got[0].Kind)
	assert.Equal(t, rhsAssignedTabName, got[0].Name)
	assert.Empty(t, got[0].Key)
	assert.Empty(t, got[0].ID)

	assert.Equal(t, RHSTabKindCategory, got[1].Kind)
	assert.Equal(t, statusCategoryKeyIndeterminate, got[1].Key)
	assert.Equal(t, "indeterminate", got[1].Key)
	assert.Equal(t, "In Progress", got[1].Name)
	assert.Empty(t, got[1].ID)
}

func TestRHSConfigSettingsInfoExposesFlag(t *testing.T) {
	tests := map[string]struct {
		rhsEnabled bool
	}{
		"enabled":  {rhsEnabled: true},
		"disabled": {rhsEnabled: false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			api := &plugintest.API{}
			p := setupTestPlugin(api)
			p.updateConfig(func(conf *config) {
				conf.EnableJiraRHS = tc.rhsEnabled
			})

			request := httptest.NewRequest(http.MethodGet, makeAPIRoute(routeAPISettingsInfo), nil)
			request.Header.Set(HeaderMattermostUserID, "test-user-id")
			w := httptest.NewRecorder()
			p.ServeHTTP(&plugin.Context{}, w, request)

			require.Equal(t, http.StatusOK, w.Result().StatusCode)
			require.Equal(t, "application/json", w.Result().Header.Get("Content-Type"))

			var body struct {
				UIEnabled  bool `json:"ui_enabled"`
				RHSEnabled bool `json:"rhs_enabled"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			assert.Equal(t, tc.rhsEnabled, body.RHSEnabled)
		})
	}
}

func TestJiraStatusUnmarshalsProjectScope(t *testing.T) {
	const raw = `[
		{"id":"3","name":"In Progress","statusCategory":{"id":4,"key":"indeterminate","name":"In Progress"}},
		{"id":"10042","name":"In Progress","statusCategory":{"id":4,"key":"indeterminate","name":"In Progress"},"scope":{"type":"PROJECT","project":{"id":"10000"}}}
	]`

	var statuses []*JiraStatus
	require.NoError(t, json.Unmarshal([]byte(raw), &statuses))
	require.Len(t, statuses, 2)
	assert.Nil(t, statuses[0].Scope)
	require.NotNil(t, statuses[1].Scope)
	require.NotNil(t, statuses[1].Scope.Project)
	assert.Equal(t, "PROJECT", statuses[1].Scope.Type)
	assert.Equal(t, "10000", statuses[1].Scope.Project.ID)
	assert.Equal(t, []string{"10000"}, uniqueStatusProjectIDs(statuses))
}

func TestUniqueIDs(t *testing.T) {
	assert.Empty(t, uniqueIDs(nil))
	assert.Empty(t, uniqueIDs([]string{}))
	assert.Empty(t, uniqueIDs([]string{"", ""}))
	assert.Equal(t, []string{"a", "b", "c"}, uniqueIDs([]string{"a", "", "b", "a", "c", "b", ""}))
}

func TestRHSTabIdentity(t *testing.T) {
	assert.Equal(t, "assigned", rhsTabIdentity(rhsAssignedTab()))
	assert.Equal(t, "assigned", rhsTabIdentity(RHSTabEntry{Kind: RHSTabKindAssigned}))
	assert.Equal(t, "category:indeterminate", rhsTabIdentity(RHSTabEntry{
		Kind: RHSTabKindCategory, Key: statusCategoryKeyIndeterminate, Name: "In Progress",
	}))
	assert.Equal(t, "status:10001", rhsTabIdentity(RHSTabEntry{
		Kind: RHSTabKindStatus, ID: "10001", Name: "Backlog",
	}))
}

func TestParseRHSTabIdentity(t *testing.T) {
	t.Run("empty and assigned", func(t *testing.T) {
		for _, s := range []string{"", "assigned"} {
			got, err := parseRHSTabIdentity(s)
			require.NoError(t, err)
			assert.Equal(t, rhsAssignedTab(), got)
			assert.Equal(t, rhsAssignedTabName, got.Name)
		}
	})

	t.Run("category and status may have empty name", func(t *testing.T) {
		cat, err := parseRHSTabIdentity("category:indeterminate")
		require.NoError(t, err)
		assert.Equal(t, RHSTabKindCategory, cat.Kind)
		assert.Equal(t, statusCategoryKeyIndeterminate, cat.Key)
		assert.Empty(t, cat.Name)
		assert.Empty(t, cat.ID)

		st, err := parseRHSTabIdentity("status:10001")
		require.NoError(t, err)
		assert.Equal(t, RHSTabKindStatus, st.Kind)
		assert.Equal(t, "10001", st.ID)
		assert.Empty(t, st.Name)
		assert.Empty(t, st.Key)
	})

	t.Run("unparseable", func(t *testing.T) {
		for _, s := range []string{"nope", "assigned:extra", "category:", "status:", "foo:bar"} {
			got, err := parseRHSTabIdentity(s)
			require.Error(t, err, s)
			assert.Equal(t, RHSTabEntry{}, got)
		}
	})
}

func TestPickRHSTab(t *testing.T) {
	tabs := []RHSTabEntry{
		rhsAssignedTab(),
		{Kind: RHSTabKindCategory, Key: statusCategoryKeyIndeterminate, Name: "In Progress"},
		{Kind: RHSTabKindStatus, ID: "10001", Name: "Backlog"},
	}

	assert.Equal(t, tabs[0], pickRHSTab(tabs, ""))
	assert.Equal(t, tabs[0], pickRHSTab(tabs, "assigned"))
	assert.Equal(t, tabs[1], pickRHSTab(tabs, "category:indeterminate"))
	assert.Equal(t, tabs[2], pickRHSTab(tabs, "status:10001"))
	assert.Equal(t, rhsAssignedTab(), pickRHSTab(tabs, "status:does-not-exist"))
	assert.Equal(t, rhsAssignedTab(), pickRHSTab(tabs, "category:missing"))
	assert.Equal(t, rhsAssignedTab(), pickRHSTab(tabs, "nope"))
	assert.Equal(t, rhsAssignedTab(), pickRHSTab(nil, "status:10001"))
}

func TestApplyStatusProjects(t *testing.T) {
	global := &JiraStatus{ID: "3", Name: "In Progress"}
	scoped := &JiraStatus{
		ID:   "10042",
		Name: "In Progress",
		Scope: &JiraStatusScope{
			Type:    "PROJECT",
			Project: &JiraStatusScopeProject{ID: "10000"},
		},
	}
	missing := &JiraStatus{
		ID:   "10043",
		Name: "In Progress",
		Scope: &JiraStatusScope{
			Type:    "PROJECT",
			Project: &JiraStatusScopeProject{ID: "999"},
		},
	}

	applyStatusProjects([]*JiraStatus{global, scoped, missing}, map[string]JiraStatusProject{
		"10000": {ID: "10000", Key: "PLAY", Name: "Playbooks"},
	})

	assert.Nil(t, global.Project)
	require.NotNil(t, scoped.Project)
	assert.Equal(t, "Playbooks", scoped.Project.Name)
	assert.Equal(t, "PLAY", scoped.Project.Key)
	assert.Nil(t, missing.Project)
}
