// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"os"
	"testing"

	"github.com/andygrunwald/go-jira"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarkdown(t *testing.T) {
	f, err := os.Open("testdata/webhook-issue-created.json")
	require.NoError(t, err)
	defer f.Close()
	wh, _, err := ParseWebhook(f)
	require.NoError(t, err)
	w := wh.(*webhook)
	require.NotNil(t, w)
	require.Equal(t,
		"Test User created story [TES-41](https://some-instance-test.atlassian.net/browse/TES-41)",
		w.headline)
}

func TestWebhookVariousErrors(t *testing.T) {
	assert.Equal(t, "", mdUser(nil))

	wh := &webhook{
		JiraWebhook: &JiraWebhook{
			Issue: jira.Issue{
				Fields: &jira.IssueFields{},
			},
		},
	}

	assert.Equal(t, "", wh.mdJiraLink("test", "/test"))
	assert.Equal(t, "", wh.mdIssueDescription())
	assert.Equal(t, "", wh.mdIssueSummary())
	assert.Equal(t, "_nobody_", wh.mdIssueAssignee())
	assert.Equal(t, "", wh.mdIssueType())
	assert.Equal(t, " ", wh.mdSummaryLink())
	assert.Equal(t, " ", wh.mdKeyLink())
	assert.Equal(t, "", wh.mdUser())
}

func TestTruncate(t *testing.T) {
	assert.Equal(t, "12345", truncate("12345", 5))
	assert.Equal(t, "12345", truncate("12345", 6))
	assert.Equal(t, "1...", truncate("12345", 4))
	assert.Equal(t, "12", truncate("12345", 2))
	assert.Equal(t, "1", truncate("12345", 1))
	assert.Equal(t, "", truncate("12345", 0))
	assert.Equal(t, "12345", truncate("12345", -1))
}

func TestJiraLink(t *testing.T) {
	var jwh JiraWebhook
	jwh.Issue.Self = "http://localhost:8080/rest/api/2/issue/10006"
	assert.Equal(t, "[1](http://localhost:8080/XXX)", jwh.mdJiraLink("1", "/XXX"))

	jwh.Issue.Self = "http://localhost:8080/foo/bar/rest/api/2/issue/10006"
	assert.Equal(t, "[1](http://localhost:8080/foo/bar/QWERTY)", jwh.mdJiraLink("1", "/QWERTY"))
}
