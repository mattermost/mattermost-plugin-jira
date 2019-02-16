// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebhookMarkdown(t *testing.T) {
	for _, tc := range []struct {
		file     string
		expected string
	}{{
		file:     "testdata/webhook-comment-created.json",
		expected: "###### Test User commented on [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41) (#TES-41)\nAdded a comment\n",
	}, {
		file:     "testdata/webhook-comment-deleted.json",
		expected: "###### Test User removed a comment from [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41) (#TES-41)\n\n",
	}, {
		file:     "testdata/webhook-comment-updated.json",
		expected: "###### Test User edited a comment in [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41) (#TES-41)\nAdded a comment, then edited it\n",
	}, {
		file:     "testdata/webhook-issue-created.json",
		expected: "## Test User created a story [TES-41: Unit test summary](https://some-instance-test.atlassian.net/browse/TES-41)\nPriority: **High**, Reported by: **Test User**, Labels: test-label, (#jira-new #TES-41)\n\n```\nUnit test description, not that long\n```",
	}, {
		file:     "testdata/webhook-issue-updated-assigned-nobody.json",
		expected: "###### Test User assigned [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41) to _nobody_ (#TES-41)\n\n",
	}, {
		file:     "testdata/webhook-issue-updated-assigned.json",
		expected: "###### Test User assigned [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41) to Test User (#TES-41)\n\n",
	}, {
		file:     "testdata/webhook-issue-updated-edited.json",
		expected: "###### Test User edited description of [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41) (#TES-41)\n```\nUnit test description, not that long, a little longer now\n```\n",
	}, {
		file:     "testdata/webhook-issue-updated-labels.json",
		expected: "###### Test User added labels [sad] to, removed labels [bad] from [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41) (#TES-41)\n\n",
	}, {
		file:     "testdata/webhook-issue-updated-lowered-priority.json",
		expected: "###### Test User lowered priority of [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41) to Low (#TES-41)\n\n",
	}, {
		file:     "testdata/webhook-issue-updated-raised-priority.json",
		expected: "###### Test User raised priority of [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41) to High (#TES-41)\n\n",
	}, {
		file:     "testdata/webhook-issue-updated-renamed.json",
		expected: "###### Test User renamed [TES-41](https://some-instance-test.atlassian.net/browse/TES-41) to Unit test summary 1 (#TES-41)\n\n",
	}, {
		file:     "testdata/webhook-issue-updated-reopened.json",
		expected: "###### Test User reopened [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41) (#TES-41)\n\n",
	}, {
		file:     "testdata/webhook-issue-updated-resolved.json",
		expected: "###### Test User resolved [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41) (#TES-41)\n\n",
	}, {
		file:     "testdata/webhook-issue-updated-started-working.json",
		expected: "###### Test User started working on [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41) (#TES-41)\n\n",
	},
	} {
		t.Run(tc.file, func(t *testing.T) {
			f, err := os.Open(tc.file)
			require.NoError(t, err)
			defer f.Close()
			var w Webhook
			require.NoError(t, json.NewDecoder(f).Decode(&w))
			assert.Equal(t, tc.expected, w.Markdown())
		})
	}
}

func TestWebhookJiraURL(t *testing.T) {
	var w Webhook
	w.Issue.Self = "http://localhost:8080/rest/api/2/issue/10006"
	assert.Equal(t, "http://localhost:8080", jiraURL(&w))

	w.Issue.Self = "http://localhost:8080/foo/bar/rest/api/2/issue/10006"
	assert.Equal(t, "http://localhost:8080/foo/bar", jiraURL(&w))
}
