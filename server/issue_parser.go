// See License for license information.
// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.

package main

import (
	"fmt"
	"regexp"
	"strings"

	jira "github.com/andygrunwald/go-jira"

	"github.com/mattermost/mattermost-server/model"
)

var jiraLinkWithTextRegex = regexp.MustCompile(`\[([^\[]+)\|([^\]]+)\]`)

func parseJiraLinksToMarkdown(text string) string {
	return jiraLinkWithTextRegex.ReplaceAllString(text, "[${1}](${2})")
}

func mdKeySummaryLink(issue *jira.Issue) string {
	pos := strings.LastIndex(issue.Self, "/rest/api")
	if pos < 0 {
		return ""
	}
	return fmt.Sprintf("[%s](%s%s)", issue.Key+": "+issue.Fields.Summary, issue.Self[:pos], "/browse/"+issue.Key)
}

func parseIssue(issue *jira.Issue) []*model.SlackAttachment {
	text := mdKeySummaryLink(issue)
	desc := truncate(issue.Fields.Description, 3000)
	desc = parseJiraLinksToMarkdown(desc)
	if desc != "" {
		text += "\n\n" + desc + "\n"
	}

	var fields []*model.SlackAttachmentField
	if issue.Fields.Assignee != nil {
		fields = append(fields, &model.SlackAttachmentField{
			Title: "Assignee",
			Value: issue.Fields.Assignee.DisplayName,
			Short: true,
		})
	}
	if issue.Fields.Priority != nil {
		fields = append(fields, &model.SlackAttachmentField{
			Title: "Priority",
			Value: issue.Fields.Priority.Name,
			Short: true,
		})
	}
	return []*model.SlackAttachment{
		{
			// TODO is this supposed to be themed?
			Color:  "#95b7d0",
			Text:   text,
			Fields: fields,
		},
	}
}
