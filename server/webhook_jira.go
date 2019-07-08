// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"fmt"
	"strings"

	"github.com/andygrunwald/go-jira"
)

type JiraWebhook struct {
	WebhookEvent string       `json:"webhookEvent,omitempty"`
	Issue        jira.Issue   `json:"issue,omitempty"`
	User         jira.User    `json:"user,omitempty"`
	Comment      jira.Comment `json:"comment,omitempty"`
	// TODO figure out why jira.Changelog didn't work
	ChangeLog struct {
		Items []struct {
			From       string
			FromString string
			To         string
			ToString   string
			Field      string
		}
	} `json:"changelog,omitempty"`
	IssueEventTypeName string `json:"issue_event_type_name"`
}

func (jwh *JiraWebhook) mdJiraLink(title, suffix string) string {
	pos := strings.LastIndex(jwh.Issue.Self, "/rest/api")
	if pos < 0 {
		return ""
	}
	return fmt.Sprintf("[%s](%s%s)", title, jwh.Issue.Self[:pos], suffix)
}

func (jwh *JiraWebhook) mdIssueDescription() string {
	return truncate(jwh.Issue.Fields.Description, 3000)
}

func (jwh *JiraWebhook) mdIssueSummary() string {
	return truncate(jwh.Issue.Fields.Summary, 80)
}

func (w *JiraWebhook) mdIssueAssignee() string {
	if w.Issue.Fields.Assignee == nil {
		return "_nobody_"
	}
	return mdUser(w.Issue.Fields.Assignee)
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
