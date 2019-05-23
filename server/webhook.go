// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/andygrunwald/go-jira"

	"github.com/mattermost/mattermost-server/model"
)

const (
	jiraCommentPostType = "custom_jira_comment"
	jiraMentionPostType = "custom_jira_mention"
)

type JIRAWebhookIssue struct {
	Self   string
	Key    string
	Fields struct {
		Assignee    *jira.User
		Reporter    *jira.User
		Summary     string
		Description string
		Priority    *struct {
			Id      string
			Name    string
			IconURL string
		}
		IssueType struct {
			Id      string
			Name    string
			IconURL string
		}
		Resolution *struct {
			Id string
		}
		Status struct {
			Id string
		}
		Labels  []string
		Project struct {
			Key string
		}
	}
}

type JIRAWebhook struct {
	WebhookEvent string
	Issue        JIRAWebhookIssue
	User         jira.User
	Comment      struct {
		Body         string
		UpdateAuthor jira.User
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

type parsedJIRAWebhook struct {
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

func httpWebhook(p *Plugin, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodPost {
		return http.StatusMethodNotAllowed,
			fmt.Errorf("Request: " + r.Method + " is not allowed, must be POST")
	}
	// TODO add JWT support
	cfg := p.getConfig()
	if cfg.Secret == "" || cfg.UserName == "" {
		return http.StatusForbidden, fmt.Errorf("Jira plugin not configured correctly; must provide Secret and UserName")
	}

	secret := r.FormValue("secret")
	for {
		if subtle.ConstantTimeCompare([]byte(secret), []byte(cfg.Secret)) == 1 {
			break
		}

		unescaped, _ := url.QueryUnescape(secret)
		if unescaped == secret {
			return http.StatusForbidden,
				fmt.Errorf("Request URL: secret did not match")
		}
		secret = unescaped
	}

	parsed, err := parse(r.Body, nil)
	if err != nil {
		return http.StatusBadRequest, err
	}

	// Post the event to the subscribed channel
	statusCode, err := p.postEvent(r, cfg, parsed)
	if err != nil {
		return statusCode, err
	}

	// Notify any affected users using a direct channel
	err = p.handleNotifications(parsed)
	if err != nil {
		p.errorf("httpWebhook, handleNotifications: %v", err)
		return http.StatusBadRequest, err
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

func parse(in io.Reader, linkf func(w *JIRAWebhook) string) (*parsedJIRAWebhook, error) {
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

	parsed := parsedJIRAWebhook{
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

func (p *parsedJIRAWebhook) fromChangeLog(issue string) (string, string) {
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

// postEvent posts the event to the channel that subscribed to it
func (p *Plugin) postEvent(r *http.Request, cfg config, parsed *parsedJIRAWebhook) (int, error) {
	teamName := r.FormValue("team")
	if teamName == "" {
		return http.StatusBadRequest,
			fmt.Errorf("Request URL: team is empty")
	}

	channelId := r.FormValue("channel")
	if channelId == "" {
		return http.StatusBadRequest,
			fmt.Errorf("Request URL: channel is empty")
	}

	user, appErr := p.API.GetUserByUsername(cfg.UserName)
	if appErr != nil {
		return appErr.StatusCode, fmt.Errorf(appErr.Message)
	}

	channel, appErr := p.API.GetChannelByNameForTeamName(teamName, channelId, false)
	if appErr != nil {
		return appErr.StatusCode, fmt.Errorf(appErr.Message)
	}

	initPost := AsSlackAttachment(parsed)

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

// handleNotifications notifies users involved in the event, if they've enabled notifications
func (p *Plugin) handleNotifications(parsed *parsedJIRAWebhook) error {
	// This bothers me, to do this for every webhook event...
	ji, err := p.LoadCurrentJIRAInstance()
	if err != nil {
		// It won't break anything if we can't find the Jira Instance here -- we just can't notify anyone.
		return nil
		// Alternative:
		//return errors.Errorf("Failed to load current Jira instance: %v", err)
	}

	switch parsed.JIRAWebhook.WebhookEvent {
	case "jira:issue_updated", "jira:issue_created":
		return p.handleIssueUpdatedNotifications(ji, parsed)
	case "comment_created":
		return p.handleCommentCreatedNotifications(ji, parsed)
	default:
		return nil
	}
}

func (p *Plugin) handleIssueUpdatedNotifications(ji Instance, parsed *parsedJIRAWebhook) error {
	for _, change := range parsed.ChangeLog.Items {
		if change.Field != "assignee" || change.ToString == "" {
			return nil
		}

		if parsed.assigneeUsername == "" {
			return nil
		}

		mattermostUserId, err := p.LoadMattermostUserId(ji, parsed.assigneeUsername)
		if err != nil {
			return err
		}

		message := "[%s](%s) assigned you to [%s](%s)"
		err = p.CreateBotDMPost(ji, mattermostUserId, fmt.Sprintf(message, parsed.authorDisplayName, parsed.authorURL, parsed.issueKey, parsed.issueURL), "custom_jira_assigned")
		if err != nil {
			return errors.Errorf("handleIssueUpdatedNotification failed: %v", err)
		}
	}
	return nil
}

func (p *Plugin) handleCommentCreatedNotifications(ji Instance, parsed *parsedJIRAWebhook) error {
	if parsed.authorUsername == "" {
		return nil
	}

	for _, u := range parseJIRAUsernamesFromText(parsed.Comment.Body) {
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
			p.errorf("handleCommentCreatedNotifications, LoadMattermostUserId: %v", err)
			continue
		}

		err = p.CreateBotDMPost(ji, mattermostUserId,
			fmt.Sprintf("[%s](%s) mentioned you on [%s](%s):\n>%s",
				parsed.authorDisplayName, parsed.authorURL, parsed.issueKey, parsed.issueURL, parsed.text),
			jiraMentionPostType)
		if err != nil {
			p.errorf("handleCommentCreatedNotifications, CreateBotDMPost: %v", err)
			continue
		}
	}

	if parsed.assigneeUsername == parsed.authorUsername {
		return nil
	}

	mattermostUserId, err := p.LoadMattermostUserId(ji, parsed.assigneeUsername)
	if err != nil {
		return err
	}

	err = p.CreateBotDMPost(ji, mattermostUserId,
		fmt.Sprintf("[%s](%s) commented on [%s](%s):\n>%s",
			parsed.authorDisplayName, parsed.authorURL, parsed.issueKey, parsed.issueURL, parsed.text),
		jiraCommentPostType)
	if err != nil {
		return errors.Errorf("handleCommentCreatedNotifications, CreateBotDMPost: %v", err)
	}

	return nil
}

func (p *Plugin) GetWebhookURL(teamId, channelId string) (string, error) {
	cf := p.getConfig()

	team, appErr := p.API.GetTeam(teamId)
	if appErr != nil {
		return "", appErr
	}

	channel, appErr := p.API.GetChannel(channelId)
	if appErr != nil {
		return "", appErr
	}

	v := url.Values{}
	secret, _ := url.QueryUnescape(cf.Secret)
	v.Add("secret", secret)
	v.Add("team", team.Name)
	v.Add("channel", channel.Name)
	return p.GetPluginURL() + "/" + routeIncomingWebhook + "?" + v.Encode(), nil
}
