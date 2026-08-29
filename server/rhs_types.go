// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"errors"

	jira "github.com/andygrunwald/go-jira"
)

type RHSTabKind string

const (
	RHSTabKindAssigned RHSTabKind = "assigned"
	RHSTabKindCategory RHSTabKind = "category"
	RHSTabKindStatus   RHSTabKind = "status"
)

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
		{Kind: RHSTabKindAssigned, Name: "Assigned"},
		{Kind: RHSTabKindCategory, Key: statusCategoryKeyIndeterminate, Name: "In Progress"},
	}
}

type JiraStatus struct {
	ID             string             `json:"id"`
	Name           string             `json:"name"`
	StatusCategory JiraStatusCategory `json:"statusCategory"`
}

type JiraStatusCategory struct {
	ID   int    `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
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
