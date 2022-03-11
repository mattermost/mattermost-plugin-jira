// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
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
			ww.p.errorf("WebhookWorker id: %d, error processing, err: %v", ww.id, err)
		}
	}
}

func (ww webhookWorker) process(msg *webhookMessage) (err error) {
	defer func() {
		if err == ErrWebhookIgnored {
			// ignore ErrWebhookIgnored - from here up it's a success
			err = nil
		}
	}()

	wh, err := ParseWebhook(msg.Data, ww.p, msg.InstanceID)
	if err != nil {
		return err
	}
	v := wh.(*webhook)
	if err = v.JiraWebhook.expandIssue(ww.p, msg.InstanceID); err != nil {
		return err
	}
	v.checkIssueWatchers(ww.p, msg.InstanceID)
	v.applyReporterNotification(ww.p, msg.InstanceID, v.Issue.Fields.Reporter)

	if _, _, err = wh.PostNotifications(ww.p, msg.InstanceID); err != nil {
		ww.p.errorf("WebhookWorker id: %d, error posting notifications, err: %v", ww.id, err)
	}

	channelsSubscribed, err := ww.p.getChannelsSubscribed(v, msg.InstanceID)
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
