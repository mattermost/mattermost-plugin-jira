// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"os"
	"testing"

	"github.com/mattermost/mattermost-server/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlackAttachment(t *testing.T) {
	f, err := os.Open("testdata/webhook-issue-created.json")
	require.NoError(t, err)
	defer f.Close()
	parsed, err := parse(f, nil)
	require.NoError(t, err)
	a := newSlackAttachment(parsed)

	assert.Equal(t, "Test User created story [TES-41](https://some-instance-test.atlassian.net/browse/TES-41) (#jira-new #TES-41)", a.Fallback)
	assert.Equal(t, "#95b7d0", a.Color)
	assert.Equal(t, "Test User created story [TES-41](https://some-instance-test.atlassian.net/browse/TES-41) (#jira-new #TES-41)", a.Pretext)
	assert.Equal(t, "[TES-41: Unit test summary](https://some-instance-test.atlassian.net/browse/TES-41)\n\nUnit test description, not that long\n", a.Text)
	assert.Equal(t, 1, len(a.Fields))
	assert.Equal(t, &model.SlackAttachmentField{Title: "Priority", Value: "High", Short: true}, a.Fields[0])
}

func TestSlackAttachmentForCoverage(t *testing.T) {
	parsed := &parsed{
		JIRAWebhook: &JIRAWebhook{},
	}
	parsed.WebhookEvent = "something-else"
	assert.Nil(t, newSlackAttachment(parsed))

	parsed.WebhookEvent = "jira:issue_updated"
	parsed.IssueEventTypeName = "something-else"
	assert.Nil(t, newSlackAttachment(parsed))
}
