// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"io"

	"github.com/mattermost/mattermost-server/model"
)

func AsSlackAttachment(in io.Reader, n notifier, ji JIRAInstance) (func(post *model.Post), error) {
	parsed, err := parse(in, nil)
	if err != nil {
		return nil, err
	}

	a := newSlackAttachment(parsed)

	if parsed.text != "" && n != nil {
		n.notify(ji, parsed, parsed.text)
	}

	// Return a function that adds to a post as a SlackAttachment
	return func(post *model.Post) {
		model.ParseSlackAttachment(post, []*model.SlackAttachment{a})
	}, nil
}

func newSlackAttachment(parsed *parsed) *model.SlackAttachment {
	if parsed.headline == "" {
		return nil
	}

	a := &model.SlackAttachment{
		Color:    "#95b7d0",
		Fallback: parsed.headline,
		Pretext:  parsed.headline,
		Text:     parsed.text,
	}

	text := parsed.mdIssueLongLink() + "\n"
	if parsed.text != "" {
		text += "\n"
		text += parsed.text + "\n"
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

	a.Text = text
	a.Fields = fields
	return a
}
