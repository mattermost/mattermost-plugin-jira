// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventEnums(t *testing.T) {
	for name, tc := range map[string]struct {
		TestWebhook    *JiraWebhook
		ExpectedEvents []string
	}{
		"issue created": {
			TestWebhook:    getJiraTestData("webhook-issue-created.json"),
			ExpectedEvents: []string{"event_created"},
		},
		"issue deleted": {
			TestWebhook:    getJiraTestData("webhook-issue-deleted.json"),
			ExpectedEvents: []string{"event_deleted"},
		},
		"assigned nobody": {
			TestWebhook:    getJiraTestData("webhook-issue-updated-assigned-nobody.json"),
			ExpectedEvents: []string{"event_updated_assignee", "event_updated_all"},
		},
		"assigned on server": {
			TestWebhook:    getJiraTestData("webhook-issue-updated-assigned-on-server.json"),
			ExpectedEvents: []string{"event_updated_assignee", "event_updated_all"},
		},
		"assigned": {
			TestWebhook:    getJiraTestData("webhook-issue-updated-assigned.json"),
			ExpectedEvents: []string{"event_updated_assignee", "event_updated_all"},
		},
		"updated attachment": {
			TestWebhook:    getJiraTestData("webhook-issue-updated-attachments.json"),
			ExpectedEvents: []string{"event_updated_attachment", "event_updated_all"},
		},
		"updated description": {
			TestWebhook:    getJiraTestData("webhook-issue-updated-edited.json"),
			ExpectedEvents: []string{"event_updated_description", "event_updated_all"},
		},
		"updated labels": {
			TestWebhook:    getJiraTestData("webhook-issue-updated-labels.json"),
			ExpectedEvents: []string{"event_updated_labels", "event_updated_all"},
		},
		"lowered priority": {
			TestWebhook:    getJiraTestData("webhook-issue-updated-lowered-priority.json"),
			ExpectedEvents: []string{"event_updated_priority", "event_updated_all"},
		},
		"raised priority": {
			TestWebhook:    getJiraTestData("webhook-issue-updated-raised-priority.json"),
			ExpectedEvents: []string{"event_updated_priority", "event_updated_all"},
		},
		"updated rank": {
			TestWebhook:    getJiraTestData("webhook-issue-updated-rank.json"),
			ExpectedEvents: []string{"event_updated_rank", "event_updated_all"},
		},
		"updated summary": {
			TestWebhook:    getJiraTestData("webhook-issue-updated-renamed.json"),
			ExpectedEvents: []string{"event_updated_summary", "event_updated_all"},
		},
		"reopened issue": {
			TestWebhook:    getJiraTestData("webhook-issue-updated-reopened.json"),
			ExpectedEvents: []string{"event_updated_reopened", "event_updated_status", "event_updated_all"},
		},
		"resolved issue": {
			TestWebhook:    getJiraTestData("webhook-issue-updated-resolved.json"),
			ExpectedEvents: []string{"event_updated_resolved", "event_updated_status", "event_updated_all"},
		},
		"updated sprint": {
			TestWebhook:    getJiraTestData("webhook-issue-updated-sprint.json"),
			ExpectedEvents: []string{"event_updated_sprint", "event_updated_all"},
		},
		"updated status": {
			TestWebhook:    getJiraTestData("webhook-issue-updated-started-working.json"),
			ExpectedEvents: []string{"event_updated_status", "event_updated_all"},
		},
		"CLOUD comment created": {
			TestWebhook:    getJiraTestData("webhook-cloud-comment-created.json"),
			ExpectedEvents: []string{"event_created_comment"},
		},
		"CLOUD comment deleted": {
			TestWebhook:    getJiraTestData("webhook-cloud-comment-deleted.json"),
			ExpectedEvents: []string{"event_deleted_comment"},
		},
		"CLOUD comment updated": {
			TestWebhook:    getJiraTestData("webhook-cloud-comment-updated.json"),
			ExpectedEvents: []string{"event_updated_comment"},
		},
		"SERVER comment created": {
			TestWebhook:    getJiraTestData("webhook-server-issue-updated-commented-1.json"),
			ExpectedEvents: []string{"event_created_comment"},
		},
		"SERVER comment deleted": {
			TestWebhook:    getJiraTestData("webhook-server-issue-updated-comment-deleted.json"),
			ExpectedEvents: []string{"event_deleted_comment"},
		},
		"SERVER comment updated": {
			TestWebhook:    getJiraTestData("webhook-server-issue-updated-comment-edited.json"),
			ExpectedEvents: []string{"event_updated_comment"},
		},
		"SERVER comment created notify": {
			TestWebhook:    getJiraTestData("webhook-server-issue-updated-commented-2.json"),
			ExpectedEvents: []string{"event_created_comment"},
		},
		"SERVER comment created ignored": {
			TestWebhook:    getJiraTestData("webhook-server-comment-created.json"),
			ExpectedEvents: []string{},
		},
	} {
		t.Run(name, func(t *testing.T) {
			actual := tc.TestWebhook.toEventEnums()
			expected := tc.ExpectedEvents
			assert.Equal(t, len(expected), len(actual))
			for _, enum := range expected {
				_, contains := actual[enum]
				if !contains {
					t.Fatalf("Event type not present: %s", enum)
				}
			}
		})
	}
}
