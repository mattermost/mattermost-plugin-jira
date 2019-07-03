// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func getJiraTestData(filename string) *JiraWebhook {
	f, err := os.Open(filepath.Join("testdata", filename))

	if err != nil {
		panic(err)
	}

	jwh := &JiraWebhook{}
	err = json.NewDecoder(f).Decode(&jwh)
	if err != nil {
		panic(err)
	}

	return jwh
}

func TestEventEnums(t *testing.T) {
	for name, tc := range map[string]struct {
		TestWebhook   *JiraWebhook
		ExpectedEvent []string
	}{
		"issue created": {
			TestWebhook:   getJiraTestData("webhook-issue-created.json"),
			ExpectedEvent: []string{"event_created"},
		},
		"issue deleted": {
			TestWebhook:   getJiraTestData("webhook-issue-deleted.json"),
			ExpectedEvent: []string{"event_deleted"},
		},
		"comment created": {
			TestWebhook:   getJiraTestData("webhook-comment-created.json"),
			ExpectedEvent: []string{"event_created_comment"},
		},
		"comment deleted": {
			TestWebhook:   getJiraTestData("webhook-comment-deleted.json"),
			ExpectedEvent: []string{"event_deleted_comment"},
		},
		"comment updated": {
			TestWebhook:   getJiraTestData("webhook-comment-updated.json"),
			ExpectedEvent: []string{"event_updated_comment"},
		},
		"assigned nobody": {
			TestWebhook:   getJiraTestData("webhook-issue-updated-assigned-nobody.json"),
			ExpectedEvent: []string{"event_updated_assignee", "event_updated_all"},
		},
		"assigned on server": {
			TestWebhook:   getJiraTestData("webhook-issue-updated-assigned-on-server.json"),
			ExpectedEvent: []string{"event_updated_assignee", "event_updated_all"},
		},
		"assigned": {
			TestWebhook:   getJiraTestData("webhook-issue-updated-assigned.json"),
			ExpectedEvent: []string{"event_updated_assignee", "event_updated_all"},
		},
		"updated attachment": {
			TestWebhook:   getJiraTestData("webhook-issue-updated-attachments.json"),
			ExpectedEvent: []string{"event_updated_attachment", "event_updated_all"},
		},
		"updated description": {
			TestWebhook:   getJiraTestData("webhook-issue-updated-edited.json"),
			ExpectedEvent: []string{"event_updated_description", "event_updated_all"},
		},
		"updated labels": {
			TestWebhook:   getJiraTestData("webhook-issue-updated-labels.json"),
			ExpectedEvent: []string{"event_updated_labels", "event_updated_all"},
		},
		"lowered priority": {
			TestWebhook:   getJiraTestData("webhook-issue-updated-lowered-priority.json"),
			ExpectedEvent: []string{"event_updated_priority", "event_updated_all"},
		},
		"raised priority": {
			TestWebhook:   getJiraTestData("webhook-issue-updated-raised-priority.json"),
			ExpectedEvent: []string{"event_updated_priority", "event_updated_all"},
		},
		"updated rank": {
			TestWebhook:   getJiraTestData("webhook-issue-updated-rank.json"),
			ExpectedEvent: []string{"event_updated_rank", "event_updated_all"},
		},
		"updated summary": {
			TestWebhook:   getJiraTestData("webhook-issue-updated-renamed.json"),
			ExpectedEvent: []string{"event_updated_summary", "event_updated_all"},
		},
		"reopened issue": {
			TestWebhook:   getJiraTestData("webhook-issue-updated-reopened.json"),
			ExpectedEvent: []string{"event_updated_reopened", "event_updated_status", "event_updated_all"},
		},
		"resolved issue": {
			TestWebhook:   getJiraTestData("webhook-issue-updated-resolved.json"),
			ExpectedEvent: []string{"event_updated_resolved", "event_updated_status", "event_updated_all"},
		},
		"updated sprint": {
			TestWebhook:   getJiraTestData("webhook-issue-updated-sprint.json"),
			ExpectedEvent: []string{"event_updated_sprint", "event_updated_all"},
		},
		"updated status": {
			TestWebhook:   getJiraTestData("webhook-issue-updated-started-working.json"),
			ExpectedEvent: []string{"event_updated_status", "event_updated_all"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			actual := tc.TestWebhook.toEventEnums()
			assert.Equal(t, tc.ExpectedEvent, actual)
		})
	}
}
