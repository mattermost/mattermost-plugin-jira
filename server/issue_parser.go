// See License for license information.
// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.

package main

import (
	"fmt"
	"regexp"

	jira "github.com/andygrunwald/go-jira"

	"github.com/mattermost/mattermost-server/v6/model"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

var jiraLinkWithTextRegex = regexp.MustCompile(`\[([^\[]+)\|([^\]]+)\]`)

func parseJiraLinksToMarkdown(text string) string {
	return jiraLinkWithTextRegex.ReplaceAllString(text, "[${1}](${2})")
}

func mdKeySummaryLink(issue *jira.Issue, instance Instance) string {
	return fmt.Sprintf("[%s: %s (%s)](%s%s)", issue.Key, issue.Fields.Summary, issue.Fields.Status.Name, instance.GetJiraBaseURL(), "/browse/"+issue.Key)
}

func reporterSummary(issue *jira.Issue) string {
	avatarLink := fmt.Sprintf("![avatar](%s =30x30)", issue.Fields.Reporter.AvatarUrls.One6X16)
	reporterSummary := avatarLink + " " + issue.Fields.Reporter.Name
	return reporterSummary
}

func getActions(instanceID types.ID, client Client, issue *jira.Issue) ([]*model.PostAction, error) {
	var actions []*model.PostAction

	ctx := map[string]interface{}{
		"issue_key":   issue.ID,
		"instance_id": instanceID.String(),
	}

	integration := &model.PostActionIntegration{
		URL:     fmt.Sprintf("/plugins/%s%s%s", Manifest.Id, routeAPI, routeIssueTransition),
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

	actions = append(actions, &model.PostAction{
		Name: "Share publicly",
		Type: "button",
		Integration: &model.PostActionIntegration{
			URL:     fmt.Sprintf("/plugins/%s%s%s", Manifest.Id, routeAPI, routeSharePublicly),
			Context: ctx,
		},
	})

	return actions, nil
}

func asSlackAttachment(instance Instance, client Client, issue *jira.Issue, showActions bool) ([]*model.SlackAttachment, error) {
	text := mdKeySummaryLink(issue, instance)
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

	var actions []*model.PostAction
	var err error
	if showActions {
		actions, err = getActions(instance.GetID(), client, issue)
		if err != nil {
			return []*model.SlackAttachment{}, err
		}
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
