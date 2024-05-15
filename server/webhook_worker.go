// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	jira "github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"

	"fmt"
)

type webhookWorker struct {
	id        int
	p         *Plugin
	workQueue <-chan *webhookMessage
}

type webhookMessage struct {
	InstanceID types.ID
	Data       []byte
}

func (ww webhookWorker) work() {
	for msg := range ww.workQueue {
		err := ww.process(msg)
		if err != nil {
			if errors.Is(err, errWebhookeventUnsupported) {
				ww.p.debugf("WebhookWorker id: %d, error processing, err: %v", ww.id, err)
			} else {
				ww.p.errorf("WebhookWorker id: %d, error processing, err: %v", ww.id, err)
			}
		}
	}
}

func isCommentRelatedWebhook(wh Webhook) bool {
	return wh.Events().Intersection(commentEvents).Len() > 0
}

func (ww webhookWorker) getCommentVisibility(msg *webhookMessage, v *webhook) (string, error) {
	mattermostUserID, err := ww.p.userStore.LoadMattermostUserID(msg.InstanceID, v.JiraWebhook.Comment.Author.AccountID)
	if err != nil {
		ww.p.API.LogDebug("Comment author is not connected with Mattermost", "Error", err.Error())
		return "", err
	}

	client, _, _, err := ww.p.getClient(msg.InstanceID, mattermostUserID)
	if err != nil {
		return "", err
	}

	comment := jira.Comment{}
	if err = client.RESTGet(fmt.Sprintf("2/issue/%s/comment/%s", v.JiraWebhook.Issue.ID, v.JiraWebhook.Comment.ID), nil, &comment); err != nil {
		return "", err
	}

	return comment.Visibility.Value, nil
}

func (ww webhookWorker) process(msg *webhookMessage) (err error) {
	defer func() {
		if err == ErrWebhookIgnored {
			// ignore ErrWebhookIgnored - from here up it's a success
			err = nil
		}
	}()

	wh, err := ParseWebhook(msg.Data)
	if err != nil {
		return err
	}

	if _, _, err = wh.PostNotifications(ww.p, msg.InstanceID); err != nil {
		ww.p.errorf("WebhookWorker id: %d, error posting notifications, err: %v", ww.id, err)
	}

	v := wh.(*webhook)
	if err = v.JiraWebhook.expandIssue(ww.p, msg.InstanceID); err != nil {
		return err
	}

	visibilityAttribute := ""
	if isCommentRelatedWebhook(wh) {
		if visibilityAttribute, err = ww.getCommentVisibility(msg, v); err != nil {
			return err
		}
	}

	channelsSubscribed, err := ww.p.getChannelsSubscribed(v, msg.InstanceID, visibilityAttribute)
	if err != nil {
		return err
	}

	botUserID := ww.p.getUserID()
	for _, channelSubscribed := range channelsSubscribed {
		if _, _, err1 := wh.PostToChannel(ww.p, msg.InstanceID, channelSubscribed.ChannelID, botUserID, channelSubscribed.Name); err1 != nil {
			ww.p.errorf("WebhookWorker id: %d, error posting to channel, err: %v", ww.id, err1)
		}
	}

	return nil
}
