// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/andygrunwald/go-jira"

	"github.com/mattermost/mattermost-server/model"
)

const (
	mdRootStyle   = "## "
	mdUpdateStyle = "###### "
)

func AsMarkdown(in io.Reader) (func(post *model.Post), error) {
	parsed, err := parse(in, func(w *JIRAWebhook) string {
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

func newMarkdownMessage(parsed *parsedJIRAWebhook) string {
	if parsed.headline == "" {
		return ""
	}
	s := parsed.style + parsed.headline + "\n"
	if parsed.details != "" {
		s += parsed.details + "\n"
	}
	if parsed.text != "" {
		s += parsed.text + "\n"
	}
	return s
}

func (w *JIRAWebhook) mdIssueCreatedDetails() string {
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

func (w *JIRAWebhook) mdIssueSummary() string {
	return truncate(w.Issue.Fields.Summary, 80)
}

func (w *JIRAWebhook) mdIssueDescription() string {
	return truncate(w.Issue.Fields.Description, 3000)
}

func (w *JIRAWebhook) mdIssueAssignee() string {
	if w.Issue.Fields.Assignee == nil {
		return "_nobody_"
	}
	return mdUser(w.Issue.Fields.Assignee)
}

func (w *JIRAWebhook) mdIssueAssignedTo() string {
	if w.Issue.Fields.Assignee == nil {
		return ""
	}
	return "Assigned to: " + mdBOLD(w.mdIssueAssignee())
}

func (w *JIRAWebhook) mdIssueReportedBy() string {
	if w.Issue.Fields.Reporter == nil {
		return ""
	}
	return "Reported by: " + mdBOLD(mdUser(w.Issue.Fields.Reporter))
}

func (w *JIRAWebhook) mdIssueLabels() string {
	if len(w.Issue.Fields.Labels) == 0 {
		return ""
	}
	return "Labels: " + strings.Join(w.Issue.Fields.Labels, ",")
}

func (w *JIRAWebhook) mdIssuePriority() string {
	return "Priority: " + mdBOLD(w.Issue.Fields.Priority.Name)
}

func (w *JIRAWebhook) mdIssueType() string {
	return strings.ToLower(w.Issue.Fields.IssueType.Name)
}

func (w *JIRAWebhook) mdIssueLongLink() string {
	return fmt.Sprintf("[%v: %v](%v/browse/%v)", w.Issue.Key, w.mdIssueSummary(), w.jiraURL(), w.Issue.Key)
}

func (w *JIRAWebhook) mdIssueLink() string {
	return fmt.Sprintf("[%v](%v/browse/%v)", w.Issue.Key, w.jiraURL(), w.Issue.Key)
}

func (w *JIRAWebhook) mdIssueHashtags() string {
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

func mdUser(user *jira.User) string {
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
