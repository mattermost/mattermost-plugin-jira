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

func TestParse(t *testing.T) {
	for _, tc := range []struct {
		file             string
		expectedStyle    string
		expectedHeadline string
		expectedDetails  string
		expectedText     string
	}{{
		file:             "testdata/webhook-comment-created.json",
		expectedStyle:    mdUpdateStyle,
		expectedHeadline: "Test User commented on story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41) (#TES-41)",
		expectedText:     "Added a comment",
	}, {
		file:             "testdata/webhook-comment-deleted.json",
		expectedStyle:    mdUpdateStyle,
		expectedHeadline: "Test User removed a comment from story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41) (#TES-41)",
	}, {
		file:             "testdata/webhook-comment-updated.json",
		expectedStyle:    mdUpdateStyle,
		expectedHeadline: "Test User edited a comment in story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41) (#TES-41)",
		expectedText:     "Added a comment, then edited it",
	}, {
		file:             "testdata/webhook-issue-created.json",
		expectedStyle:    mdRootStyle,
		expectedHeadline: "Test User created story [TES-41: Unit test summary](https://some-instance-test.atlassian.net/browse/TES-41) (#jira-new #TES-41)",
		expectedDetails:  "Priority: **High**, Reported by: **Test User**, Labels: test-label",
		expectedText:     "Unit test description, not that long",
	}, {
		file:             "testdata/webhook-issue-updated-assigned-nobody.json",
		expectedStyle:    mdUpdateStyle,
		expectedHeadline: "Test User assigned story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41) to _nobody_ (#TES-41)",
	}, {
		file:             "testdata/webhook-issue-updated-assigned.json",
		expectedStyle:    mdUpdateStyle,
		expectedHeadline: "Test User assigned story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41) to Test User (#TES-41)",
	}, {
		file:             "testdata/webhook-issue-updated-attachments.json",
		expectedStyle:    mdUpdateStyle,
		expectedHeadline: "Test User attached [test.gif] to, removed attachments [test.json] from story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41) (#TES-41)",
	}, {
		file:             "testdata/webhook-issue-updated-edited.json",
		expectedStyle:    mdUpdateStyle,
		expectedHeadline: "Test User edited description of story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41) (#TES-41)",
		expectedText:     "Unit test description, not that long, a little longer now",
	}, {
		file:             "testdata/webhook-issue-updated-labels.json",
		expectedStyle:    mdUpdateStyle,
		expectedHeadline: "Test User added labels [sad] to, removed labels [bad] from story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41) (#TES-41)",
	}, {
		file:             "testdata/webhook-issue-updated-lowered-priority.json",
		expectedStyle:    mdUpdateStyle,
		expectedHeadline: "Test User lowered priority of story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41) to Low (#TES-41)",
	}, {
		file:             "testdata/webhook-issue-updated-raised-priority.json",
		expectedStyle:    mdUpdateStyle,
		expectedHeadline: "Test User raised priority of story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41) to High (#TES-41)",
	}, {
		file:             "testdata/webhook-issue-updated-rank.json",
		expectedStyle:    mdUpdateStyle,
		expectedHeadline: "Test User ranked higher story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41) (#TES-41)",
	}, {
		file:             "testdata/webhook-issue-updated-renamed.json",
		expectedStyle:    mdUpdateStyle,
		expectedHeadline: "Test User renamed story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41) to Unit test summary 1 (#TES-41)",
	}, {
		file:             "testdata/webhook-issue-updated-reopened.json",
		expectedStyle:    mdUpdateStyle,
		expectedHeadline: "Test User reopened story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41) (#TES-41)",
	}, {
		file:             "testdata/webhook-issue-updated-resolved.json",
		expectedStyle:    mdUpdateStyle,
		expectedHeadline: "Test User resolved story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41) (#TES-41)",
	}, {
		file:             "testdata/webhook-issue-updated-sprint.json",
		expectedStyle:    mdUpdateStyle,
		expectedHeadline: "Test User moved story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41) to Sprint 2 (#TES-41)",
	}, {
		file:             "testdata/webhook-issue-updated-started-working.json",
		expectedStyle:    mdUpdateStyle,
		expectedHeadline: "Test User started working on story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41) (#TES-41)",
	},
	} {
		t.Run(tc.file, func(t *testing.T) {
			f, err := os.Open(tc.file)
			require.NoError(t, err)
			defer f.Close()
			parsed, err := parse(f, func(w *JIRAWebhook) string {
				return w.mdIssueLongLink()
			})
			require.NoError(t, err)
			assert.Equal(t, tc.expectedStyle, parsed.style)
			assert.Equal(t, tc.expectedHeadline, parsed.headline)
			assert.Equal(t, tc.expectedDetails, parsed.details)
			assert.Equal(t, tc.expectedText, parsed.text)
		})
	}
}

func TestMarkdown(t *testing.T) {
	f, err := os.Open("testdata/webhook-issue-created.json")
	require.NoError(t, err)
	defer f.Close()
	parsed, err := parse(f, func(w *JIRAWebhook) string {
		return w.mdIssueLongLink()
	})
	require.NoError(t, err)
	m := newMarkdownMessage(parsed)

	assert.Equal(t, "## Test User created story [TES-41: Unit test summary](https://some-instance-test.atlassian.net/browse/TES-41) (#jira-new #TES-41)\nPriority: **High**, Reported by: **Test User**, Labels: test-label\nUnit test description, not that long\n", m)
}

func TestWebhookVariousErrorsForCoverage(t *testing.T) {
	assert.Equal(t, "", mdUser(nil))

	parsed := &parsed{
		JIRAWebhook: &JIRAWebhook{},
	}
	assert.Equal(t, "", parsed.mdIssueReportedBy())
	assert.Equal(t, "", parsed.mdIssueLabels())
	assert.Equal(t, "", parsed.jiraURL())
	parsed.fromChangeLog("link")
	assert.Equal(t, "", parsed.headline)
	assert.Equal(t, "", parsed.text)

	parsed.WebhookEvent = "something-else"
	assert.Equal(t, "", newMarkdownMessage(parsed))

	parsed.WebhookEvent = "jira:issue_updated"
	parsed.IssueEventTypeName = "something-else"
	assert.Equal(t, "", newMarkdownMessage(parsed))

	parsed.Issue.Fields.Assignee = &jira.User{
		DisplayName: "test",
	}
	assert.Equal(t, "Assigned to: **test**", parsed.mdIssueAssignedTo())
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

func TestWebhookJiraURL(t *testing.T) {
	var w JIRAWebhook
	w.Issue.Self = "http://localhost:8080/rest/api/2/issue/10006"
	assert.Equal(t, "http://localhost:8080", w.jiraURL())

	w.Issue.Self = "http://localhost:8080/foo/bar/rest/api/2/issue/10006"
	assert.Equal(t, "http://localhost:8080/foo/bar", w.jiraURL())
}
