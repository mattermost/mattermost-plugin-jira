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

	for _, field := range parsed.fields {
		a.Fields = append(a.Fields, &model.SlackAttachmentField{
			Title: field.name,
			Value: field.value,
			Short: true,
		})
	}

	return a
}
