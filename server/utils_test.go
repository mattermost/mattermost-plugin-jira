// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseJIRAUsernamesFromText(t *testing.T) {
	tcs := []struct {
		Text     string
		Expected []string
	}{
		{Text: "[~]", Expected: []string{}},
		{Text: "[~j]", Expected: []string{"j"}},
		{Text: "[~jira]", Expected: []string{"jira"}},
		{Text: "[~jira.user]", Expected: []string{"jira.user"}},
		{Text: "[~jira_user]", Expected: []string{"jira_user"}},
		{Text: "[~jira-user]", Expected: []string{"jira-user"}},
		{Text: "[~jira:user]", Expected: []string{"jira:user"}},
		{Text: "[~jira_user_3]", Expected: []string{"jira_user_3"}},
		{Text: "[~jira-user-4]", Expected: []string{"jira-user-4"}},
		{Text: "[~JiraUser5]", Expected: []string{"JiraUser5"}},
		{Text: "[~jira-user+6]", Expected: []string{"jira-user+6"}},
		{Text: "[~2023]", Expected: []string{"2023"}},
		{Text: "[~jira.user@company.com]", Expected: []string{"jira.user@company.com"}},
		{Text: "[~jira_user@mattermost.com]", Expected: []string{"jira_user@mattermost.com"}},
		{Text: "[~jira-unique-user@mattermost.com] [~jira-unique-user@mattermost.com] [~jira-unique-user@mattermost.com]", Expected: []string{"jira-unique-user@mattermost.com"}},
		{Text: "[jira_incorrect_user]", Expected: []string{}},
		{Text: "[~jira_user_reviewer], Hi! Can you review the PR from [~jira_user_contributor]? Thanks!", Expected: []string{"jira_user_reviewer", "jira_user_contributor"}},
	}

	for _, tc := range tcs {
		assert.Equal(t, tc.Expected, parseJIRAUsernamesFromText(tc.Text))
	}
}
