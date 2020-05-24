// See License for license information.
// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.

package main

import (
	"fmt"
	"regexp"
	"strings"

	jira "github.com/andygrunwald/go-jira"

	"github.com/mattermost/mattermost-server/v5/model"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

var jiraLinkWithTextRegex = regexp.MustCompile(`\[([^\[]+)\|([^\]]+)\]`)

func parseJiraLinksToMarkdown(text string) string {
	return jiraLinkWithTextRegex.ReplaceAllString(text, "[${1}](${2})")
}

func mdKeySummaryLink(issue *jira.Issue) string {
	// Use Self URL only to extract the full hostname from it
	pos := strings.LastIndex(issue.Self, "/rest/api")
	if pos < 0 {
		return ""
	}
	return fmt.Sprintf("[%s](%s%s)", issue.Key+": "+issue.Fields.Summary+" ("+issue.Fields.Status.Name+")", issue.Self[:pos], "/browse/"+issue.Key)
}

func reporterSummary(issue *jira.Issue) string {
	avatarLink := fmt.Sprintf("![avatar](%s =30x30)", issue.Fields.Reporter.AvatarUrls.One6X16)
	reporterSummary := avatarLink + " " + issue.Fields.Reporter.Name
	return reporterSummary
}

func getTransitionActions(instanceID types.ID, client Client, issue *jira.Issue) ([]*model.PostAction, error) {
	var actions []*model.PostAction

	ctx := map[string]interface{}{
		"issueKey": issue.ID,
	}

	integration := &model.PostActionIntegration{
		URL:     fmt.Sprintf("/plugins/%s%s", manifest.Id, instancePath(routeIssueTransition, instanceID)),
		Context: ctx,
	}

	var options []*model.PostActionOptions

	transitions, err := client.GetTransitions(issue.Key)
	if err != nil {
		return actions, err
	}

	// Remove current issue status from possible transitions
	for _, transition := range transitions {
		if transition.Name != issue.Fields.Status.Name {
			options = append(options, &model.PostActionOptions{
				Text:  transition.Name,
				Value: transition.To.Name,
			})
		}
	}

	actions = append(actions, &model.PostAction{
		Name:        "Transition issue",
		Type:        "select",
		Options:     options,
		Integration: integration,
	})

	return actions, nil
}

func asSlackAttachment(instanceID types.ID, client Client, issue *jira.Issue) ([]*model.SlackAttachment, error) {
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

	fields = append(fields, &model.SlackAttachmentField{
		Title: "Reporter",
		Value: reporterSummary(issue),
		Short: true,
	})

	actions, err := getTransitionActions(instanceID, client, issue)
	if err != nil {
		return []*model.SlackAttachment{}, err
	}

	return []*model.SlackAttachment{
		{
			// TODO is this supposed to be themed?
			Color:   "#95b7d0",
			Text:    text,
			Fields:  fields,
			Actions: actions,
		},
	}, nil
}
