// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/mattermost/mattermost/server/public/model"
)

var errWebhookeventUnsupported = errors.New("Unsupported webhook event")

var webhookWrapperFunc func(wh Webhook) Webhook

func ParseWebhook(bb []byte) (wh Webhook, err error) {
	defer func() {
		if err == nil || err == ErrWebhookIgnored {
			return
		}
		if os.Getenv("MM_PLUGIN_JIRA_DEBUG_WEBHOOKS") == "" {
			return
		}
		f, _ := os.CreateTemp(os.TempDir(),
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
		return nil, errors.New("no webhook event")
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
	case commentCreated:
		wh, err = parseWebhookCommentCreated(jwh)
	case commentUpdated:
		wh, err = parseWebhookCommentUpdated(jwh)
	case commentDeleted:
		wh, err = parseWebhookCommentDeleted(jwh)
	case worklogUpdated:
		// not supported
	default:
		err = errors.Wrapf(errWebhookeventUnsupported, "event: %v", jwh.WebhookEvent)
	}
	if err != nil {
		return nil, err
	}
	if wh == nil {
		return nil, errors.Wrapf(errWebhookeventUnsupported, "event: %v", jwh.WebhookEvent)
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
	for _, item := range jwh.ChangeLog.Items {
		field := item.Field
		fieldID := item.FieldID
		if fieldID == "" {
			fieldID = field
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
		case field == resolutionField && to == "" && from != "":
			event = parseWebhookReopened(jwh, from)
		case field == resolutionField && to != "" && from == "":
			event = parseWebhookResolved(jwh, to)
		case field == statusField:
			event = parseWebhookUpdatedField(jwh, eventUpdatedStatus, field, fieldID, fromWithDefault, toWithDefault)
		case field == priorityField:
			event = parseWebhookUpdatedField(jwh, eventUpdatedPriority, field, fieldID, fromWithDefault, toWithDefault)
		case field == "summary":
			event = parseWebhookUpdatedField(jwh, eventUpdatedSummary, field, fieldID, fromWithDefault, toWithDefault)
		case field == descriptionField:
			event = parseWebhookUpdatedDescription(jwh, from, to)
		case field == "Sprint" && len(to) > 0:
			event = parseWebhookUpdatedField(jwh, eventUpdatedSprint, field, fieldID, fromWithDefault, toWithDefault)
		case field == "Rank" && len(to) > 0:
			event = parseWebhookUpdatedField(jwh, eventUpdatedRank, field, fieldID, strings.ToLower(fromWithDefault), strings.ToLower(toWithDefault))
		case field == "Attachment":
			event = parseWebhookUpdatedAttachments(jwh, from, to, fromWithDefault, toWithDefault)
		case field == labelsField:
			event = parseWebhookUpdatedLabels(jwh, from, to, fromWithDefault, toWithDefault)
		case field == "assignee":
			event = parseWebhookAssigned(jwh, from, to)
		case field == "issuetype":
			event = parseWebhookUpdatedField(jwh, eventUpdatedIssuetype, field, fieldID, fromWithDefault, toWithDefault)
		case field == "Fix Version":
			event = parseWebhookUpdatedField(jwh, eventUpdatedFixVersion, field, fieldID, fromWithDefault, toWithDefault)
		case field == "Version":
			event = parseWebhookUpdatedField(jwh, eventUpdatedAffectsVersion, field, fieldID, fromWithDefault, toWithDefault)
		case field == "reporter":
			event = parseWebhookUpdatedField(jwh, eventUpdatedReporter, field, fieldID, fromWithDefault, toWithDefault)
		case field == "Component":
			event = parseWebhookUpdatedField(jwh, eventUpdatedComponents, field, fieldID, fromWithDefault, toWithDefault)
		case item.FieldType == "custom":
			eventType := fmt.Sprintf("event_updated_%s", fieldID)
			event = parseWebhookUpdatedField(jwh, eventType, field, fieldID, fromWithDefault, toWithDefault)
		}

		if event != nil {
			events = append(events, event)
		}
	}

	switch len(events) {
	case 0:
		return nil
	case 1:
		return events[0]
	default:
		return mergeWebhookEvents(events)
	}
}

func parseWebhookCreated(jwh *JiraWebhook) Webhook {
	wh := newWebhook(jwh, eventCreated, "**created**")
	wh.text = preProcessText(jwh.mdIssueDescription())

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
	commentAuthor := mdUser(&jwh.Comment.UpdateAuthor)

	wh := &webhook{
		JiraWebhook: jwh,
		eventTypes:  NewStringSet(eventCreatedComment),
		headline:    fmt.Sprintf("%s **commented** on %s", commentAuthor, jwh.mdKeySummaryLink()),
		text:        truncate(quoteIssueComment(preProcessText(jwh.Comment.Body)), 3000),
	}

	appendCommentNotifications(wh, "**mentioned** you in a new comment on")

	return wh, nil
}

// appendCommentNotifications modifies wh
func appendCommentNotifications(wh *webhook, verb string) {
	jwh := wh.JiraWebhook
	commentAuthor := mdUser(&jwh.Comment.UpdateAuthor)

	// Process Jira markup to markdown before quoting
	processedComment := preProcessText(jwh.Comment.Body)
	message := fmt.Sprintf("%s %s %s:\n%s",
		commentAuthor, verb, jwh.mdKeySummaryLink(), quoteIssueComment(processedComment))
	assigneeMentioned := false

	for _, u := range parseJIRAUsernamesFromText(wh.Comment.Body) {
		isAccountID := false
		if strings.HasPrefix(u, "accountid:") {
			u = u[10:]
			isAccountID = true
		}

		// don't mention the author of the comment
		if u == jwh.User.Name || u == jwh.User.AccountID || u == jwh.Comment.Author.AccountID {
			continue
		}

		// Avoid duplicated mention for assignee. Boolean value is checked after the loop.
		if jwh.Issue.Fields.Assignee != nil && (u == jwh.Issue.Fields.Assignee.Name || u == jwh.Issue.Fields.Assignee.AccountID) {
			assigneeMentioned = true
		}

		notification := webhookUserNotification{
			message:          message,
			postType:         PostTypeMention,
			commentSelf:      jwh.Comment.Self,
			notificationType: "mention",
		}

		if isAccountID {
			notification.jiraAccountID = u
		} else {
			notification.jiraUsername = u
		}

		wh.notifications = append(wh.notifications, notification)
	}

	// Don't send a notification to the assignee if they don't exist, or if are also the author.
	// Also, if the assignee was mentioned above, avoid sending a duplicate notification here.
	// Jira Server uses name field, Jira Cloud uses the AccountID field.
	if assigneeMentioned || jwh.Issue.Fields.Assignee == nil ||
		(jwh.Issue.Fields.Assignee.Name != "" && jwh.Issue.Fields.Assignee.Name == jwh.User.Name) ||
		(jwh.Issue.Fields.Assignee.AccountID != "" && jwh.Issue.Fields.Assignee.AccountID == jwh.Comment.UpdateAuthor.AccountID) {
		return
	}

	wh.notifications = append(wh.notifications, webhookUserNotification{
		jiraUsername:     jwh.Issue.Fields.Assignee.Name,
		jiraAccountID:    jwh.Issue.Fields.Assignee.AccountID,
		message:          fmt.Sprintf("%s **commented** on %s:\n%s", commentAuthor, jwh.mdKeySummaryLink(), quoteIssueComment(processedComment)),
		postType:         PostTypeComment,
		commentSelf:      jwh.Comment.Self,
		notificationType: "assignee",
	})
}

func quoteIssueComment(comment string) string {
	if strings.TrimSpace(comment) == "" {
		return ""
	}
	return "> " + strings.ReplaceAll(comment, "\n", "\n> ")
}

// preProcessText processes the given string to apply various formatting transformations.
// The purpose of the function is to convert the formatting provided by JIRA into the corresponding formatting supported by Mattermost.
// This includes converting asterisks to bold, hyphens to strikethrough, JIRA-style headings to Markdown headings,
// JIRA code blocks to inline code, numbered lists to Markdown lists, colored text to plain text, and JIRA links to Markdown links.
// For more reference, please visit https://github.com/mattermost/mattermost-plugin-jira/issues/1096
func preProcessText(jiraMarkdownString string) string {
	asteriskRegex := regexp.MustCompile(`\*(\w+)\*`)
	hyphenRegex := regexp.MustCompile(`\B-([\w\d\s]+)-\B`)
	headingRegex := regexp.MustCompile(`(?m)^(h[1-6]\.)\s+`)
	langSpecificCodeBlockRegex := regexp.MustCompile(`\{code:[^}]+\}([\s\S]*?)\{code\}`)
	numberedListRegex := regexp.MustCompile(`^#\s+`)
	colouredTextRegex := regexp.MustCompile(`\{color:[^}]+\}(.*?)\{color\}`)
	linkRegex := regexp.MustCompile(`\[(.*?)\|([^|\]]+)(?:\|([^|\]]+))?\]`)
	quoteRegex := regexp.MustCompile(`\{quote\}(.*?)\{quote\}`)
	codeBlockRegex := regexp.MustCompile(`\{\{(.+?)\}\}`)
	noFormatRegex := regexp.MustCompile(`\{noformat\}([\s\S]*?)\{noformat\}`)
	doubleCurlyRegex := regexp.MustCompile(`\{\{(.*?)\}\}`)

	// the below code converts lines starting with "#" into a numbered list. It increments the counter if consecutive lines are numbered,
	// otherwise resets it to 1. The "#" is replaced with the corresponding number and period. Non-numbered lines are added unchanged.
	var counter int
	var lastLineWasNumberedList bool
	var result []string
	lines := strings.Split(jiraMarkdownString, "\n")
	for _, line := range lines {
		if numberedListRegex.MatchString(line) {
			if !lastLineWasNumberedList {
				counter = 1
			} else {
				counter++
			}
			line = strconv.Itoa(counter) + ". " + strings.TrimPrefix(line, "# ")
			lastLineWasNumberedList = true
		} else {
			lastLineWasNumberedList = false
		}
		result = append(result, line)
	}
	processedString := strings.Join(result, "\n")

	// the below code converts links in the format "[text|url]" or "[text|url|optional]" to Markdown links. If the text is empty,
	// the URL is used for both the text and link. If the optional part is present, it's ignored. Unrecognized patterns remain unchanged.
	processedString = linkRegex.ReplaceAllStringFunc(processedString, func(link string) string {
		parts := linkRegex.FindStringSubmatch(link)
		if len(parts) == 4 {
			if parts[1] == "" {
				return "[" + parts[2] + "](" + parts[2] + ")"
			}
			if parts[3] != "" {
				return "[" + parts[1] + "](" + parts[2] + ")"
			}
			return "[" + parts[1] + "](" + parts[2] + ")"
		}
		return link
	})

	processedString = asteriskRegex.ReplaceAllStringFunc(processedString, func(word string) string {
		return "**" + strings.Trim(word, "*") + "**"
	})

	processedString = hyphenRegex.ReplaceAllStringFunc(processedString, func(word string) string {
		if strings.Contains(word, "accountid:") {
			return word
		}
		return "~~" + strings.Trim(word, "-") + "~~"
	})

	processedString = headingRegex.ReplaceAllStringFunc(processedString, func(heading string) string {
		level := heading[1]
		hashes := strings.Repeat("#", int(level-'0'))
		return hashes + " "
	})

	processedString = codeBlockRegex.ReplaceAllStringFunc(processedString, func(match string) string {
		curlyContent := codeBlockRegex.FindStringSubmatch(match)[1]
		return "`" + curlyContent + "`"
	})

	processedString = colouredTextRegex.ReplaceAllString(processedString, "$1")

	processedString = quoteRegex.ReplaceAllStringFunc(processedString, func(quote string) string {
		quotedText := quote[strings.Index(quote, "}")+1 : strings.LastIndex(quote, "{quote}")]
		return "> " + quotedText
	})

	processedString = doubleCurlyRegex.ReplaceAllStringFunc(processedString, func(match string) string {
		content := match[2 : len(match)-2]
		return fmt.Sprintf("`%s`", content)
	})

	// handles single and multi line language specific code blocks
	processedString = langSpecificCodeBlockRegex.ReplaceAllStringFunc(processedString, func(langSpecificBlock string) string {
		startIndex := strings.Index(langSpecificBlock, "{code:")
		endIndex := strings.LastIndex(langSpecificBlock, "{code}")
		if startIndex == -1 || endIndex == -1 || startIndex == endIndex {
			return langSpecificBlock
		}

		langEndIndex := strings.Index(langSpecificBlock[startIndex:], "}")
		if langEndIndex == -1 {
			return langSpecificBlock
		}

		contentStartIndex := startIndex + langEndIndex + 1
		content := langSpecificBlock[contentStartIndex:endIndex]

		lines := strings.Split(content, "\n")

		if len(lines) == 1 {
			return "`" + lines[0] + "`"
		}

		for i := range lines {
			if len(lines[i]) > 0 {
				lines[i] = "`" + lines[i] + "`"
			}
		}

		return "\n" + strings.Join(lines, "\n") + "\n"
	})

	// handles single and multi line non-language specific code blocks
	processedString = noFormatRegex.ReplaceAllStringFunc(processedString, func(noFormatBlock string) string {
		startIndex := strings.Index(noFormatBlock, "{noformat}")
		endIndex := strings.LastIndex(noFormatBlock, "{noformat}")
		if startIndex == -1 || endIndex == -1 || startIndex == endIndex {
			return noFormatBlock
		}

		content := noFormatBlock[startIndex+len("{noformat}") : endIndex]

		lines := strings.Split(content, "\n")

		if len(lines) == 1 {
			return "`" + lines[0] + "`"
		}

		for i := range lines {
			if len(lines[i]) > 0 {
				lines[i] = "`" + lines[i] + "`"
			}
		}

		return "\n" + strings.Join(lines, "\n") + "\n"
	})

	return processedString
}

func parseWebhookCommentDeleted(jwh *JiraWebhook) (Webhook, error) {
	// Jira server vs Jira cloud pass the user info differently
	user := ""
	if jwh.User.Key != "" {
		user = mdUser(&jwh.User)
	} else if jwh.Comment.UpdateAuthor.Key != "" || jwh.Comment.UpdateAuthor.AccountID != "" {
		user = mdUser(&jwh.Comment.UpdateAuthor)
	}
	if user == "" {
		return nil, errors.New("no update author found")
	}

	return &webhook{
		JiraWebhook: jwh,
		eventTypes:  NewStringSet(eventDeletedComment),
		headline:    fmt.Sprintf("%s **deleted comment** in %s", user, jwh.mdKeySummaryLink()),
	}, nil
}

func parseWebhookCommentUpdated(jwh *JiraWebhook) (Webhook, error) {
	wh := &webhook{
		JiraWebhook: jwh,
		eventTypes:  NewStringSet(eventUpdatedComment),
		headline:    fmt.Sprintf("%s **edited comment** in %s", mdUser(&jwh.Comment.UpdateAuthor), jwh.mdKeySummaryLink()),
		text:        truncate(quoteIssueComment(preProcessText(jwh.Comment.Body)), 3000),
	}

	return wh, nil
}

func parseWebhookAssigned(jwh *JiraWebhook, from, to string) *webhook {
	wh := newWebhook(jwh, eventUpdatedAssignee, "**assigned** %s to", jwh.mdIssueAssignee())
	fromFixed := from
	if fromFixed == "" {
		fromFixed = Nobody
	}
	toFixed := to
	if toFixed == "" {
		toFixed = Nobody
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
	wh.fieldInfo = webhookField{"reopened", resolutionField, from, "Open"}
	return wh
}

func parseWebhookResolved(jwh *JiraWebhook, to string) *webhook {
	wh := newWebhook(jwh, eventUpdatedResolved, "**resolved**")
	wh.fieldInfo = webhookField{"resolved", resolutionField, "Open", to}
	return wh
}

func parseWebhookUpdatedField(jwh *JiraWebhook, eventType string, field, fieldID, from, to string) *webhook {
	wh := newWebhook(jwh, eventType, "**updated** %s from %q to %q on", field, from, to)
	wh.fieldInfo = webhookField{field, fieldID, from, to}
	return wh
}

func parseWebhookUpdatedDescription(jwh *JiraWebhook, from, to string) *webhook {
	wh := newWebhook(jwh, eventUpdatedDescription, "**edited** the description of")
	fromFmttd := "\n**From:** " + truncate(from, 500)
	toFmttd := "\n**To:** " + truncate(to, 500)
	wh.fieldInfo = webhookField{descriptionField, descriptionField, fromFmttd, toFmttd}
	wh.text = preProcessText(jwh.mdIssueDescription())
	return wh
}

func parseWebhookUpdatedAttachments(jwh *JiraWebhook, from, to, fromWithDefault, toWithDefault string) *webhook {
	wh := newWebhook(jwh, eventUpdatedAttachment, "%s", mdAddRemove(from, to, "**attached**", "**removed** attachments"))
	wh.fieldInfo = webhookField{"attachments", "attachment", from, to}
	return wh
}

func parseWebhookUpdatedLabels(jwh *JiraWebhook, from, to, fromWithDefault, toWithDefault string) *webhook {
	wh := newWebhook(jwh, eventUpdatedLabels, "%s", mdAddRemove(from, to, "**added** labels", "**removed** labels"))
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
		strike := "~~"
		if event.fieldInfo.name == descriptionField || strings.HasPrefix(event.fieldInfo.from, strike) {
			strike = ""
		}
		// Use the english language for now. Using the server's local might be better.
		msg := "**" + cases.Title(language.English, cases.NoLower).String(event.fieldInfo.name) + ":** " + strike +
			event.fieldInfo.from + strike + " " + event.fieldInfo.to
		merged.fields = append(merged.fields, &model.SlackAttachmentField{
			Value: msg,
			Short: false,
		})
	}

	return merged
}
