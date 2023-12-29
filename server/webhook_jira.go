// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"fmt"
	"strings"

	"github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

type JiraWebhook struct {
	WebhookEvent string       `json:"webhookEvent,omitempty"`
	Issue        jira.Issue   `json:"issue,omitempty"`
	User         jira.User    `json:"user,omitempty"`
	Comment      jira.Comment `json:"comment,omitempty"`
	ChangeLog    struct {
		Items []struct {
			From       string
			FromString string
			To         string
			ToString   string
			Field      string
			FieldID    string
			FieldType  string `json:"fieldtype"`
		}
	} `json:"changelog,omitempty"`
	IssueEventTypeName string `json:"issue_event_type_name"`
}

func (jwh *JiraWebhook) expandIssue(p *Plugin, instanceID types.ID) error {
	instance, err := p.instanceStore.LoadInstance(instanceID)
	if err != nil {
		return err
	}

	if !instance.Common().IsCloudInstance() {
		return nil
	}

	// TODO: The data sent for "Status" field is invalid in case of issue created event, so we are fetching it again here. This can be updated when the issue is fixed from Jira side.
	// Jira Cloud comment event. We need to fetch issue data because it is not expanded in webhook payload.
	isCommentEvent := jwh.WebhookEvent == commentCreated || jwh.WebhookEvent == commentUpdated || jwh.WebhookEvent == commentDeleted || jwh.WebhookEvent == issueCreated
	if isCommentEvent {
		if _, ok := instance.(*cloudInstance); ok {
			issue, err := p.getIssueDataForCloudWebhook(instance, jwh.Issue.ID)
			if err != nil {
				return err
			}

			jwh.Issue = *issue
		} else if instance, ok := instance.(*cloudOAuthInstance); ok {
			accountID := jwh.Comment.Author.AccountID
			if jwh.WebhookEvent == issueCreated {
				accountID = jwh.Issue.Fields.Creator.AccountID
			}

			mmUserID, err := p.userStore.LoadMattermostUserID(instanceID, accountID)
			if err != nil {
				// User is not connected, so we try to fall back to JWT bot
				if instance.JWTInstance == nil {
					return errors.Wrap(err, "Cannot create subscription posts for this comment as the Jira comment author is not connected to Mattermost.")
				}

				// Fetch issue details with bot JWT bot
				var issue *jira.Issue
				issue, err = p.getIssueDataForCloudWebhook(instance.JWTInstance, jwh.Issue.ID)
				if err != nil {
					return errors.Wrap(err, "failed to getIssueDataForCloudWebhook using bot account")
				}

				jwh.Issue = *issue
				return nil
			}

			conn, err := p.userStore.LoadConnection(instance.GetID(), mmUserID)
			if err != nil {
				return err
			}

			client, err := instance.GetClient(conn)
			if err != nil {
				return err
			}

			issue, err := client.GetIssue(jwh.Issue.ID, nil)
			if err != nil {
				return err
			}

			jwh.Issue = *issue
		}
	}

	return nil
}

func (jwh *JiraWebhook) mdJiraLink(title, suffix string) string {
	// Use Self URL only to extract the full hostname from it
	pos := strings.LastIndex(jwh.Issue.Self, "/rest/api")
	if pos < 0 {
		return ""
	}
	// TODO: For Jira OAuth, the Self URL is sent as https://api.atlassian.com/ instead of the Jira Instance URL - to check this and handle accordingly
	return fmt.Sprintf("[%s](%s%s)", title, jwh.Issue.Self[:pos], suffix)
}

func (jwh *JiraWebhook) mdIssueDescription() string {
	return truncate(jwh.Issue.Fields.Description, 3000)
}

func (jwh *JiraWebhook) mdIssueSummary() string {
	return truncate(jwh.Issue.Fields.Summary, 80)
}

func (jwh *JiraWebhook) mdIssueAssignee() string {
	if jwh.Issue.Fields.Assignee == nil {
		return Nobody
	}
	return mdUser(jwh.Issue.Fields.Assignee)
}

func (jwh *JiraWebhook) mdSummaryLink() string {
	return jwh.mdIssueType() + " " + jwh.mdJiraLink(jwh.mdIssueSummary(), "/browse/"+jwh.Issue.Key)
}

func (jwh *JiraWebhook) mdKeySummaryLink() string {
	return jwh.mdIssueType() + " " + jwh.mdJiraLink(
		jwh.Issue.Key+": "+jwh.mdIssueSummary(),
		"/browse/"+jwh.Issue.Key)
}

func (jwh *JiraWebhook) mdKeyLink() string {
	return jwh.mdIssueType() + " " + jwh.mdJiraLink(jwh.Issue.Key, "/browse/"+jwh.Issue.Key)
}

func (jwh *JiraWebhook) mdUser() string {
	return mdUser(&jwh.User)
}

func (jwh *JiraWebhook) mdIssueType() string {
	return strings.ToLower(jwh.Issue.Fields.Type.Name)
}

func mdAddRemove(from, to, add, remove string) string {
	added := mdDiff(from, to)
	removed := mdDiff(to, from)
	s := ""
	if added != "" {
		s += fmt.Sprintf("%v [%v] to", add, added)
	}
	if removed != "" {
		if added != "" {
			s += ", "
		}
		s += fmt.Sprintf("%v [%v] from", remove, removed)
	}
	return s
}

func mdDiff(from, to string) string {
	fromStrings := strings.Split(from, " ")
	toStrings := strings.Split(to, " ")
	fromMap := map[string]bool{}
	for _, s := range fromStrings {
		fromMap[s] = true
	}
	toMap := map[string]bool{}
	for _, s := range toStrings {
		toMap[s] = true
	}
	added := []string{}
	for s := range toMap {
		if !fromMap[s] {
			added = append(added, s)
		}
	}

	return strings.Join(added, ",")
}

func mdUser(user *jira.User) string {
	if user == nil {
		return ""
	}
	return user.DisplayName
}

func truncate(s string, max int) string {
	if len(s) <= max || max < 0 {
		return s
	}
	if max > 3 {
		return s[:max-3] + "..."
	}
	return s[:max]
}
