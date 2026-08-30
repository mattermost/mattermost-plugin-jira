// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func canonicalValidCategoryKeys() map[string]bool {
	return map[string]bool{
		statusCategoryKeyNew:           true,
		statusCategoryKeyIndeterminate: true,
		statusCategoryKeyDone:          true,
		statusCategoryKeyUndefined:     true,
	}
}

type rhsFakeIssue struct {
	Key         string
	CategoryKey string
	StatusID    string
}

func assignedMembershipFixture() []rhsFakeIssue {
	return []rhsFakeIssue{
		{Key: "TES-NEW", CategoryKey: statusCategoryKeyNew, StatusID: "10001"},
		{Key: "TES-IP", CategoryKey: statusCategoryKeyIndeterminate, StatusID: "3"},
		{Key: "TES-DONE", CategoryKey: statusCategoryKeyDone, StatusID: "10000"},
		{Key: "TES-UNDEF", CategoryKey: statusCategoryKeyUndefined, StatusID: "1"},
	}
}

// issuesMatchingJQL models Cloud JQL: an unknown statusCategory makes "=" match
// nothing and "!=" match everything. Canonical keys are independent of our validator.
func issuesMatchingJQL(jql string, issues []rhsFakeIssue) []rhsFakeIssue {
	categoryRe := regexp.MustCompile(`statusCategory (!=|=) (\S+)`)
	statusRe := regexp.MustCompile(`status = (\S+)`)

	canonical := canonicalValidCategoryKeys()

	if m := categoryRe.FindStringSubmatch(jql); len(m) == 3 {
		op, operand := m[1], m[2]
		resolved := canonical[operand]
		var out []rhsFakeIssue
		for _, issue := range issues {
			switch {
			case op == "=" && !resolved:
				// Cloud: matches nothing
			case op == "!=" && !resolved:
				out = append(out, issue) // Cloud: matches everything
			case op == "=" && issue.CategoryKey == operand:
				out = append(out, issue)
			case op == "!=" && issue.CategoryKey != operand:
				out = append(out, issue)
			}
		}
		return out
	}
	if m := statusRe.FindStringSubmatch(jql); len(m) == 2 {
		id := m[1]
		var out []rhsFakeIssue
		for _, issue := range issues {
			if issue.StatusID == id {
				out = append(out, issue)
			}
		}
		return out
	}
	return nil
}

func issueKeys(issues []rhsFakeIssue) []string {
	keys := make([]string, 0, len(issues))
	for _, issue := range issues {
		keys = append(keys, issue.Key)
	}
	return keys
}

func TestRHSJQLBuildExactStrings(t *testing.T) {
	valid := canonicalValidCategoryKeys()
	assigned := RHSTabEntry{Kind: RHSTabKindAssigned, Name: rhsAssignedTabName}
	inProgress := RHSTabEntry{Kind: RHSTabKindCategory, Key: statusCategoryKeyIndeterminate, Name: "In Progress"}
	todo := RHSTabEntry{Kind: RHSTabKindCategory, Key: statusCategoryKeyNew, Name: "To Do"}
	status := RHSTabEntry{Kind: RHSTabKindStatus, ID: "10001", Name: "Backlog"}
	statusOpen := RHSTabEntry{Kind: RHSTabKindStatus, ID: "3", Name: "In Progress"}

	tests := map[string]struct {
		tab  RHSTabEntry
		sort string
		want string
	}{
		"assigned updated": {
			tab: assigned, sort: "updated",
			want: "assignee = currentUser() AND statusCategory != done ORDER BY updated DESC, key ASC",
		},
		"assigned created": {
			tab: assigned, sort: "created",
			want: "assignee = currentUser() AND statusCategory != done ORDER BY created DESC, key ASC",
		},
		"category indeterminate updated": {
			tab: inProgress, sort: "updated",
			want: "assignee = currentUser() AND statusCategory = indeterminate ORDER BY updated DESC, key ASC",
		},
		"category indeterminate created": {
			tab: inProgress, sort: "created",
			want: "assignee = currentUser() AND statusCategory = indeterminate ORDER BY created DESC, key ASC",
		},
		"category new updated": {
			tab: todo, sort: "updated",
			want: "assignee = currentUser() AND statusCategory = new ORDER BY updated DESC, key ASC",
		},
		"status 10001 updated": {
			tab: status, sort: "updated",
			want: "assignee = currentUser() AND status = 10001 ORDER BY updated DESC, key ASC",
		},
		"status 3 created": {
			tab: statusOpen, sort: "created",
			want: "assignee = currentUser() AND status = 3 ORDER BY created DESC, key ASC",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := buildTabJQL(tc.tab, tc.sort, valid)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRHSJQLAssignedErrorsWhenDoneOmittedFromValidKeys(t *testing.T) {
	valid := map[string]bool{
		statusCategoryKeyNew:           true,
		statusCategoryKeyIndeterminate: true,
		statusCategoryKeyUndefined:     true,
		// done is deliberately omitted
	}
	tab := RHSTabEntry{Kind: RHSTabKindAssigned, Name: rhsAssignedTabName}

	got, err := buildTabJQL(tab, "updated", valid)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidStatusCategory)
	assert.Empty(t, got, "Assigned must not emit JQL when done is not in validCategoryKeys")

	t.Run("nil map", func(t *testing.T) {
		got, err := buildTabJQL(RHSTabEntry{Kind: RHSTabKindAssigned, Name: rhsAssignedTabName}, "updated", nil)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidStatusCategory)
		assert.Empty(t, got)
	})
}

func TestRHSJQLCategoryErrorsWhenKeyMissingFromValidKeys(t *testing.T) {
	valid := canonicalValidCategoryKeys()
	tab := RHSTabEntry{Kind: RHSTabKindCategory, Key: "bogusCategoryXYZ", Name: "Bogus"}

	got, err := buildTabJQL(tab, "updated", valid)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidStatusCategory)
	assert.Empty(t, got)
}

func TestRHSJQLRejectsSortOutsideAllowlist(t *testing.T) {
	valid := canonicalValidCategoryKeys()
	tab := RHSTabEntry{Kind: RHSTabKindAssigned, Name: rhsAssignedTabName}

	tests := map[string]string{
		"empty":        "",
		"uppercase":    "UPDATED",
		"priority":     "priority",
		"updated DESC": "updated DESC",
	}

	for name, sortField := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := buildTabJQL(tab, sortField, valid)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidRHSSort)
			assert.Empty(t, got)
		})
	}
}

func TestRHSJQLAssignedExcludesDoneMembership(t *testing.T) {
	jql, err := buildTabJQL(
		RHSTabEntry{Kind: RHSTabKindAssigned, Name: rhsAssignedTabName},
		"updated",
		canonicalValidCategoryKeys(),
	)
	require.NoError(t, err)

	got := issuesMatchingJQL(jql, assignedMembershipFixture())
	keys := issueKeys(got)

	assert.NotContains(t, keys, "TES-DONE")
	assert.Contains(t, keys, "TES-NEW")
	assert.Contains(t, keys, "TES-IP")
	assert.Contains(t, keys, "TES-UNDEF")
}

func TestRHSJQLNegatedBogusOperandWidensToIncludeDone(t *testing.T) {
	jql := "assignee = currentUser() AND statusCategory != bogusCategoryXYZ ORDER BY updated DESC, key ASC"
	got := issuesMatchingJQL(jql, assignedMembershipFixture())
	keys := issueKeys(got)

	assert.Contains(t, keys, "TES-DONE", "Cloud != <bogus> matches everything; the interpreter must model that")
	assert.Contains(t, keys, "TES-NEW")
	assert.GreaterOrEqual(t, len(keys), 4)
}

func TestRHSJQLUnknownKindAndEmptyStatusID(t *testing.T) {
	valid := canonicalValidCategoryKeys()

	t.Run("unknown kind", func(t *testing.T) {
		got, err := buildTabJQL(RHSTabEntry{Kind: "nope"}, "updated", valid)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrUnknownRHSTabKind)
		assert.Empty(t, got)
	})

	t.Run("empty status id", func(t *testing.T) {
		got, err := buildTabJQL(RHSTabEntry{Kind: RHSTabKindStatus, ID: ""}, "updated", valid)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidRHSTab)
		assert.Empty(t, got)
	})
}

func TestRHSJQLResolveTabsAssignedAlwaysFirst(t *testing.T) {
	statuses := []*JiraStatus{
		{ID: "3"},
		{ID: "10001"},
	}
	categories := []*JiraStatusCategory{
		{Key: statusCategoryKeyIndeterminate},
		{Key: statusCategoryKeyNew},
		{Key: statusCategoryKeyDone},
		{Key: statusCategoryKeyUndefined},
	}

	configured := []RHSTabEntry{
		{Kind: RHSTabKindCategory, Key: statusCategoryKeyIndeterminate, Name: "In Progress"},
		{Kind: RHSTabKindAssigned, Name: rhsAssignedTabName},
		{Kind: RHSTabKindStatus, ID: "10001", Name: "Backlog"},
	}

	got := resolveTabs(configured, statuses, categories)
	require.Len(t, got, 3)
	assert.Equal(t, RHSTabKindAssigned, got[0].Kind)
	assert.Equal(t, rhsAssignedTabName, got[0].Name)

	assignedCount := 0
	for _, tab := range got {
		if tab.Kind == RHSTabKindAssigned {
			assignedCount++
		}
	}
	assert.Equal(t, 1, assignedCount)

	assert.Equal(t, RHSTabKindCategory, got[1].Kind)
	assert.Equal(t, statusCategoryKeyIndeterminate, got[1].Key)
	assert.Equal(t, RHSTabKindStatus, got[2].Kind)
	assert.Equal(t, "10001", got[2].ID)
}

func TestRHSJQLResolveTabsHidesVanishedStatusAndCategory(t *testing.T) {
	statuses := []*JiraStatus{
		{ID: "3", Name: "In Progress"},
		{ID: "10001", Name: "Backlog"},
	}
	categories := []*JiraStatusCategory{
		{Key: statusCategoryKeyUndefined},
		{Key: statusCategoryKeyNew},
		{Key: statusCategoryKeyIndeterminate},
		{Key: statusCategoryKeyDone},
	}
	configured := []RHSTabEntry{
		{Kind: RHSTabKindStatus, ID: "99999", Name: "Gone"},
		{Kind: RHSTabKindCategory, Key: "not-a-real-key", Name: "Bogus"},
		{Kind: RHSTabKindCategory, Key: statusCategoryKeyIndeterminate, Name: "In Progress"},
		{Kind: RHSTabKindStatus, ID: "10001", Name: "Backlog"},
		{Kind: "nope"},
	}

	got := resolveTabs(configured, statuses, categories)
	require.Len(t, got, 3)
	assert.Equal(t, RHSTabKindAssigned, got[0].Kind)
	assert.Equal(t, rhsAssignedTabName, got[0].Name)
	assert.Equal(t, RHSTabKindCategory, got[1].Kind)
	assert.Equal(t, statusCategoryKeyIndeterminate, got[1].Key)
	assert.Equal(t, "In Progress", got[1].Name)
	assert.Equal(t, RHSTabKindStatus, got[2].Kind)
	assert.Equal(t, "10001", got[2].ID)
	assert.Equal(t, "Backlog", got[2].Name)
}

func TestRHSJQLResolveTabsEmptyConfigUsesDefaultSeed(t *testing.T) {
	statuses := []*JiraStatus{
		{ID: "3"},
		{ID: "10001"},
	}
	categories := []*JiraStatusCategory{
		{Key: statusCategoryKeyUndefined},
		{Key: statusCategoryKeyNew},
		{Key: statusCategoryKeyIndeterminate},
		{Key: statusCategoryKeyDone},
	}

	t.Run("nil config", func(t *testing.T) {
		got := resolveTabs(nil, statuses, categories)
		require.Len(t, got, 2)
		assert.Equal(t, RHSTabKindAssigned, got[0].Kind)
		assert.Equal(t, RHSTabKindCategory, got[1].Kind)
		assert.Equal(t, statusCategoryKeyIndeterminate, got[1].Key)
	})

	t.Run("empty config is Assigned only", func(t *testing.T) {
		got := resolveTabs([]RHSTabEntry{}, statuses, categories)
		require.Len(t, got, 1)
		assert.Equal(t, RHSTabKindAssigned, got[0].Kind)
		assert.Equal(t, rhsAssignedTabName, got[0].Name)
	})

	t.Run("nil config and nil categories", func(t *testing.T) {
		got := resolveTabs(nil, statuses, nil)
		require.Len(t, got, 1)
		assert.Equal(t, RHSTabKindAssigned, got[0].Kind)
		assert.Equal(t, rhsAssignedTabName, got[0].Name)
	})
}
