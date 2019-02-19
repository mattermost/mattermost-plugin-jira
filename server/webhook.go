// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"fmt"
	"strings"
)

type WebhookUser struct {
	Self         string
	Name         string
	Key          string
	EmailAddress string
	AvatarURLs   map[string]string
	DisplayName  string
	Active       bool
	TimeZone     string
}

type Webhook struct {
	WebhookEvent string
	Issue        struct {
		Self   string
		Key    string
		Fields struct {
			Assignee    *WebhookUser
			Reporter    *WebhookUser
			Summary     string
			Description string
			Priority    *struct {
				Id      string
				Name    string
				IconURL string
			}
			IssueType struct {
				Name    string
				IconURL string
			}
			Resolution *struct {
				Id string
			}
			Status struct {
				Id string
			}
			Labels []string
		}
	}
	User    WebhookUser
	Comment struct {
		Body         string
		UpdateAuthor WebhookUser
	}
	ChangeLog struct {
		Items []struct {
			From       string
			FromString string
			To         string
			ToString   string
			Field      string
		}
	}
	IssueEventTypeName string `json:"issue_event_type_name"`
	RawJSON            string
	showDescription    bool
}

const (
	mdRootStyle   = "## "
	mdUpdateStyle = "###### "
)

func (w *Webhook) Markdown() string {
	switch w.WebhookEvent {
	case "jira:issue_created":
		return mdIssueCreated(w)
	case "jira:issue_updated":
		return mdIssueUpdated(w)
	case "comment_created", "comment_updated", "comment_deleted":
		return mdComment(w)
	}
	return ""
}

func mdIssueCreated(w *Webhook) string {
	s := mdRootStyle
	s += fmt.Sprintf("%v created a %v %v", mdUser(&w.User), mdIssueType(w), mdIssueLongLink(w))
	s += "\n"

	s += join(w,
		mdIssuePriority,
		mdIssueAssignedTo,
		mdIssueReportedBy,
		mdIssueLabels,
		mdIssueHashtags,
	)
	s += "\n"
	s += "\n"

	s += mdIssueDescription(w)

	return s
}

func mdIssueUpdated(w *Webhook) string {
	s := mdUpdateStyle
	headline := ""
	extra := ""

	switch w.IssueEventTypeName {
	case "issue_assigned":
		assignee := "_nobody_"
		if w.Issue.Fields.Assignee != nil {
			assignee = mdUser(w.Issue.Fields.Assignee)
		}
		headline = fmt.Sprintf("assigned %v to %v", mdIssueLongLink(w), assignee)

	case "issue_updated",
		"issue_generic":
		// edited summary, description, updated priority, status, etc.
		headline = mdHeadlineFromChangeLog(w)
		if w.showDescription {
			extra = mdIssueDescription(w)
		}
	}
	if headline == "" {
		return ""
	}

	s += mdUser(&w.User) + " " + headline + " " + mdIssueHashtags(w)
	s += "\n"

	if extra != "" {
		s += extra
	}
	s += "\n"

	return s
}

func mdComment(w *Webhook) string {
	s := mdUpdateStyle
	headline := ""
	extra := ""

	switch w.WebhookEvent {
	case "comment_deleted":
		headline = fmt.Sprintf("removed a comment from %v", mdIssueLongLink(w))

	case "comment_updated":
		headline = fmt.Sprintf("edited a comment in %v", mdIssueLongLink(w))
		extra = w.Comment.Body

	case "comment_created":
		headline = fmt.Sprintf("commented on %v", mdIssueLongLink(w))
		extra = w.Comment.Body

	}
	if headline == "" {
		return ""
	}

	s += mdUser(&w.Comment.UpdateAuthor) + " " + headline + " " + mdIssueHashtags(w)
	s += "\n"
	if extra != "" {
		s += extra
	}
	s += "\n"
	return s
}

func mdHeadlineFromChangeLog(w *Webhook) string {
	for _, item := range w.ChangeLog.Items {
		to := item.ToString
		from := item.FromString
		switch {
		case item.Field == "resolution" && to == "" && from != "":
			return fmt.Sprintf("reopened %v", mdIssueLongLink(w))

		case item.Field == "resolution" && to != "" && from == "":
			return fmt.Sprintf("resolved %v", mdIssueLongLink(w))

		case item.Field == "status" && to == "In Progress":
			return fmt.Sprintf("started working on %v", mdIssueLongLink(w))

		case item.Field == "priority" && item.From > item.To:
			return fmt.Sprintf("raised priority of %v to %v", mdIssueLongLink(w), to)

		case item.Field == "priority" && item.From < item.To:
			return fmt.Sprintf("lowered priority of %v to %v", mdIssueLongLink(w), to)

		case item.Field == "summary":
			return fmt.Sprintf("renamed %v to %v", mdIssueLink(w), mdIssueSummary(w))

		case item.Field == "description":
			w.showDescription = true
			return fmt.Sprintf("edited description of %v", mdIssueLongLink(w))

		case item.Field == "Sprint" && len(to) > 0:
			return fmt.Sprintf("moved %v to %v", mdIssueLongLink(w), to)

		case item.Field == "Rank" && len(to) > 0:
			return fmt.Sprintf("%v %v", strings.ToLower(to), mdIssueLongLink(w))

		case item.Field == "Attachment":
			return fmt.Sprintf("%v %v", mdAddRemove(from, to, "attached", "removed attachments"), mdIssueLongLink(w))

		case item.Field == "labels":
			return fmt.Sprintf("%v %v", mdAddRemove(from, to, "added labels", "removed labels"), mdIssueLongLink(w))
		}
	}
	return ""
}

func mdIssueSummary(w *Webhook) string {
	return truncate(w.Issue.Fields.Summary, 80)
}

func mdIssueDescription(w *Webhook) string {
	return fmt.Sprintf(
		"\n%s\n",
		truncate(
			jiraToMarkdown(w.Issue.Fields.Description),
			3000),
	)
	// return fmt.Sprintf("```\n%v\n```", truncate(w.Issue.Fields.Description, 3000))
}

func mdIssueAssignedTo(w *Webhook) string {
	if w.Issue.Fields.Assignee == nil {
		return ""
	}
	return "Assigned to: " + mdBOLD(mdUser(w.Issue.Fields.Assignee))
}

func mdIssueReportedBy(w *Webhook) string {
	if w.Issue.Fields.Reporter == nil {
		return ""
	}
	return "Reported by: " + mdBOLD(mdUser(w.Issue.Fields.Reporter))
}

func mdIssueLabels(w *Webhook) string {
	if len(w.Issue.Fields.Labels) == 0 {
		return ""
	}
	return "Labels: " + strings.Join(w.Issue.Fields.Labels, ",")
}

func mdIssuePriority(w *Webhook) string {
	return "Priority: " + mdBOLD(w.Issue.Fields.Priority.Name)
}

func mdIssueType(w *Webhook) string {
	return strings.ToLower(w.Issue.Fields.IssueType.Name)
}

func mdIssueLongLink(w *Webhook) string {
	return fmt.Sprintf("[%v: %v](%v/browse/%v)", w.Issue.Key, mdIssueSummary(w), jiraURL(w), w.Issue.Key)
}

func mdIssueLink(w *Webhook) string {
	return fmt.Sprintf("[%v](%v/browse/%v)", w.Issue.Key, jiraURL(w), w.Issue.Key)
}

func mdIssueHashtags(w *Webhook) string {
	s := "("
	if w.WebhookEvent == "jira:issue_created" {
		s += "#jira-new "
	}
	s += "#" + w.Issue.Key
	s += ")"
	return s
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

func mdUser(user *WebhookUser) string {
	if user == nil {
		return ""
	}
	return user.DisplayName
}

func jiraURL(w *Webhook) string {
	pos := strings.LastIndex(w.Issue.Self, "/rest/api")
	if pos < 0 {
		return ""
	}
	return w.Issue.Self[:pos]
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

func join(w *Webhook, functions ...func(w *Webhook) string) string {
	attrs := []string{}
	for _, f := range functions {
		attr := f(w)
		if attr != "" {
			attrs = append(attrs, attr)
		}
	}
	return strings.Join(attrs, ", ")
}

func mdBOLD(s string) string {
	return "**" + s + "**"
}
