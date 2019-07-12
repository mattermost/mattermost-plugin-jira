// See License for license information.
// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/model"
)

var webhookWrapperFunc func(wh Webhook) Webhook

func ParseWebhook(in io.Reader) (wh Webhook, jwh *JiraWebhook, err error) {
	bb, err := ioutil.ReadAll(in)
	if err != nil {
		return nil, nil, err
	}
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

	jwh = &JiraWebhook{}
	err = json.Unmarshal(bb, &jwh)
	if err != nil {
		return nil, nil, err
	}
	if jwh.WebhookEvent == "" {
		return nil, jwh, errors.New("No webhook event")
	}
	if jwh.Issue.Fields == nil {
		return nil, jwh, ErrWebhookIgnored
	}

	switch jwh.WebhookEvent {
	case "jira:issue_created":
		wh = parseWebhookCreated(jwh)
	case "jira:issue_deleted":
		wh = parseWebhookDeleted(jwh)
	case "jira:issue_updated":
		switch jwh.IssueEventTypeName {
		case "issue_assigned":
			wh = parseWebhookAssigned(jwh)
		case "issue_updated", "issue_generic":
			wh = parseWebhookChangeLog(jwh)
		case "issue_commented":
			wh, err = parseWebhookCommentCreated(jwh)
		case "issue_comment_edited":
			wh = parseWebhookCommentUpdated(jwh)
		case "issue_comment_deleted":
			wh, err = parseWebhookCommentDeleted(jwh)
		}
	case "comment_created":
		wh, err = parseWebhookCommentCreated(jwh)
	case "comment_updated":
		wh = parseWebhookCommentUpdated(jwh)
	case "comment_deleted":
		wh, err = parseWebhookCommentDeleted(jwh)
	}
	if err != nil {
		return nil, jwh, err
	}
	if wh == nil {
		return nil, jwh, errors.Errorf("Unsupported webhook data: %v", jwh.WebhookEvent)
	}

	// For HTTP testing, so we can capture the output of the interface
	if webhookWrapperFunc != nil {
		wh = webhookWrapperFunc(wh)
	}

	return wh, jwh, nil
}

func parseWebhookChangeLog(jwh *JiraWebhook) Webhook {
	var events []*webhook
	for _, item := range jwh.ChangeLog.Items {
		field := item.Field
		to := item.ToString
		from := item.FromString
		switch {
		case field == "resolution" && to == "" && from != "":
			events = append(events, parseWebhookReopened(jwh, from))
		case field == "resolution" && to != "" && from == "":
			events = append(events, parseWebhookResolved(jwh, to))
		case field == "status":
			events = append(events, parseWebhookUpdatedField(jwh, eventUpdatedStatus, field, from, to))
		case field == "priority":
			events = append(events, parseWebhookUpdatedField(jwh, eventUpdatedPriority, field, from, to))
		case field == "summary":
			events = append(events, parseWebhookUpdatedField(jwh, eventUpdatedSummary, field, from, to))
		case field == "description":
			// need to handle wh.text
			events = append(events, parseWebhookUpdatedDescription(jwh))
		case field == "Sprint" && len(to) > 0:
			events = append(events, parseWebhookUpdatedField(jwh, eventUpdatedSprint, field, from, to))
		case field == "Rank" && len(to) > 0:
			events = append(events, parseWebhookUpdatedField(jwh, eventUpdatedRank, field, strings.ToLower(from), strings.ToLower(to)))
		case field == "Attachment":
			events = append(events, parseWebhookUpdatedAttachments(jwh, from, to))
		case field == "labels":
			events = append(events, parseWebhookUpdatedLabels(jwh, from, to))
		case field == "assignee":
			events = append(events, parseWebhookAssigned(jwh))
		case field == "issuetype":
			events = append(events, parseWebhookUpdatedField(jwh, eventUpdatedIssuetype, field, from, to))
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
	wh := newWebhook(jwh, eventCreated, "created")
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

	return wh
}

func parseWebhookDeleted(jwh *JiraWebhook) Webhook {
	wh := newWebhook(jwh, eventDeleted, "deleted")
	if jwh.Issue.Fields != nil && jwh.Issue.Fields.Resolution == nil {
		wh.eventMask = wh.eventMask | eventDeletedUnresolved
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
		eventMask:   eventCreatedComment,
		headline:    fmt.Sprintf("%s commented on %s", commentAuthor, jwh.mdKeySummaryLink()),
		text:        truncate(jwh.Comment.Body, 3000),
	}

	message := fmt.Sprintf("%s mentioned you on %s:\n>%s",
		jwh.mdUser(), jwh.mdKeySummaryLink(), jwh.Comment.Body)
	for _, u := range parseJIRAUsernamesFromText(wh.Comment.Body) {
		// don't mention the author of the comment
		if u == jwh.User.Name {
			continue
		}
		// don't mention the Issue assignee, will gets a special notice
		if jwh.Issue.Fields.Assignee != nil && u == jwh.Issue.Fields.Assignee.Name {
			continue
		}

		wh.notifications = append(wh.notifications, webhookNotification{
			jiraUsername: u,
			message:      message,
			postType:     PostTypeMention,
			commentSelf:  jwh.Comment.Self,
		})
	}

	// Don't send a notification to the assignee if they don't exist, or if are also the author.
	// Jira Server uses name field, Jira Cloud uses the AccountID field.
	if jwh.Issue.Fields.Assignee == nil || jwh.Issue.Fields.Assignee.Name == jwh.User.Name ||
		(jwh.Issue.Fields.Assignee.AccountID != "" && jwh.Comment.UpdateAuthor.AccountID != "" && jwh.Issue.Fields.Assignee.AccountID == jwh.Comment.UpdateAuthor.AccountID) {
		return wh, nil
	}

	wh.notifications = append(wh.notifications, webhookNotification{
		jiraUsername:  jwh.Issue.Fields.Assignee.Name,
		jiraAccountID: jwh.Issue.Fields.Assignee.AccountID,
		message:       fmt.Sprintf("%s commented on %s:\n>%s", commentAuthor, jwh.mdKeySummaryLink(), jwh.Comment.Body),
		postType:      PostTypeComment,
		commentSelf:   jwh.Comment.Self,
	})

	return wh, nil
}

func parseWebhookCommentDeleted(jwh *JiraWebhook) (Webhook, error) {
	// Jira server vs Jira cloud pass the user info differently
	user := ""
	if jwh.User.Key != "" {
		user = mdUser(&jwh.User)
	} else if jwh.Comment.UpdateAuthor.Key != "" {
		user = mdUser(&jwh.Comment.UpdateAuthor)
	}
	if user == "" {
		return nil, errors.New("No update author found")
	}

	return &webhook{
		JiraWebhook: jwh,
		eventMask:   eventDeletedComment,
		headline:    fmt.Sprintf("%s deleted comment in %s", user, jwh.mdKeySummaryLink()),
	}, nil
}

func parseWebhookCommentUpdated(jwh *JiraWebhook) Webhook {
	return &webhook{
		JiraWebhook: jwh,
		eventMask:   eventUpdatedComment,
		headline:    fmt.Sprintf("%s edited comment in %s", mdUser(&jwh.Comment.UpdateAuthor), jwh.mdKeySummaryLink()),
		text:        truncate(jwh.Comment.Body, 3000),
	}
}

func parseWebhookAssigned(jwh *JiraWebhook) *webhook {
	wh := newWebhook(jwh, eventUpdatedAssignee, "assigned %s to", jwh.mdIssueAssignee())
	if jwh.Issue.Fields.Assignee == nil {
		return wh
	}

	// Don't send a notification to the assignee if they are the one who made the change. (They probably know already.)
	if (jwh.User.Name != "" && jwh.User.Name == jwh.Issue.Fields.Assignee.Name) ||
		(jwh.User.AccountID != "" && jwh.Issue.Fields.Assignee.AccountID != "" && jwh.User.AccountID == jwh.Issue.Fields.Assignee.AccountID) {
		return wh
	}

	wh.notifications = append(wh.notifications, webhookNotification{
		jiraUsername:  jwh.Issue.Fields.Assignee.Name,
		jiraAccountID: jwh.Issue.Fields.Assignee.AccountID,
		message:       fmt.Sprintf("%s assigned you to %s", jwh.mdUser(), jwh.mdKeySummaryLink()),
	})
	return wh
}

func parseWebhookReopened(jwh *JiraWebhook, from string) *webhook {
	wh := newWebhook(jwh, eventUpdatedReopened, "reopened")
	wh.eventInfo = webhookEvent{"reopened", from, "Open"}
	return wh
}

func parseWebhookResolved(jwh *JiraWebhook, to string) *webhook {
	wh := newWebhook(jwh, eventUpdatedResolved, "resolved")
	wh.eventInfo = webhookEvent{"resolved", "Open", to}
	return wh
}

func parseWebhookUpdatedField(jwh *JiraWebhook, eventMask uint64, field, from, to string) *webhook {
	wh := newWebhook(jwh, eventMask, "updated %s from %q to %q on", field, from, to)
	wh.eventInfo = webhookEvent{field, from, to}
	return wh
}

func parseWebhookUpdatedDescription(jwh *JiraWebhook) *webhook {
	wh := newWebhook(jwh, eventUpdatedDescription, "edited the description of")
	wh.text = jwh.mdIssueDescription()
	return wh
}

func parseWebhookUpdatedAttachments(jwh *JiraWebhook, from, to string) *webhook {
	wh := newWebhook(jwh, eventUpdatedAttachment, mdAddRemove(from, to, "attached", "removed attachments"))
	wh.eventInfo = webhookEvent{field: "attachments"}
	return wh
}

func parseWebhookUpdatedLabels(jwh *JiraWebhook, from, to string) *webhook {
	wh := newWebhook(jwh, eventUpdatedLabels, mdAddRemove(from, to, "added labels", "removed labels"))
	wh.eventInfo = webhookEvent{field: "labels"}
	return wh
}

// mergeWebhookEvents assumes len(events) > 1
func mergeWebhookEvents(events []*webhook) Webhook {
	merged := &webhook{
		headline: events[0].mdUser() + " updated " + events[0].mdKeySummaryLink(),
	}

	for _, event := range events {
		merged.eventMask = merged.eventMask | event.eventMask
		msg := "**" + strings.Title(event.eventInfo.field) + ":** ~~" +
			event.eventInfo.from + "~~ " + event.eventInfo.to
		merged.fields = append(merged.fields, &model.SlackAttachmentField{
			Value: msg,
			Short: false,
		})
	}

	return merged
}
