// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"fmt"
	"strings"

	"github.com/andygrunwald/go-jira"
)

const (
	eventCreatedStr            = "event_created"
	eventCreatedCommentStr     = "event_created_comment"
	eventDeletedStr            = "event_deleted"
	eventDeletedCommentStr     = "event_deleted_comment"
	eventDeletedUnresolvedStr  = "event_deleted_unresolved" // unused
	eventUpdatedAllStr         = "event_updated_all"
	eventUpdatedAssigneeStr    = "event_updated_assignee"
	eventUpdatedAttachmentStr  = "event_updated_attachment"
	eventUpdatedCommentStr     = "event_updated_comment"
	eventUpdatedDescriptionStr = "event_updated_description"
	eventUpdatedLabelsStr      = "event_updated_labels"
	eventUpdatedPriorityStr    = "event_updated_priority"
	eventUpdatedRankStr        = "event_updated_rank"
	eventUpdatedReopenedStr    = "event_updated_reopened"
	eventUpdatedResolvedStr    = "event_updated_resolved"
	eventUpdatedSprintStr      = "event_updated_sprint"
	eventUpdatedStatusStr      = "event_updated_status"
	eventUpdatedSummaryStr     = "event_updated_summary"
)

var webhookEventToEnumMap = map[string]string{
	"comment_created":    eventCreatedCommentStr,
	"comment_deleted":    eventDeletedCommentStr,
	"comment_updated":    eventUpdatedCommentStr,
	"jira:issue_created": eventCreatedStr,
	"jira:issue_deleted": eventDeletedStr,
}

var eventTypeNameToEnumMap = map[string]string{
	"issue_created":         eventCreatedStr,
	"issue_commented":       eventCreatedCommentStr,
	"issue_comment_deleted": eventDeletedCommentStr,
	"issue_comment_edited":  eventUpdatedCommentStr,
}

var updateFieldToEnumMap = map[string]string{
	"assignee":    eventUpdatedAssigneeStr,
	"Attachment":  eventUpdatedAttachmentStr,
	"description": eventUpdatedDescriptionStr,
	"labels":      eventUpdatedLabelsStr,
	"priority":    eventUpdatedPriorityStr,
	"Rank":        eventUpdatedRankStr,
	"Sprint":      eventUpdatedSprintStr,
	"summary":     eventUpdatedSummaryStr,
	"status":      eventUpdatedStatusStr,
}

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

// toEventEnums converts a JiraWebhook struct into a slice of internal event identifiers
func (jwh *JiraWebhook) toEventEnums() map[string]bool {
	we := jwh.WebhookEvent
	etn := jwh.IssueEventTypeName
	result := map[string]bool{}

	if jwh.Issue.Fields == nil {
		return result
	}

	switch etn {
	case "issue_updated", "issue_generic":
		result[eventUpdatedAllStr] = true
		for _, item := range jwh.ChangeLog.Items {
			field := item.Field
			if updateFieldToEnumMap[field] != "" {
				// Typical field
				result[updateFieldToEnumMap[field]] = true
			} else if field == "resolution" {
				// Resolution
				if item.ToString == "Done" {
					result[eventUpdatedResolvedStr] = true
				} else {
					result[eventUpdatedReopenedStr] = true
				}
			} else {
				// Custom field
			}
		}
	case "issue_assigned":
		result[eventUpdatedAllStr] = true
		result[eventUpdatedAssigneeStr] = true
	case "":
		result[webhookEventToEnumMap[we]] = true
	default:
		result[eventTypeNameToEnumMap[etn]] = true
	}

	return result
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
