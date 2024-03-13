// See License for license information.
// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.

package main

import (
	"fmt"
	"testing"

	jira "github.com/andygrunwald/go-jira"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAsSlackAttachment(t *testing.T) {
	instance := testInstance2
	client := testClient{}

	for name, tc := range map[string]struct {
		issue              *jira.Issue
		showActions        bool
		expectedAttachment *model.SlackAttachment
	}{
		"minimum data": {
			issue: &jira.Issue{
				Key: "MM-57208",
				Fields: &jira.IssueFields{
					Summary: "",
					Status: &jira.Status{
						Name: "Open",
					},
				},
			},
			expectedAttachment: &model.SlackAttachment{
				Text:  "[MM-57208:  (Open)](jiraurl2/browse/MM-57208)",
				Color: "#95b7d0",
			},
		},
		"with summary": {
			issue: &jira.Issue{
				Key: "MM-57208",
				Fields: &jira.IssueFields{
					Summary: "A Summary",
					Status: &jira.Status{
						Name: "Open",
					},
				},
			},
			expectedAttachment: &model.SlackAttachment{
				Text:  "[MM-57208: A Summary (Open)](jiraurl2/browse/MM-57208)",
				Color: "#95b7d0",
			},
		},
		"with Assignee, Priority and Reporter": {
			issue: &jira.Issue{
				Key: "MM-57208",
				Fields: &jira.IssueFields{
					Priority: &jira.Priority{
						Name: "Very important",
					},
					Assignee: &jira.User{
						DisplayName: "The Assignee",
					},
					Description: "A Description",
					Summary:     "A Summary",
					Reporter: &jira.User{
						Name: "The Reporter",
						AvatarUrls: jira.AvatarUrls{
							One6X16: "https://example.org/icon",
						},
					},
					Status: &jira.Status{
						Name: "Open",
					},
				},
			},
			expectedAttachment: &model.SlackAttachment{
				Text:  "[MM-57208: A Summary (Open)](jiraurl2/browse/MM-57208)\n\nA Description\n",
				Color: "#95b7d0",
				Fields: []*model.SlackAttachmentField{
					{
						Title: "Assignee",
						Value: "The Assignee",
						Short: true,
					},
					{
						Title: "Priority",
						Value: "Very important",
						Short: true,
					},
					{
						Title: "Reporter",
						Value: "![avatar](https://example.org/icon =30x30) The Reporter",
						Short: true,
					},
				},
			},
		},
		"with actions": {
			issue: &jira.Issue{
				ID:  "some ID",
				Key: "MM-57208",
				Fields: &jira.IssueFields{
					Summary: "",
					Status: &jira.Status{
						Name: "Open",
					},
				},
			},
			showActions: true,
			expectedAttachment: &model.SlackAttachment{
				Text:  "[MM-57208:  (Open)](jiraurl2/browse/MM-57208)",
				Color: "#95b7d0",
				Actions: []*model.PostAction{
					{
						Type: "select",
						Name: "Transition issue",
						Options: []*model.PostActionOptions{
							{
								Value: "To Do",
							},
							{
								Value: "In Progress",
							},
							{
								Value: "In Testing",
							},
						},
						Integration: &model.PostActionIntegration{
							URL: fmt.Sprintf("/plugins/%s/api/v2/transition", manifest.Id),
							Context: map[string]any{
								"issue_key":   "some ID",
								"instance_id": testInstance2.GetID().String(),
							},
						},
					},
					{
						Type: "button",
						Name: "Share publicly",
						Integration: &model.PostActionIntegration{
							URL: fmt.Sprintf("/plugins/%s/api/v2/share-issue-publicly", manifest.Id),
							Context: map[string]any{
								"issue_key":   "some ID",
								"instance_id": testInstance2.GetID().String(),
							},
						},
					},
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			attachments, err := asSlackAttachment(instance, client, tc.issue, tc.showActions)
			assert.NoError(t, err)
			require.Len(t, attachments, 1)
			assert.Equal(t, tc.expectedAttachment, attachments[0])
		})
	}
}
