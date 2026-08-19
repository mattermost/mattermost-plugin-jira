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

// Canonical Jira Cloud status-category keys. Use these as JQL operands.
// Do not hardcode the numeric ids (2/4/3/1) — they are counterintuitive.
const (
	statusCategoryKeyNew           = "new"
	statusCategoryKeyIndeterminate = "indeterminate"
	statusCategoryKeyDone          = "done"
	statusCategoryKeyUndefined     = "undefined"
)

// RHSTabEntry is one admin-configured (or virtual-seed) tab.
// Nested fields keep camelCase-style json names; LoadPluginConfiguration only
// lowercases the top-level setting key.
type RHSTabEntry struct {
	Kind RHSTabKind `json:"kind"`
	Key  string     `json:"key,omitempty"` // status-category key, e.g. "indeterminate"
	ID   string     `json:"id,omitempty"`  // numeric status id, e.g. "10001"
	Name string     `json:"name"`
}

// rhsDefaultTabs is the virtual seed (D2): Assigned (always-on) plus In Progress.
// Called when an instance has no stored tab config. Does not write plugin config.
func rhsDefaultTabs() []RHSTabEntry {
	return []RHSTabEntry{
		{Kind: RHSTabKindAssigned, Name: "Assigned"},
		{Kind: RHSTabKindCategory, Key: statusCategoryKeyIndeterminate, Name: "In Progress"},
	}
}

// JiraStatus is one instance-wide workflow status from GET /rest/api/3/status.
// Extra JSON fields (self, description, iconUrl) are ignored.
type JiraStatus struct {
	ID             string             `json:"id"`
	Name           string             `json:"name"`
	StatusCategory JiraStatusCategory `json:"statusCategory"`
}

// JiraStatusCategory is a Cloud status category.
// ID is numeric (official schema int64). Key is the JQL operand
// (new / indeterminate / done / undefined).
type JiraStatusCategory struct {
	ID   int    `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

// CloudSearchParams is the input to jiraCloudClient.SearchJQL.
type CloudSearchParams struct {
	JQL           string
	Fields        []string
	MaxResults    int
	NextPageToken string
}

// CloudSearchResult is GET /rest/api/3/search/jql.
// Cloud returns no total — do not add one.
type CloudSearchResult struct {
	Issues        []jira.Issue `json:"issues"`
	NextPageToken string       `json:"nextPageToken"`
	IsLast        bool         `json:"isLast"`
}

// ErrRateLimited is returned when a 429 is not retried (global quota) or
// when retries are exhausted. Phase 4 maps this to JSON error rate_limited.
var ErrRateLimited = errors.New("jira rate limited")
