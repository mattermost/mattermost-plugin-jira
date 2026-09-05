// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"errors"
	"fmt"
	"strings"

	jira "github.com/andygrunwald/go-jira"
)

type RHSTabKind string

const (
	RHSTabKindAssigned RHSTabKind = "assigned"
	RHSTabKindCategory RHSTabKind = "category"
	RHSTabKindStatus   RHSTabKind = "status"
)

const rhsAssignedTabName = "Assigned to me"

func rhsAssignedTab() RHSTabEntry {
	return RHSTabEntry{Kind: RHSTabKindAssigned, Name: rhsAssignedTabName}
}

func rhsTabIdentity(tab RHSTabEntry) string {
	switch tab.Kind {
	case RHSTabKindAssigned:
		return string(RHSTabKindAssigned)
	case RHSTabKindCategory:
		return string(RHSTabKindCategory) + ":" + tab.Key
	case RHSTabKindStatus:
		return string(RHSTabKindStatus) + ":" + tab.ID
	default:
		return string(tab.Kind)
	}
}

func parseRHSTabIdentity(s string) (RHSTabEntry, error) {
	if s == "" || s == string(RHSTabKindAssigned) {
		return rhsAssignedTab(), nil
	}
	kind, rest, ok := strings.Cut(s, ":")
	if !ok || rest == "" {
		return RHSTabEntry{}, fmt.Errorf("invalid rhs tab identity: %q", s)
	}
	switch RHSTabKind(kind) {
	case RHSTabKindAssigned:
		return RHSTabEntry{}, fmt.Errorf("invalid rhs tab identity: %q", s)
	case RHSTabKindCategory:
		return RHSTabEntry{Kind: RHSTabKindCategory, Key: rest}, nil
	case RHSTabKindStatus:
		return RHSTabEntry{Kind: RHSTabKindStatus, ID: rest}, nil
	default:
		return RHSTabEntry{}, fmt.Errorf("invalid rhs tab identity: %q", s)
	}
}

// Cloud statusCategory JQL operands. Do not use the numeric ids (2/4/3/1).
const (
	statusCategoryKeyNew           = "new"
	statusCategoryKeyIndeterminate = "indeterminate"
	statusCategoryKeyDone          = "done"
	statusCategoryKeyUndefined     = "undefined"
)

// Nested json names stay camelCase; LoadPluginConfiguration only lowercases
// the top-level setting key.
type RHSTabEntry struct {
	Kind RHSTabKind `json:"kind"`
	Key  string     `json:"key,omitempty"` // status-category key, e.g. "indeterminate"
	ID   string     `json:"id,omitempty"`  // numeric status id, e.g. "10001"
	Name string     `json:"name"`
}

// rhsDefaultTabs is Assigned plus In Progress when an instance has no stored tabs.
func rhsDefaultTabs() []RHSTabEntry {
	return []RHSTabEntry{
		rhsAssignedTab(),
		{Kind: RHSTabKindCategory, Key: statusCategoryKeyIndeterminate, Name: "In Progress"},
	}
}

type JiraStatus struct {
	ID             string             `json:"id"`
	Name           string             `json:"name"`
	StatusCategory JiraStatusCategory `json:"statusCategory"`
	Scope          *JiraStatusScope   `json:"scope,omitempty"`
	Project        *JiraStatusProject `json:"project,omitempty"`
}

// JiraStatusScope is Cloud's per-status scope. Team-managed statuses set
// type=PROJECT and project.id; company-managed statuses are typically GLOBAL.
type JiraStatusScope struct {
	Type    string                  `json:"type"`
	Project *JiraStatusScopeProject `json:"project,omitempty"`
}

type JiraStatusScopeProject struct {
	ID string `json:"id"`
}

// JiraStatusProject is the picker-facing project label. GET /status only
// returns scope.project.id; the admin /rhs/statuses path fills key/name via
// /project/search?id=....
type JiraStatusProject struct {
	ID   string `json:"id,omitempty"`
	Key  string `json:"key,omitempty"`
	Name string `json:"name,omitempty"`
}

type JiraStatusCategory struct {
	ID   int    `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

func statusProjectID(status *JiraStatus) string {
	if status == nil || status.Scope == nil || status.Scope.Project == nil {
		return ""
	}
	return status.Scope.Project.ID
}

func uniqueIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func uniqueStatusProjectIDs(statuses []*JiraStatus) []string {
	ids := make([]string, 0, len(statuses))
	for _, status := range statuses {
		ids = append(ids, statusProjectID(status))
	}
	return uniqueIDs(ids)
}

func applyStatusProjects(statuses []*JiraStatus, byID map[string]JiraStatusProject) {
	if len(byID) == 0 {
		return
	}
	for _, status := range statuses {
		id := statusProjectID(status)
		if id == "" {
			continue
		}
		project, ok := byID[id]
		if !ok {
			continue
		}
		projectCopy := project
		status.Project = &projectCopy
	}
}

type CloudSearchParams struct {
	JQL           string
	Fields        []string
	MaxResults    int
	NextPageToken string
}

// CloudSearchResult is GET /rest/api/3/search/jql. Cloud returns no total.
type CloudSearchResult struct {
	Issues        []jira.Issue `json:"issues"`
	NextPageToken string       `json:"nextPageToken"`
	IsLast        bool         `json:"isLast"`
}

var ErrRateLimited = errors.New("jira rate limited")

// ErrInvalidStatusCategory is returned when a statusCategory JQL operand is
// missing from validCategoryKeys. Assigned's hardcoded "done" uses this path.
var ErrInvalidStatusCategory = errors.New("invalid status category operand")

var ErrInvalidRHSSort = errors.New("invalid rhs sort field")

var ErrUnknownRHSTabKind = errors.New("unknown rhs tab kind")

var ErrInvalidRHSTab = errors.New("invalid rhs tab")

var ErrRHSNotCloud = errors.New("jira rhs is available for jira cloud only")

type RHSIssueStatus struct {
	Name        string `json:"name"`
	CategoryKey string `json:"categoryKey"`
}

type RHSIssue struct {
	Key              string         `json:"key"`
	Summary          string         `json:"summary"`
	BrowseURL        string         `json:"browseUrl"`
	Status           RHSIssueStatus `json:"status"`
	Priority         string         `json:"priority"`
	IssueType        string         `json:"issueType"`
	IssueTypeIconURL string         `json:"issueTypeIconUrl,omitempty"`
	Project          string         `json:"project"`
	Assignee         string         `json:"assignee"`
	Reporter         string         `json:"reporter"`
	Created          string         `json:"created"`
	Updated          string         `json:"updated"`
	DueDate          string         `json:"dueDate"`
	Labels           []string       `json:"labels"`
}
