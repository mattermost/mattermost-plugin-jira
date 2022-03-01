// See License for license information.
// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.

package main

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/v6/model"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

const (
	Nobody         = "_nobody_"
	commentDeleted = "comment_deleted"
	commentUpdated = "comment_updated"
	commentCreated = "comment_created"
)

type Webhook interface {
	Events() StringSet
	PostToChannel(p *Plugin, instanceID types.ID, channelID, fromUserID, subscriptionName string) (*model.Post, int, error)
	PostNotifications(p *Plugin, instanceID types.ID) ([]*model.Post, int, error)
}

type webhookField struct {
	name string
	id   string
	from string
	to   string
}

type webhook struct {
	*JiraWebhook
	eventTypes    StringSet
	headline      string
	text          string
	fields        []*model.SlackAttachmentField
	notifications []webhookUserNotification
	fieldInfo     webhookField
}

type webhookUserNotification struct {
	jiraUsername  string
	jiraAccountID string
	message       string
	postType      string
	commentSelf   string
}

func (wh *webhook) Events() StringSet {
	return wh.eventTypes
}

func (wh webhook) PostToChannel(p *Plugin, instanceID types.ID, channelID, fromUserID, subscriptionName string) (*model.Post, int, error) {
	if wh.headline == "" {
		return nil, http.StatusBadRequest, errors.Errorf("unsupported webhook")
	} else if p.getConfig().DisplaySubscriptionNameInNotifications && subscriptionName != "" {
		wh.headline = fmt.Sprintf("%s\nSubscription: **%s**", wh.headline, subscriptionName)
	}

	post := &model.Post{
		ChannelId: channelID,
		UserId:    fromUserID,
	}

	text := ""
	if wh.text != "" && !p.getConfig().HideDecriptionComment {
		text = p.replaceJiraAccountIds(instanceID, wh.text)
	}

	if text != "" || len(wh.fields) != 0 {
		model.ParseSlackAttachment(post, []*model.SlackAttachment{
			{
				// TODO is this supposed to be themed?
				Color:    "#95b7d0",
				Fallback: wh.headline,
				Pretext:  wh.headline,
				Text:     text,
				Fields:   wh.fields,
			},
		})
	} else {
		post.Message = wh.headline
	}

	_, appErr := p.API.CreatePost(post)
	if appErr != nil {
		return nil, appErr.StatusCode, appErr
	}

	return post, http.StatusOK, nil
}

func (wh *webhook) PostNotifications(p *Plugin, instanceID types.ID) ([]*model.Post, int, error) {
	if len(wh.notifications) == 0 {
		return nil, http.StatusOK, nil
	}

	// We will only send webhook events if we have a connected instance.
	instance, err := p.instanceStore.LoadInstance(instanceID)
	if err != nil {
		// This isn't an internal server error. There's just no instance installed.
		return nil, http.StatusOK, nil
	}

	posts := []*model.Post{}
	for _, notification := range wh.notifications {
		var mattermostUserID types.ID
		var err error

		// prefer accountId to username when looking up UserIds
		if notification.jiraAccountID != "" {
			mattermostUserID, err = p.userStore.LoadMattermostUserID(instance.GetID(), notification.jiraAccountID)
		} else {
			mattermostUserID, err = p.userStore.LoadMattermostUserID(instance.GetID(), notification.jiraUsername)
		}

		if err != nil {
			continue
		}

		// Check if the user has permissions.
		c, err2 := p.userStore.LoadConnection(instance.GetID(), mattermostUserID)
		if err2 != nil {
			// Not connected to Jira, so can't check permissions
			continue
		}

		client, err2 := instance.GetClient(c)
		if err2 != nil {
			p.errorf("PostNotifications: error while getting jiraClient, err: %v", err2)
			continue
		}
		// If this is a comment-related webhook, we need to check if they have permissions to read that.
		// Otherwise, check if they can view the issue.

		isCommentEvent := wh.Events().Intersection(commentEvents).Len() > 0
		if isCommentEvent {
			err = client.RESTGet(notification.commentSelf, nil, &struct{}{})
		} else {
			_, err = client.GetIssue(wh.Issue.ID, nil)
		}

		if err != nil {
			p.errorf("PostNotifications: failed to get self: %v", err)
			continue
		}

		notification.message = p.replaceJiraAccountIds(instance.GetID(), notification.message)

		post, err := p.CreateBotDMPost(instance.GetID(), mattermostUserID, notification.message, notification.postType)
		if err != nil {
			p.errorf("PostNotifications: failed to create notification post, err: %v", err)
			continue
		}
		posts = append(posts, post)
	}
	return posts, http.StatusOK, nil
}

func newWebhook(jwh *JiraWebhook, eventType string, format string, args ...interface{}) *webhook {
	return &webhook{
		JiraWebhook: jwh,
		eventTypes:  NewStringSet(eventType),
		headline:    jwh.mdUser() + " " + fmt.Sprintf(format, args...) + " " + jwh.mdKeySummaryLink(),
	}
}

func (p *Plugin) GetWebhookURL(jiraURL string, teamID, channelID string) (subURL, legacyURL string, err error) {
	cf := p.getConfig()

	instanceID, err := p.ResolveWebhookInstanceURL(jiraURL)
	if err != nil {
		return "", "", err
	}

	team, appErr := p.API.GetTeam(teamID)
	if appErr != nil {
		return "", "", appErr
	}

	channel, appErr := p.API.GetChannel(channelID)
	if appErr != nil {
		return "", "", appErr
	}

	v := url.Values{}
	v.Add("secret", cf.Secret)
	subURL = p.GetPluginURL() + instancePath(routeAPISubscribeWebhook, instanceID) + "?" + v.Encode()

	// For the legacy URL, add team and channel. Secret is already in the map.
	v.Add("team", team.Name)
	v.Add("channel", channel.Name)
	legacyURL = p.GetPluginURL() + instancePath(routeIncomingWebhook, instanceID) + "?" + v.Encode()

	return subURL, legacyURL, nil
}

func (wh *webhook) applyReporterNotification(p *Plugin, instanceID types.ID, reporter *jira.User) {
	if !wh.eventTypes.ContainsAny("event_created_comment") {
		return
	}

	jwhook := wh.JiraWebhook
	if reporter == nil ||
		(reporter.Name != "" && reporter.Name == jwhook.User.Name) ||
		(reporter.AccountID != "" && reporter.AccountID == jwhook.Comment.UpdateAuthor.AccountID) {
		return
	}

	if wh.checkNotificationAlreadyExist(reporter.Name, reporter.AccountID) {
		return
	}

	commentAuthor := mdUser(&jwhook.Comment.UpdateAuthor)

	commentMessage := fmt.Sprintf("%s **commented** on %s:\n>%s", commentAuthor, jwhook.mdKeySummaryLink(), jwhook.Comment.Body)

	c, err := wh.GetUserSetting(p, instanceID, reporter.Name, reporter.AccountID)
	if err != nil || c.Settings == nil || !c.Settings.ShouldReceiveReporterNotifications() {
		return
	}

	wh.notifications = append(wh.notifications, webhookUserNotification{
		jiraUsername:  reporter.Name,
		jiraAccountID: reporter.AccountID,
		message:       commentMessage,
		postType:      PostTypeComment,
		commentSelf:   jwhook.Comment.Self,
	})
}

func (wh *webhook) checkNotificationAlreadyExist(username, accountID string) bool {
	for _, val := range wh.notifications {
		if val.jiraUsername == username && val.jiraAccountID == accountID {
			return true
		}
	}

	return false
}

func (wh *webhook) GetUserSetting(p *Plugin, instanceID types.ID, jiraAccountID, jiraUsername string) (*Connection, error) {
	var err error
	instance, err := p.instanceStore.LoadInstance(instanceID)
	if err != nil {
		return nil, err
	}
	var mattermostUserID types.ID
	if jiraAccountID != "" {
		mattermostUserID, err = p.userStore.LoadMattermostUserID(instance.GetID(), jiraAccountID)
	} else {
		mattermostUserID, err = p.userStore.LoadMattermostUserID(instance.GetID(), jiraUsername)
	}

	if err != nil {
		return nil, err
	}

	c, err := p.userStore.LoadConnection(instanceID, mattermostUserID)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (s *ConnectionSettings) ShouldReceiveAssigneeNotifications() bool {
	if s.SendNotificationsForAssignee != nil {
		return *s.SendNotificationsForAssignee
	}

	// Check old setting for backwards compatibility
	return *s.Notifications
}

func (s *ConnectionSettings) ShouldReceiveReporterNotifications() bool {
	if s.SendNotificationsForReporter != nil {
		return *s.SendNotificationsForReporter
	}

	// Check old setting for backwards compatibility
	return *s.Notifications
}

func (s *ConnectionSettings) ShouldReceiveMentionNotifications() bool {
	if s.SendNotificationsForMention != nil {
		return *s.SendNotificationsForMention
	}

	// Check old setting for backwards compatibility
	return *s.Notifications
}

func (p *Plugin) getSubscriptionsWebhookURL(instanceID types.ID) string {
	cf := p.getConfig()
	v := url.Values{}
	v.Add("secret", cf.Secret)
	return p.GetPluginURL() + instancePath(routeAPISubscribeWebhook, instanceID) + "?" + v.Encode()
}
