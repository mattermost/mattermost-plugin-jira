// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

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
