// See License for license information.
// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.

package main

import (
	"fmt"
	"net/http"

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

const (
	recipientTypeAssignee = "assignee"
	recipientTypeReporter = "reporter"
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
	recipientType string
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

		post, err := p.CreateBotDMPost(instance.GetID(), mattermostUserID, notification.message, notification.postType, notification.recipientType)
		if err != nil {
			p.errorf("PostNotifications: failed to create notification post, err: %v", err)
			continue
		}
		posts = append(posts, post)
	}
	return posts, http.StatusOK, nil
}

func (wh *webhook) checkIssueWatchers(p *Plugin, instanceID types.ID) {
	instance, err := p.instanceStore.LoadInstance(instanceID)
	if err != nil && instance == nil {
		// This isn't an internal server error. There's just no instance installed.
		return
	}

	ci, ok := instance.(*cloudInstance)
	if !ok {
		return
	}
	client, err := ci.getClientForBot()
	// client, err := instance.GetClient(c)
	if err != nil {
		return
	}

	watcherUsers, resp, err := client.Issue.GetWatchers(wh.Issue.ID)
	if err != nil {
		err = userFriendlyJiraError(resp, err)
		return
	}

	for _, watcherUser := range *watcherUsers {
		postType := ""
		message := ""
		var shouldNotReceiveNotification bool
		for _, notification := range wh.notifications {
			if notification.jiraAccountID == watcherUser.AccountID || (notification.jiraUsername != "" && notification.jiraUsername == watcherUser.Name) {
				shouldNotReceiveNotification = true
				break
			}
			postType = notification.postType
			message = notification.message
		}
		if shouldNotReceiveNotification {
			continue
		}
		whUserNotification := webhookUserNotification{
			jiraUsername:  watcherUser.Name,
			jiraAccountID: watcherUser.AccountID,
			message:       message,
			postType:      postType,
			commentSelf:   wh.JiraWebhook.Comment.Self,
		}

		wh.notifications = append(wh.notifications, whUserNotification)
	}
}

func (wh *webhook) applyReporterNotification(reporter *jira.User) {
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

	wh.notifications = append(wh.notifications, webhookUserNotification{
		jiraUsername:  reporter.Name,
		jiraAccountID: reporter.AccountID,
		message:       commentMessage,
		postType:      PostTypeComment,
		commentSelf:   jwhook.Comment.Self,
		recipientType: recipientTypeReporter,
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
