// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/mattermost/mattermost-server/model"
)

type JIRAWebhookUser struct {
	AccountId    string
	Self         string
	Name         string
	Key          string
	EmailAddress string
	AvatarURLs   map[string]string
	DisplayName  string
	Active       bool
	TimeZone     string
}

type JIRAWebhookIssue struct {
	Self   string
	Key    string
	Fields struct {
		Assignee    *JIRAWebhookUser
		Reporter    *JIRAWebhookUser
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

type JIRAWebhook struct {
	WebhookEvent string
	Issue        JIRAWebhookIssue
	User         JIRAWebhookUser
	Comment      struct {
		Body         string
		UpdateAuthor JIRAWebhookUser
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
}

type parsed struct {
	*JIRAWebhook
	RawJSON           string
	headline          string
	details           string
	text              string
	style             string
	authorDisplayName string
	authorUsername    string
	authorURL         string
	assigneeUsername  string
	issueKey          string
	issueURL          string
}

type notifier interface {
	notify(ji Instance, parsed *parsed, text string)
}

func (p *Plugin) handleHTTPWebhook(w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodPost {
		return http.StatusMethodNotAllowed,
			fmt.Errorf("Request: " + r.Method + " is not allowed, must be POST")
	}
	// TODO add JWT support
	config := p.getConfig()
	if config.Secret == "" || config.UserName == "" {
		return http.StatusForbidden, fmt.Errorf("JIRA plugin not configured correctly; must provide Secret and UserName")
	}

	err := r.ParseForm()
	if err != nil {
		return http.StatusBadRequest, err
	}
	if subtle.ConstantTimeCompare([]byte(r.Form.Get("secret")), []byte(config.Secret)) != 1 {
		return http.StatusForbidden,
			fmt.Errorf("Request URL: secret did not match")
	}

	teamName := r.Form.Get("team")
	if teamName == "" {
		return http.StatusBadRequest,
			fmt.Errorf("Request URL: team is empty")
	}
	channelId := r.Form.Get("channel")
	if channelId == "" {
		return http.StatusBadRequest,
			fmt.Errorf("Request URL: channel is empty")
	}

	user, appErr := p.API.GetUserByUsername(config.UserName)
	if appErr != nil {
		return appErr.StatusCode, fmt.Errorf(appErr.Message)
	}

	channel, appErr := p.API.GetChannelByNameForTeamName(teamName, channelId, false)
	if appErr != nil {
		return appErr.StatusCode, fmt.Errorf(appErr.Message)
	}

	ji, err := p.LoadCurrentJIRAInstance()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	initPost, err := AsSlackAttachment(r.Body, p, ji)
	if err != nil {
		return http.StatusBadRequest, err
	}

	post := &model.Post{
		ChannelId: channel.Id,
		UserId:    user.Id,
		Props: map[string]interface{}{
			"from_webhook":  "true",
			"use_user_icon": "true",
		},
	}
	initPost(post)

	_, appErr = p.API.CreatePost(post)
	if appErr != nil {
		return appErr.StatusCode, fmt.Errorf(appErr.Message)
	}

	return http.StatusOK, nil
}

func (w *JIRAWebhook) jiraURL() string {
	pos := strings.LastIndex(w.Issue.Self, "/rest/api")
	if pos < 0 {
		return ""
	}
	return w.Issue.Self[:pos]
}

func parse(in io.Reader, linkf func(w *JIRAWebhook) string) (*parsed, error) {
	bb, err := ioutil.ReadAll(in)
	if err != nil {
		return nil, err
	}

	webhook := JIRAWebhook{}
	err = json.Unmarshal(bb, &webhook)
	if err != nil {
		return nil, err
	}
	if webhook.WebhookEvent == "" {
		return nil, fmt.Errorf("No webhook event")
	}

	parsed := parsed{
		JIRAWebhook: &webhook,
	}
	parsed.RawJSON = string(bb)
	if linkf == nil {
		linkf = func(w *JIRAWebhook) string {
			return parsed.mdIssueLink()
		}
	}

	headline := ""
	user := &parsed.User
	parsed.style = mdUpdateStyle
	issue := parsed.mdIssueType() + " " + linkf(parsed.JIRAWebhook)
	switch parsed.WebhookEvent {
	case "jira:issue_created":
		parsed.style = mdRootStyle
		headline = fmt.Sprintf("created %v", issue)
		parsed.details = parsed.mdIssueCreatedDetails()
		parsed.text = parsed.mdIssueDescription()
	case "jira:issue_deleted":
		headline = fmt.Sprintf("deleted %v", issue)
	case "jira:issue_updated":
		switch parsed.IssueEventTypeName {
		case "issue_assigned":
			headline = fmt.Sprintf("assigned %v to %v", issue, parsed.mdIssueAssignee())

		case "issue_updated", "issue_generic":
			// text summary, description, updated priority, status, etc.
			headline, parsed.text = parsed.fromChangeLog(issue)
		}
	case "comment_deleted":
		user = &parsed.Comment.UpdateAuthor
		headline = fmt.Sprintf("removed a comment from %v", issue)

	case "comment_updated":
		user = &parsed.Comment.UpdateAuthor
		headline = fmt.Sprintf("edited a comment in %v", issue)
		parsed.text = truncate(parsed.Comment.Body, 3000)

	case "comment_created":
		user = &parsed.Comment.UpdateAuthor
		headline = fmt.Sprintf("commented on %v", issue)
		parsed.text = truncate(parsed.Comment.Body, 3000)
	}
	if headline == "" {
		return nil, fmt.Errorf("Unsupported webhook data: %v", parsed.WebhookEvent)
	}
	parsed.headline = fmt.Sprintf("%v %v %v", mdUser(user), headline, parsed.mdIssueHashtags())

	parsed.authorDisplayName = user.DisplayName
	parsed.authorUsername = user.Name
	parsed.authorURL = getUserURL(&parsed.Issue, user)
	if parsed.Issue.Fields.Assignee != nil {
		parsed.assigneeUsername = parsed.Issue.Fields.Assignee.Name
	}
	parsed.issueKey = parsed.Issue.Key
	parsed.issueURL = getIssueURL(&parsed.Issue)

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

		case item.Field == "status" && to == "Backlog":
			return fmt.Sprintf("moved %v to backlog", issue), ""

		case item.Field == "status" && to == "In Progress":
			return fmt.Sprintf("started working on %v", issue), ""

		case item.Field == "status" && to == "Selected for Development":
			return fmt.Sprintf("selected %v for development", issue), ""

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

func (p *Plugin) notify(ji Instance, parsed *parsed, text string) {
	if parsed.authorUsername == "" {
		return
	}

	for _, u := range parseJIRAUsernamesFromText(text) {
		// don't mention the author of the text
		if u == parsed.authorUsername {
			continue
		}
		// assignee gets a special notice
		if u == parsed.assigneeUsername {
			continue
		}

		mattermostUserId, err := p.LoadMattermostUserId(ji, u)
		if err != nil {
			continue
		}

		p.CreateBotDMPost(mattermostUserId,
			fmt.Sprintf("[%s](%s) mentioned you on [%s](%s):\n>%s",
				parsed.authorDisplayName, parsed.authorURL, parsed.issueKey, parsed.issueURL, text),
			"custom_jira_mention")
	}

	if parsed.assigneeUsername == parsed.authorUsername {
		return
	}

	mattermostUserId, err := p.LoadMattermostUserId(ji, parsed.assigneeUsername)
	if err != nil {
		return
	}

	p.CreateBotDMPost(mattermostUserId,
		fmt.Sprintf("[%s](%s) commented on [%s](%s):\n>%s",
			parsed.authorDisplayName, parsed.authorURL, parsed.issueKey, parsed.issueURL, text),
		"custom_jira_comment")
}
