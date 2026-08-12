// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

func TestGetJiraBaseURLTrimsTrailingSlash(t *testing.T) {
	for name, tc := range map[string]struct {
		instance Instance
		expected string
	}{
		"cloud-oauth, no trailing slash": {
			instance: &cloudOAuthInstance{JiraBaseURL: "https://mmtest.atlassian.net"},
			expected: "https://mmtest.atlassian.net",
		},
		"cloud-oauth, trailing slash": {
			instance: &cloudOAuthInstance{JiraBaseURL: "https://mmtest.atlassian.net/"},
			expected: "https://mmtest.atlassian.net",
		},
		"cloud-oauth, repeated trailing slashes": {
			instance: &cloudOAuthInstance{JiraBaseURL: "https://mmtest.atlassian.net///"},
			expected: "https://mmtest.atlassian.net",
		},
		"cloud, trailing slash": {
			instance: &cloudInstance{
				AtlassianSecurityContext: &AtlassianSecurityContext{BaseURL: "https://mmtest.atlassian.net/"},
			},
			expected: "https://mmtest.atlassian.net",
		},
		"server, trailing slash": {
			instance: &serverInstance{
				InstanceCommon: &InstanceCommon{InstanceID: types.ID("https://jira.example.com/")},
			},
			expected: "https://jira.example.com",
		},
		"server, no trailing slash": {
			instance: &serverInstance{
				InstanceCommon: &InstanceCommon{InstanceID: types.ID("https://jira.example.com")},
			},
			expected: "https://jira.example.com",
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.instance.GetJiraBaseURL())
		})
	}
}
