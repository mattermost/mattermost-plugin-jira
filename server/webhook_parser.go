// See License for license information.
// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/v5/model"
)

var webhookWrapperFunc func(wh Webhook) Webhook

func ParseWebhook(bb []byte) (wh Webhook, err error) {
	defer func() {
		if err == nil || err == ErrWebhookIgnored {
			return
		}
		if os.Getenv("MM_PLUGIN_JIRA_DEBUG_WEBHOOKS") == "" {
			return
		}
		f, _ := ioutil.TempFile(os.TempDir(),
			fmt.Sprintf("jira_plugin_webhook_%s_*.json",
				time.Now().Format("2006-01-02-15-04")))
		if f == nil {
			return
		}
		_, _ = f.Write(bb)
		_ = f.Close()
		err = errors.WithMessagef(err, "Failed to process webhook. Body stored in %s", f.Name())
	}()

	jwh := &JiraWebhook{}
	err = json.Unmarshal(bb, &jwh)
	if err != nil {
		return nil, err
	}
	if jwh.WebhookEvent == "" {
		return nil, errors.New("No webhook event")
	}
	if jwh.Issue.Fields == nil {
		return nil, ErrWebhookIgnored
	}

	switch jwh.WebhookEvent {
	case "jira:issue_created":
		wh = parseWebhookCreated(jwh)
	case "jira:issue_deleted":
		wh = parseWebhookDeleted(jwh)
	case "jira:issue_updated":
		switch jwh.IssueEventTypeName {
		case "issue_assigned":
			wh = parseWebhookAssigned(jwh, jwh.ChangeLog.Items[0].FromString, jwh.ChangeLog.Items[0].ToString)
		case "issue_updated", "issue_generic", "issue_resolved", "issue_closed", "issue_work_started", "issue_reopened":
			wh = parseWebhookChangeLog(jwh)
		case "issue_commented":
			wh, err = parseWebhookCommentCreated(jwh)
		case "issue_comment_edited":
			wh, err = parseWebhookCommentUpdated(jwh)
		case "issue_comment_deleted":
			wh, err = parseWebhookCommentDeleted(jwh)
		default:
			wh, err = parseWebhookUnspecified(jwh)
		}
	case "comment_created":
		wh, err = parseWebhookCommentCreated(jwh)
	case "comment_updated":
		wh, err = parseWebhookCommentUpdated(jwh)
	case "comment_deleted":
		wh, err = parseWebhookCommentDeleted(jwh)
	default:
		err = errors.Errorf("Unsupported webhook event: %v", jwh.WebhookEvent)

	}
	if err != nil {
		return nil, err
	}
	if wh == nil {
		return nil, errors.Errorf("Unsupported webhook data: %v", jwh.WebhookEvent)
	}

	// For HTTP testing, so we can capture the output of the interface
	if webhookWrapperFunc != nil {
		wh = webhookWrapperFunc(wh)
	}

	return wh, nil
}

func parseWebhookUnspecified(jwh *JiraWebhook) (Webhook, error) {
	if len(jwh.ChangeLog.Items) > 0 {
		return parseWebhookChangeLog(jwh), nil
	}

	if jwh.Comment.ID != "" {
		if jwh.Comment.Updated == jwh.Comment.Created {
			return parseWebhookCommentCreated(jwh)
		}
		return parseWebhookCommentUpdated(jwh)
	}

	return nil, errors.Errorf("Unsupported webhook event: %v", jwh.WebhookEvent)
}

func parseWebhookChangeLog(jwh *JiraWebhook) Webhook {
	var events []*webhook
	var fieldsNotFound []string
	for _, item := range jwh.ChangeLog.Items {
		field := item.Field
		fieldId := item.FieldId
		if fieldId == "" {
			fieldId = field
		}

		from := item.FromString
		to := item.ToString
		fromWithDefault := from
		if fromWithDefault == "" {
			fromWithDefault = "~~None~~"
		}
		toWithDefault := to
		if toWithDefault == "" {
			toWithDefault = "None"
		}

		var event *webhook
		switch {
		case field == "resolution" && to == "" && from != "":
			event = parseWebhookReopened(jwh, from)
		case field == "resolution" && to != "" && from == "":
			event = parseWebhookResolved(jwh, to)
		case field == "status":
			event = parseWebhookUpdatedField(jwh, eventUpdatedStatus, field, fieldId, fromWithDefault, toWithDefault)
		case field == "priority":
			event = parseWebhookUpdatedField(jwh, eventUpdatedPriority, field, fieldId, fromWithDefault, toWithDefault)
		case field == "summary":
			event = parseWebhookUpdatedField(jwh, eventUpdatedSummary, field, fieldId, fromWithDefault, toWithDefault)
		case field == "description":
			event = parseWebhookUpdatedDescription(jwh, from, to)
		case field == "Sprint" && len(to) > 0:
			event = parseWebhookUpdatedField(jwh, eventUpdatedSprint, field, fieldId, fromWithDefault, toWithDefault)
		case field == "Rank" && len(to) > 0:
			event = parseWebhookUpdatedField(jwh, eventUpdatedRank, field, fieldId, strings.ToLower(fromWithDefault), strings.ToLower(toWithDefault))
		case field == "Attachment":
			event = parseWebhookUpdatedAttachments(jwh, from, to)
		case field == "labels":
			event = parseWebhookUpdatedLabels(jwh, from, to, fromWithDefault, toWithDefault)
		case field == "assignee":
			event = parseWebhookAssigned(jwh, from, to)
		case field == "issuetype":
			event = parseWebhookUpdatedField(jwh, eventUpdatedIssuetype, field, fieldId, fromWithDefault, toWithDefault)
		case field == "Fix Version":
			event = parseWebhookUpdatedField(jwh, eventUpdatedFixVersion, field, fieldId, fromWithDefault, toWithDefault)
		case field == "Version":
			event = parseWebhookUpdatedField(jwh, eventUpdatedAffectsVersion, field, fieldId, fromWithDefault, toWithDefault)
		case field == "reporter":
			event = parseWebhookUpdatedField(jwh, eventUpdatedReporter, field, fieldId, fromWithDefault, toWithDefault)
		case field == "Component":
			event = parseWebhookUpdatedField(jwh, eventUpdatedComponents, field, fieldId, fromWithDefault, toWithDefault)
		case item.FieldType == "custom":
			eventType := fmt.Sprintf("event_updated_%s", fieldId)
			event = parseWebhookUpdatedField(jwh, eventType, field, fieldId, fromWithDefault, toWithDefault)
		}

		if event != nil {
			events = append(events, event)
		} else {
			fieldsNotFound = append(fieldsNotFound, field)
		}
	}
	if len(events) == 0 {
		return nil
	} else if len(events) == 1 {
		return events[0]
	} else {
		return mergeWebhookEvents(events)
	}
}

func parseWebhookCreated(jwh *JiraWebhook) Webhook {
	wh := newWebhook(jwh, eventCreated, "**created**")
	wh.text = jwh.mdIssueDescription()

	if jwh.Issue.Fields == nil {
		return wh
	}

	var fields []*model.SlackAttachmentField
	if jwh.Issue.Fields.Assignee != nil {
		fields = append(fields, &model.SlackAttachmentField{
			Title: "Assignee",
			Value: jwh.Issue.Fields.Assignee.DisplayName,
			Short: true,
		})
	}
	if jwh.Issue.Fields.Priority != nil {
		fields = append(fields, &model.SlackAttachmentField{
			Title: "Priority",
			Value: jwh.Issue.Fields.Priority.Name,
			Short: true,
		})
	}
	if len(fields) > 0 {
		wh.fields = fields
	}

	appendNotificationForAssignee(wh)

	return wh
}

func parseWebhookDeleted(jwh *JiraWebhook) Webhook {
	wh := newWebhook(jwh, eventDeleted, "**deleted**")
	if jwh.Issue.Fields != nil && jwh.Issue.Fields.Resolution == nil {
		wh.eventTypes = wh.eventTypes.Add(eventDeletedUnresolved)
	}
	return wh
}

func parseWebhookCommentCreated(jwh *JiraWebhook) (Webhook, error) {
	// The "comment_xxx" events from Jira Server come incomplete,
	// i.e. with just minimal metadata. We toss them out since they
	// are rather useless for our use case. Instead the Jira server
	// webhooks receive and process jira:issue_updated with eventTypes
	// "issue_commented", etc.
	//
	// Detect this condition by checking that jwh.Issue.ID
	if jwh.Issue.ID == "" {
		return nil, ErrWebhookIgnored
	}

	commentAuthor := mdUser(&jwh.Comment.UpdateAuthor)

	wh := &webhook{
		JiraWebhook: jwh,
		eventTypes:  NewStringSet(eventCreatedComment),
		headline:    fmt.Sprintf("%s **commented** on %s", commentAuthor, jwh.mdKeySummaryLink()),
		text:        truncate(quoteIssueComment(jwh.Comment.Body), 3000),
	}

	appendCommentNotifications(wh, "**mentioned** you in a new comment on")

	return wh, nil
}

// appendCommentNotifications modifies wh
func appendCommentNotifications(wh *webhook, verb string) {
	jwh := wh.JiraWebhook
	commentAuthor := mdUser(&jwh.Comment.UpdateAuthor)

	message := fmt.Sprintf("%s %s %s:\n%s",
		commentAuthor, verb, jwh.mdKeySummaryLink(), quoteIssueComment(jwh.Comment.Body))
	assigneeMentioned := false

	for _, u := range parseJIRAUsernamesFromText(wh.Comment.Body) {
		isAccountId := false
		if strings.HasPrefix(u, "accountid:") {
			u = u[10:]
			isAccountId = true
		}

		// don't mention the author of the comment
		if u == jwh.User.Name || u == jwh.User.AccountID {
			continue
		}

		// Avoid duplicated mention for assignee. Boolean value is checked after the loop.
		if jwh.Issue.Fields.Assignee != nil && (u == jwh.Issue.Fields.Assignee.Name || u == jwh.Issue.Fields.Assignee.AccountID) {
			assigneeMentioned = true
		}

		notification := webhookUserNotification{
			message:     message,
			postType:    PostTypeMention,
			commentSelf: jwh.Comment.Self,
		}

		if isAccountId {
			notification.jiraAccountID = u
		} else {
			notification.jiraUsername = u
		}

		wh.notifications = append(wh.notifications, notification)
	}

	// Don't send a notification to the assignee if they don't exist, or if are also the author.
	// Also, if the assignee was mentioned above, avoid sending a duplicate notification here.
	// Jira Server uses name field, Jira Cloud uses the AccountID field.
	if assigneeMentioned || jwh.Issue.Fields.Assignee == nil || jwh.Issue.Fields.Assignee.Name == jwh.User.Name ||
		(jwh.Issue.Fields.Assignee.AccountID != "" && jwh.Issue.Fields.Assignee.AccountID == jwh.Comment.UpdateAuthor.AccountID) {
		return
	}

	wh.notifications = append(wh.notifications, webhookUserNotification{
		jiraUsername:  jwh.Issue.Fields.Assignee.Name,
		jiraAccountID: jwh.Issue.Fields.Assignee.AccountID,
		message:       fmt.Sprintf("%s **commented** on %s:\n>%s", commentAuthor, jwh.mdKeySummaryLink(), jwh.Comment.Body),
		postType:      PostTypeComment,
		commentSelf:   jwh.Comment.Self,
	})
}

func quoteIssueComment(comment string) string {
	return "> " + strings.ReplaceAll(comment, "\n", "\n> ")
}

func parseWebhookCommentDeleted(jwh *JiraWebhook) (Webhook, error) {
	if jwh.Issue.ID == "" {
		return nil, ErrWebhookIgnored
	}

	// Jira server vs Jira cloud pass the user info differently
	user := ""
	if jwh.User.Key != "" {
		user = mdUser(&jwh.User)
	} else if jwh.Comment.UpdateAuthor.Key != "" || jwh.Comment.UpdateAuthor.AccountID != "" {
		user = mdUser(&jwh.Comment.UpdateAuthor)
	}
	if user == "" {
		return nil, errors.New("No update author found")
	}

	return &webhook{
		JiraWebhook: jwh,
		eventTypes:  NewStringSet(eventDeletedComment),
		headline:    fmt.Sprintf("%s **deleted comment** in %s", user, jwh.mdKeySummaryLink()),
	}, nil
}

func parseWebhookCommentUpdated(jwh *JiraWebhook) (Webhook, error) {
	if jwh.Issue.ID == "" {
		return nil, ErrWebhookIgnored
	}

	wh := &webhook{
		JiraWebhook: jwh,
		eventTypes:  NewStringSet(eventUpdatedComment),
		headline:    fmt.Sprintf("%s **edited comment** in %s", mdUser(&jwh.Comment.UpdateAuthor), jwh.mdKeySummaryLink()),
		text:        truncate(quoteIssueComment(jwh.Comment.Body), 3000),
	}

	appendCommentNotifications(wh, "**mentioned** you in a comment update on")
	return wh, nil
}

func parseWebhookAssigned(jwh *JiraWebhook, from, to string) *webhook {
	wh := newWebhook(jwh, eventUpdatedAssignee, "**assigned** %s to", jwh.mdIssueAssignee())
	fromFixed := from
	if fromFixed == "" {
		fromFixed = "_nobody_"
	}
	toFixed := to
	if toFixed == "" {
		toFixed = "_nobody_"
	}
	wh.fieldInfo = webhookField{"assignee", "assignee", fromFixed, toFixed}

	appendNotificationForAssignee(wh)

	return wh
}

// appendNotificationForAssignee modifies wh
func appendNotificationForAssignee(wh *webhook) {
	jwh := wh.JiraWebhook
	if jwh.Issue.Fields.Assignee == nil {
		return
	}

	// Don't send a notification to the assignee if they are the one who made the change. (They probably know already.)
	if (jwh.User.Name != "" && jwh.User.Name == jwh.Issue.Fields.Assignee.Name) ||
		(jwh.User.AccountID != "" && jwh.User.AccountID == jwh.Issue.Fields.Assignee.AccountID) {
		return
	}

	wh.notifications = append(wh.notifications, webhookUserNotification{
		jiraUsername:  jwh.Issue.Fields.Assignee.Name,
		jiraAccountID: jwh.Issue.Fields.Assignee.AccountID,
		message:       fmt.Sprintf("%s **assigned** you to %s", jwh.mdUser(), jwh.mdKeySummaryLink()),
	})
}

func parseWebhookReopened(jwh *JiraWebhook, from string) *webhook {
	wh := newWebhook(jwh, eventUpdatedReopened, "**reopened**")
	wh.fieldInfo = webhookField{"reopened", "resolution", from, "Open"}
	return wh
}

func parseWebhookResolved(jwh *JiraWebhook, to string) *webhook {
	wh := newWebhook(jwh, eventUpdatedResolved, "**resolved**")
	wh.fieldInfo = webhookField{"resolved", "resolution", "Open", to}
	return wh
}

func parseWebhookUpdatedField(jwh *JiraWebhook, eventType string, field, fieldId, from, to string) *webhook {
	wh := newWebhook(jwh, eventType, "**updated** %s from %q to %q on", field, from, to)
	wh.fieldInfo = webhookField{field, fieldId, from, to}
	return wh
}

func parseWebhookUpdatedDescription(jwh *JiraWebhook, from, to string) *webhook {
	wh := newWebhook(jwh, eventUpdatedDescription, "**edited** the description of")
	fromFmttd := "\n**From:** " + truncate(from, 500)
	toFmttd := "\n**To:** " + truncate(to, 500)
	wh.fieldInfo = webhookField{"description", "description", fromFmttd, toFmttd}
	wh.text = jwh.mdIssueDescription()
	return wh
}

func parseWebhookUpdatedAttachments(jwh *JiraWebhook, from, to string) *webhook {
	wh := newWebhook(jwh, eventUpdatedAttachment, mdAddRemove(from, to, "**attached**", "**removed** attachments"))
	wh.fieldInfo = webhookField{name: "attachments"}
	return wh
}

func parseWebhookUpdatedLabels(jwh *JiraWebhook, from, to, fromWithDefault, toWithDefault string) *webhook {
	wh := newWebhook(jwh, eventUpdatedLabels, mdAddRemove(from, to, "**added** labels", "**removed** labels"))
	wh.fieldInfo = webhookField{"labels", "labels", fromWithDefault, toWithDefault}
	return wh
}

// mergeWebhookEvents assumes len(events) > 1
func mergeWebhookEvents(events []*webhook) Webhook {
	merged := &webhook{
		JiraWebhook: events[0].JiraWebhook,
		headline:    events[0].mdUser() + " **updated** " + events[0].mdKeySummaryLink(),
		eventTypes:  NewStringSet(),
	}

	for _, event := range events {
		merged.eventTypes = merged.eventTypes.Union(event.eventTypes)
		strikePre := "~~"
		strikePost := "~~"
		if event.fieldInfo.name == "description" || strings.HasPrefix(event.fieldInfo.from, "~~") {
			strikePre = ""
			strikePost = ""
		}
		msg := "**" + strings.Title(event.fieldInfo.name) + ":** " + strikePre +
			event.fieldInfo.from + strikePost + " " + event.fieldInfo.to
		merged.fields = append(merged.fields, &model.SlackAttachmentField{
			Value: msg,
			Short: false,
		})
	}

	return merged
}
