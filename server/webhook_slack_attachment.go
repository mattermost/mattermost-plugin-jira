// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"github.com/mattermost/mattermost-server/model"
)

func newSlackAttachment(parsed *parsedJIRAWebhook) *model.SlackAttachment {
	if parsed.headline == "" {
		return nil
	}

	a := &model.SlackAttachment{
		Color:    "#95b7d0",
		Fallback: parsed.headline,
		Pretext:  parsed.headline,
		Text:     parsed.text,
	}

	var fields []*model.SlackAttachmentField
	if parsed.WebhookEvent == "jira:issue_created" {

		if parsed.Issue.Fields.Assignee != nil {
			fields = append(fields, &model.SlackAttachmentField{
				Title: "Assignee",
				Value: parsed.Issue.Fields.Assignee.DisplayName,
				Short: true,
			})
		}
		if parsed.Issue.Fields.Priority != nil {
			fields = append(fields, &model.SlackAttachmentField{
				Title: "Priority",
				Value: parsed.Issue.Fields.Priority.Name,
				Short: true,
			})
		}
	}

	a.Fields = fields
	return a
}
