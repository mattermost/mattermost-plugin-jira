// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/mattermost/mattermost-server/model"
)

const (
	mdRootStyle   = "## "
	mdUpdateStyle = "###### "
)

type parsed struct {
	*Webhook
	RawJSON  string
	headline string
	details  string
	edited   string
	style    string
}

func AsMarkdown(in io.Reader) (func(post *model.Post), error) {
	parsed, err := parse(in, func(w *Webhook) string {
		return w.mdIssueLongLink()
	})
	if err != nil {
		return nil, err
	}

	s := newMarkdownMessage(parsed)

	// Return a function that sets the Message on a post
	return func(post *model.Post) {
		post.Message = s
	}, nil
}

func newMarkdownMessage(parsed *parsed) string {
	if parsed.headline == "" {
		return ""
	}
	s := parsed.style + parsed.headline + "\n"
	if parsed.details != "" {
		s += parsed.details + "\n"
	}
	if parsed.edited != "" {
		s += parsed.edited + "\n"
	}
	return s
}

func parse(in io.Reader, linkf func(w *Webhook) string) (*parsed, error) {
	bb, err := ioutil.ReadAll(in)
	if err != nil {
		return nil, err
	}

	webhook := Webhook{}
	err = json.Unmarshal(bb, &webhook)
	if err != nil {
		return nil, err
	}
	if webhook.WebhookEvent == "" {
		return nil, fmt.Errorf("No webhook event")
	}

	parsed := parsed{
		Webhook: &webhook,
	}
	parsed.RawJSON = string(bb)
	if linkf == nil {
		linkf = func(w *Webhook) string {
			return parsed.mdIssueLink()
		}
	}

	headline := ""
	user := &parsed.User
	parsed.style = mdUpdateStyle
	issue := parsed.mdIssueType() + " " + linkf(parsed.Webhook)
	switch parsed.WebhookEvent {
	case "jira:issue_created":
		parsed.style = mdRootStyle
		headline = fmt.Sprintf("created %v", issue)
		parsed.details = parsed.mdIssueCreatedDetails()
		parsed.edited = parsed.mdIssueDescription()
	case "jira:issue_deleted":
		headline = fmt.Sprintf("deleted %v", issue)
	case "jira:issue_updated":
		switch parsed.IssueEventTypeName {
		case "issue_assigned":
			headline = fmt.Sprintf("assigned %v to %v", issue, parsed.mdIssueAssignee())

		case "issue_updated", "issue_generic":
			// edited summary, description, updated priority, status, etc.
			headline, parsed.edited = parsed.fromChangeLog(issue)
		}
	case "comment_deleted":
		user = &parsed.Comment.UpdateAuthor
		headline = fmt.Sprintf("removed a comment from %v", issue)

	case "comment_updated":
		user = &parsed.Comment.UpdateAuthor
		headline = fmt.Sprintf("edited a comment in %v", issue)
		parsed.edited = truncate(parsed.Comment.Body, 3000)

	case "comment_created":
		user = &parsed.Comment.UpdateAuthor
		headline = fmt.Sprintf("commented on %v", issue)
		parsed.edited = truncate(parsed.Comment.Body, 3000)
	}
	if headline == "" {
		return nil, fmt.Errorf("Unsupported webhook")
	}
	parsed.headline = fmt.Sprintf("%v %v %v", mdUser(user), headline, parsed.mdIssueHashtags())
	return &parsed, nil
}

func (p *parsed) fromChangeLog(issue string) (string, string) {
	for _, item := range p.ChangeLog.Items {
		to := item.ToString
		from := item.FromString
		switch {
		case item.Field == "resolution" && to == "" && from != "":
			return fmt.Sprintf("reopened %v", issue), ""

		case item.Field == "resolution" && to != "" && from == "":
			return fmt.Sprintf("resolved %v", issue), ""

		case item.Field == "status" && to == "In Progress":
			return fmt.Sprintf("started working on %v", issue), ""

		case item.Field == "priority" && item.From > item.To:
			return fmt.Sprintf("raised priority of %v to %v", issue, to), ""

		case item.Field == "priority" && item.From < item.To:
			return fmt.Sprintf("lowered priority of %v to %v", issue, to), ""

		case item.Field == "summary":
			return fmt.Sprintf("renamed %v to %v", issue, p.mdIssueSummary()), ""

		case item.Field == "description":
			return fmt.Sprintf("edited description of %v", issue),
				p.mdIssueDescription()

		case item.Field == "Sprint" && len(to) > 0:
			return fmt.Sprintf("moved %v to %v", issue, to), ""

		case item.Field == "Rank" && len(to) > 0:
			return fmt.Sprintf("%v %v", strings.ToLower(to), issue), ""

		case item.Field == "Attachment":
			return fmt.Sprintf("%v %v", mdAddRemove(from, to, "attached", "removed attachments"), issue), ""

		case item.Field == "labels":
			return fmt.Sprintf("%v %v", mdAddRemove(from, to, "added labels", "removed labels"), issue), ""
		}
	}
	return "", ""
}

func (w *Webhook) mdIssueCreatedDetails() string {
	attrs := []string{}
	for _, a := range []string{
		w.mdIssuePriority(),
		w.mdIssueAssignedTo(),
		w.mdIssueReportedBy(),
		w.mdIssueLabels(),
	} {
		if a != "" {
			attrs = append(attrs, a)
		}
	}
	s := strings.Join(attrs, ", ")
	return s
}

func (w *Webhook) mdIssueSummary() string {
	return truncate(w.Issue.Fields.Summary, 80)
}

func (w *Webhook) mdIssueDescription() string {
	return truncate(w.Issue.Fields.Description, 3000)
}

func (w *Webhook) mdIssueAssignee() string {
	if w.Issue.Fields.Assignee == nil {
		return "_nobody_"
	}
	return mdUser(w.Issue.Fields.Assignee)
}

func (w *Webhook) mdIssueAssignedTo() string {
	if w.Issue.Fields.Assignee == nil {
		return ""
	}
	return "Assigned to: " + mdBOLD(w.mdIssueAssignee())
}

func (w *Webhook) mdIssueReportedBy() string {
	if w.Issue.Fields.Reporter == nil {
		return ""
	}
	return "Reported by: " + mdBOLD(mdUser(w.Issue.Fields.Reporter))
}

func (w *Webhook) mdIssueLabels() string {
	if len(w.Issue.Fields.Labels) == 0 {
		return ""
	}
	return "Labels: " + strings.Join(w.Issue.Fields.Labels, ",")
}

func (w *Webhook) mdIssuePriority() string {
	return "Priority: " + mdBOLD(w.Issue.Fields.Priority.Name)
}

func (w *Webhook) mdIssueType() string {
	return strings.ToLower(w.Issue.Fields.IssueType.Name)
}

func (w *Webhook) mdIssueLongLink() string {
	return fmt.Sprintf("[%v: %v](%v/browse/%v)", w.Issue.Key, w.mdIssueSummary(), w.jiraURL(), w.Issue.Key)
}

func (w *Webhook) mdIssueLink() string {
	return fmt.Sprintf("[%v](%v/browse/%v)", w.Issue.Key, w.jiraURL(), w.Issue.Key)
}

func (w *Webhook) mdIssueHashtags() string {
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

func mdBOLD(s string) string {
	return "**" + s + "**"
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
func (w *Webhook) jiraURL() string {
	pos := strings.LastIndex(w.Issue.Self, "/rest/api")
	if pos < 0 {
		return ""
	}
	return w.Issue.Self[:pos]
}
