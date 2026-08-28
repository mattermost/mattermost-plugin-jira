// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import "fmt"

func validCategoryKeysFrom(categories []*JiraStatusCategory) map[string]bool {
	out := make(map[string]bool, len(categories))
	for _, c := range categories {
		if c == nil || c.Key == "" {
			continue
		}
		out[c.Key] = true
	}
	return out
}

func statusIDSet(statuses []*JiraStatus) map[string]bool {
	out := make(map[string]bool, len(statuses))
	for _, s := range statuses {
		if s == nil || s.ID == "" {
			continue
		}
		out[s.ID] = true
	}
	return out
}

// Assigned's hardcoded done must go through this: Cloud treats != <unknown> as match-all.
func validateStatusCategoryOperand(operand string, validCategoryKeys map[string]bool) error {
	if operand == "" || validCategoryKeys == nil || !validCategoryKeys[operand] {
		return fmt.Errorf("%w: %q", ErrInvalidStatusCategory, operand)
	}
	return nil
}

func rhsOrderBy(sortField string) (string, error) {
	switch sortField {
	case "updated", "created":
		return " ORDER BY " + sortField + " DESC, key ASC", nil
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidRHSSort, sortField)
	}
}

func buildTabJQL(tab RHSTabEntry, sortField string, validCategoryKeys map[string]bool) (string, error) {
	order, err := rhsOrderBy(sortField)
	if err != nil {
		return "", err
	}

	const assignee = "assignee = currentUser()"

	switch tab.Kind {
	case RHSTabKindAssigned:
		if err := validateStatusCategoryOperand(statusCategoryKeyDone, validCategoryKeys); err != nil {
			return "", err
		}
		return assignee + " AND statusCategory != " + statusCategoryKeyDone + order, nil

	case RHSTabKindCategory:
		if err := validateStatusCategoryOperand(tab.Key, validCategoryKeys); err != nil {
			return "", err
		}
		return assignee + " AND statusCategory = " + tab.Key + order, nil

	case RHSTabKindStatus:
		if tab.ID == "" {
			return "", fmt.Errorf("%w: empty status id", ErrInvalidRHSTab)
		}
		return assignee + " AND status = " + tab.ID + order, nil

	default:
		return "", fmt.Errorf("%w: %q", ErrUnknownRHSTabKind, tab.Kind)
	}
}

func resolveTabs(configured []RHSTabEntry, statuses []*JiraStatus, categories []*JiraStatusCategory) []RHSTabEntry {
	out := []RHSTabEntry{{Kind: RHSTabKindAssigned, Name: "Assigned"}}

	extras := configured
	if len(configured) == 0 {
		extras = rhsDefaultTabs()[1:]
	}

	validCats := validCategoryKeysFrom(categories)
	validIDs := statusIDSet(statuses)

	for _, tab := range extras {
		switch tab.Kind {
		case RHSTabKindAssigned:
			continue
		case RHSTabKindCategory:
			if tab.Key != "" && validCats[tab.Key] {
				out = append(out, tab)
			}
		case RHSTabKindStatus:
			if tab.ID != "" && validIDs[tab.ID] {
				out = append(out, tab)
			}
		}
	}
	return out
}
