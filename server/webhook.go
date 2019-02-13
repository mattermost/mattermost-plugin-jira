// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"fmt"
	"strings"
	// "github.com/mattermost/mattermost-server/model"
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
	md_RootStyle   = "## "
	md_UpdateStyle = "###### "
)

var formatters = map[string]func(w *Webhook) string{
	"jira:issue_created": mdIssueCreated,
	"jira:issue_updated": mdIssueUpdated,
	"comment_created":    mdComment,
	"comment_updated":    mdComment,
	"comment_deleted":    mdComment,
}

func (w *Webhook) Markdown() string {
	f := formatters[w.WebhookEvent]
	if f == nil {
		f = mdDefault
	}
	return f(w)
}

func mdDefault(w *Webhook) string {
	return ""
}

func mdIssueCreated(w *Webhook) string {
	s := md_RootStyle
	s += fmt.Sprintf("%v created a %v %v", md_User(&w.User), md_IssueType(w), md_IssueLongLink(w))
	s += "\n"

	s += join(w,
		md_IssuePriority,
		md_IssueAssignedTo,
		md_IssueReportedBy,
		md_IssueLabels,
		md_IssueHashtags,
	)
	s += "\n"
	s += "\n"

	s += md_IssueDescription(w)

	return s
}

func mdIssueUpdated(w *Webhook) string {
	s := md_UpdateStyle
	headline := ""
	extra := ""

	switch w.IssueEventTypeName {
	case "issue_assigned":
		assignee := "_nobody_"
		if w.Issue.Fields.Assignee != nil {
			assignee = md_User(w.Issue.Fields.Assignee)
		}
		headline = fmt.Sprintf("assigned %v to %v", md_IssueLongLink(w), assignee)

	case "issue_updated",
		"issue_generic":
		// edited summary, description, updated priority, status, etc.
		headline = md_HeadlineFromChangeLog(w)
		if w.showDescription {
			extra = md_IssueDescription(w)
		}
	}
	if headline == "" {
		// return fmt.Sprintf("```%v```\n", w.RawJSON)
		return ""
	}

	s += md_User(&w.User) + " " + headline + " " + md_IssueHashtags(w)
	s += "\n"

	if extra != "" {
		s += extra
	}
	s += "\n"

	return s
}

func mdComment(w *Webhook) string {
	s := md_UpdateStyle
	headline := ""
	extra := ""

	switch w.WebhookEvent {
	case "comment_deleted":
		headline = fmt.Sprintf("removed a comment from %v", md_IssueLongLink(w))

	case "comment_updated":
		headline = fmt.Sprintf("edited a comment in %v", md_IssueLongLink(w))
		extra = w.Comment.Body

	case "comment_created":
		headline = fmt.Sprintf("commented on %v", md_IssueLongLink(w))
		extra = w.Comment.Body

	}
	if headline == "" {
		return ""
	}

	s += md_User(&w.Comment.UpdateAuthor) + " " + headline + " " + md_IssueHashtags(w)
	s += "\n"
	if extra != "" {
		s += extra
	}
	s += "\n"
	return s
}

func md_HeadlineFromChangeLog(w *Webhook) string {
	for _, item := range w.ChangeLog.Items {
		to := item.ToString
		from := item.FromString
		switch {
		case item.Field == "resolution" && to == "" && from != "":
			return fmt.Sprintf("reopened %v", md_IssueLongLink(w))

		case item.Field == "resolution" && to != "" && from == "":
			return fmt.Sprintf("resolved %v", md_IssueLongLink(w))

		case item.Field == "status" && to == "In Progress":
			return fmt.Sprintf("started working on %v", md_IssueLongLink(w))

		case item.Field == "priority" && item.From > item.To:
			return fmt.Sprintf("raised priority of %v to %v", md_IssueLongLink(w), to)

		case item.Field == "priority" && item.From < item.To:
			return fmt.Sprintf("lowered priority of %v to %v", md_IssueLongLink(w), to)

		case item.Field == "summary":
			return fmt.Sprintf("renamed %v to %v", md_IssueLink(w), md_IssueSummary(w))

		case item.Field == "description":
			w.showDescription = true
			return fmt.Sprintf("edited description of %v", md_IssueLongLink(w))

		case item.Field == "Sprint" && len(to) > 0:
			return fmt.Sprintf("moved %v to %v", md_IssueLongLink(w), to)

		case item.Field == "Rank" && len(to) > 0:
			return fmt.Sprintf("%v %v", strings.ToLower(to), md_IssueLongLink(w))

		case item.Field == "Attachment":
			return fmt.Sprintf("%v %v", md_AddRemove(from, to, "attached", "removed attachments"), md_IssueLongLink(w))

		case item.Field == "labels":
			return fmt.Sprintf("%v %v", md_AddRemove(from, to, "added labels", "removed labels"), md_IssueLongLink(w))

		default:
			// return fmt.Sprintf("updated %v from %v to %v on %v", item.Field, from, to, md_IssueLongLink(w))
		}
	}
	return ""
}

func md_IssueSummary(w *Webhook) string {
	return truncate(w.Issue.Fields.Summary, 80)
}

func md_IssueDescription(w *Webhook) string {
	return fmt.Sprintf("```\n%v\n```", truncate(w.Issue.Fields.Description, 3000))
}

func md_IssueAssignedTo(w *Webhook) string {
	if w.Issue.Fields.Assignee == nil {
		return ""
	}
	return "Assigned to: " + mdBOLD(md_User(w.Issue.Fields.Assignee))
}

func md_IssueReportedBy(w *Webhook) string {
	if w.Issue.Fields.Reporter == nil {
		return ""
	}
	return "Reported by: " + mdBOLD(md_User(w.Issue.Fields.Reporter))
}

func md_IssueLabels(w *Webhook) string {
	if len(w.Issue.Fields.Labels) == 0 {
		return ""
	}
	return "Labels: " + strings.Join(w.Issue.Fields.Labels, ",")
}

func md_IssuePriority(w *Webhook) string {
	return "Priority: " + mdBOLD(w.Issue.Fields.Priority.Name)
}

func md_IssueType(w *Webhook) string {
	return strings.ToLower(w.Issue.Fields.IssueType.Name)
}

func md_IssueLongLink(w *Webhook) string {
	return fmt.Sprintf("[%v: %v](%v/browse/%v)", w.Issue.Key, md_IssueSummary(w), jiraURL(w), w.Issue.Key)
}

func md_IssueLink(w *Webhook) string {
	return fmt.Sprintf("[%v](%v/browse/%v)", w.Issue.Key, jiraURL(w), w.Issue.Key)
}

func md_IssueHashtags(w *Webhook) string {
	s := "("
	if w.WebhookEvent == "jira:issue_created" {
		s += "#jira-new "
	}
	s += "#" + w.Issue.Key
	s += ")"
	return s
}

func md_AddRemove(from, to, add, remove string) string {
	added := md_Diff(from, to)
	removed := md_Diff(to, from)
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

func md_Diff(from, to string) string {
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
	for s, _ := range toMap {
		if !fromMap[s] {
			added = append(added, s)
		}
	}

	return strings.Join(added, ",")
}

func md_User(user *WebhookUser) string {
	if user == nil {
		return ""
	}
	return user.DisplayName
}

func md_UserWithAvatar(user *WebhookUser) string {
	if user == nil {
		return ""
	}

	avatar := ""
	if len(user.AvatarURLs) > 0 {
		avatar = fmt.Sprintf("![](%v) ", user.AvatarURLs["24x24"])
	}
	return avatar + user.DisplayName
}

func jiraURL(w *Webhook) string {
	pos := strings.LastIndex(w.Issue.Self, "/rest/api")
	if pos < 0 {
		return ""
	}
	return w.Issue.Self[:pos]
}

func truncate(s string, max int) string {
	if len(s) < max {
		return s
	}
	if max > 3 {
		max -= 3
	}
	return s[:max] + "..."
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
