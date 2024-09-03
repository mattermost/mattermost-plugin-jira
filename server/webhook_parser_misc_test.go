// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"io"
	"os"
	"strings"
	"testing"

	jira "github.com/andygrunwald/go-jira"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarkdown(t *testing.T) {
	f, err := os.Open("testdata/webhook-issue-created.json")
	require.NoError(t, err)
	defer f.Close()
	bb, err := io.ReadAll(f)
	require.Nil(t, err)
	wh, err := ParseWebhook(bb)
	require.NoError(t, err)
	w := wh.(*webhook)
	require.NotNil(t, w)
	require.Equal(t,
		"Test User **created** story [TES-41: Unit test summary](https://some-instance-test.atlassian.net/browse/TES-41)",
		w.headline)
}

func TestEventTypeFormat(t *testing.T) {
	for _, value := range map[string]struct {
		filename string
		expected string
	}{
		"issue created format":                        {"testdata/webhook-issue-created.json", "Test User **created** story"},
		"issue updated assigned format":               {"testdata/webhook-issue-updated-assigned.json", "Test User **assigned** Test User to story"},
		"issue updated reopened format":               {"testdata/webhook-issue-updated-reopened.json", "Test User **updated** story"},
		"issue updated reopened format one changelog": {"testdata/webhook-issue-updated-reopened-one-changelog.json", "Test User **reopened** story"},
		"issue updated resolved format":               {"testdata/webhook-issue-updated-resolved.json", "Test User **updated** story"},
		"issue updated resolved format one changelog": {"testdata/webhook-issue-updated-resolved-one-changelog.json", "Test User **resolved** story"},
		"issue deleted":                               {"testdata/webhook-issue-deleted.json", "Test User **deleted** task"},
		"issue updated commented created":             {"testdata/webhook-server-issue-updated-commented-3.json", "Test User **commented** on improvement"},
		"issue updated comment edited":                {"testdata/webhook-server-issue-updated-comment-edited.json", "Lev Brouk **edited comment** in story"},
		"issue updated comment deleted":               {"testdata/webhook-server-issue-updated-comment-deleted.json", "Lev Brouk **deleted comment** in story"},
	} {
		f, err := os.Open(value.filename)
		require.NoError(t, err)
		defer f.Close()
		bb, err := io.ReadAll(f)
		require.Nil(t, err)
		wh, err := ParseWebhook(bb)
		require.NoError(t, err)
		w := wh.(*webhook)
		require.NotNil(t, w)
		require.Contains(t, w.headline, value.expected)
	}
}

func TestNotificationsFormat(t *testing.T) {
	for _, value := range map[string]struct {
		filename string
		expected string
	}{
		"issue updated commented created": {"testdata/webhook-server-issue-updated-commented-3.json", "Test User **mentioned** you in a new comment on improvement"},
	} {
		f, err := os.Open(value.filename)
		require.NoError(t, err)
		defer f.Close()
		bb, err := io.ReadAll(f)
		require.Nil(t, err)
		wh, err := ParseWebhook(bb)
		require.NoError(t, err)
		w := wh.(*webhook)
		require.NotNil(t, w)
		require.NotNil(t, w.notifications)
		require.Contains(t, w.notifications[0].message, value.expected)
	}
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
	assert.Equal(t, Nobody, wh.mdIssueAssignee())
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

func TestWebhookQuotedComment(t *testing.T) {
	for _, value := range []string{
		"testdata/webhook-server-issue-updated-commented-3.json",
		"testdata/webhook-server-issue-updated-comment-edited.json",
	} {
		f, err := os.Open(value)
		require.NoError(t, err)
		defer f.Close()
		bb, err := io.ReadAll(f)
		require.Nil(t, err)
		wh, err := ParseWebhook(bb)
		require.NoError(t, err)
		w := wh.(*webhook)
		require.NotNil(t, w)
		assert.True(t, strings.HasPrefix(w.text, ">"))
	}
}

func TestPreProcessText(t *testing.T) {
	tests := map[string]struct {
		input          string
		expectedOutput string
	}{
		"BOLD formatting": {
			input:          "*BOLD*",
			expectedOutput: "**BOLD**",
		},
		"STRIKETHROUGH formatting": {
			input:          "-STRIKETHROUGH-",
			expectedOutput: "~~STRIKETHROUGH~~",
		},
		"Colored text formatting": {
			input:          "{color:#ff5630}RED{color} {color:#4c9aff}BLUE{color} {color:#36b37e}GREEN{color}",
			expectedOutput: "RED BLUE GREEN",
		},
		"Numbered list with mixed content formatting": {
			input: `# NUMBERED LIST ROW 1
# NUMBERED LIST ROW 2
non-numbered list text
# NUMBERED LIST ROW 1`,
			expectedOutput: `1. NUMBERED LIST ROW 1
2. NUMBERED LIST ROW 2
non-numbered list text
1. NUMBERED LIST ROW 1`,
		},
		"Code block formatting": {
			input:          "{code:go}fruit := \"APPLE\"{code}",
			expectedOutput: "`fruit := \"APPLE\"`",
		},
		"Bullet list formatting": {
			input: `* BULLET LIST ROW 1
* BULLET LIST ROW 2`,
			expectedOutput: `* BULLET LIST ROW 1
* BULLET LIST ROW 2`,
		},
		"Heading formatting": {
			input: `h1. HEADING 1
h2. HEADING 2
h3. HEADING 3
h4. HEADING 4
h5. HEADING 5
h6. HEADING 6`,
			expectedOutput: `# HEADING 1
## HEADING 2
### HEADING 3
#### HEADING 4
##### HEADING 5
###### HEADING 6`,
		},
		"Link formatting with text": {
			input:          "[www.googlesd.com|http://www.googlesd.com]",
			expectedOutput: "[www.googlesd.com](http://www.googlesd.com)",
		},
		"Link formatting with smart-link": {
			input:          "[http://www.google.com|http://www.google.com|smart-link]",
			expectedOutput: "[http://www.google.com](http://www.google.com)",
		},
		"Link formatting with title": {
			input:          "[google|http://www.google.com]",
			expectedOutput: "[google](http://www.google.com)",
		},
		"Quote formatting": {
			input:          "{quote}This is a quote{quote}",
			expectedOutput: "> This is a quote",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			actualOutput := preProcessText(tc.input)
			assert.Equal(t, tc.expectedOutput, actualOutput)
		})
	}
}
