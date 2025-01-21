// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEndpointNameFromRequest(t *testing.T) {
	tests := []struct {
		name, url, method, expected string
	}{
		{"GetCreateIssueMetadata", "https://hostname/rest/api/2/issue", "GET", "api/jira/2/issue/GET"},
		{"AddAttachment", "https://hostname/2/issue/MM-1543/attachments", "POST", "api/jira/2/issue/attachments/POST"},
		{"GetUserGroups", "https://hostname/3/user/groups", "GET", "api/jira/3/user/groups/GET"},
		{"Myself", "https://hostname/2/myself", "GET", "api/jira/2/myself/GET"},
		{"SearchUserAssignableToIssue", "https://hostname/2/user/assignable/search", "GET", "api/jira/2/user/assignable/search/GET"},
		{"GetProject", "https://hostname/2/project/XYZ", "GET", "api/jira/2/project/GET"},
		{"GetIssue", "https://hostname/2/issue/XYZ-1234", "GET", "api/jira/2/issue/GET"},
		{"CreateIssue", "https://hostname/2/issue", "POST", "api/jira/2/issue/POST"},
		{"GetTransitions", "https://hostname/2/issue/XYZ-1234/transitions", "GET", "api/jira/2/issue/transitions/GET"},
		{"UpdateIssueAssignee", "https://hostname/2/issue/XYZ-1234/assignee", "PUT", "api/jira/2/issue/assignee/PUT"},
		{"AddComment", "https://hostname/2/issue/XYZ-1234/comment", "POST", "api/jira/2/issue/comment/POST"},
		{"UpdateComment", "https://hostname/2/issue/XYZ-1234/comment/XXX", "PUT", "api/jira/2/issue/comment/PUT"},
		{"SearchIssues", "https://hostname/2/search", "GET", "api/jira/2/search/GET"},
		{"DoTransition", "https://hostname/2/issue/XYZ-4321/transitions", "POST", "api/jira/2/issue/transitions/POST"},
		{"GetCreateMetaInfo", "https://hostname/2/issue/createmeta", "GET", "api/jira/2/issue/createmeta/GET"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := http.NewRequest(tt.method, tt.url, nil)
			require.NoError(t, err)
			require.NotNil(t, r)

			name := endpointNameFromRequest(r)
			require.Equal(t, tt.expected, name)
		})
	}
}
