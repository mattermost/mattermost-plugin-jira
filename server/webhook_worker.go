// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"github.com/pkg/errors"

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
		ww.p.client.Log.Debug("webhookWorker.work")
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

func (ww webhookWorker) process(msg *webhookMessage) (err error) {
	defer func() {
		if err == ErrWebhookIgnored {
			// ignore ErrWebhookIgnored - from here up it's a success
			err = nil
		}
	}()

	ww.p.client.Log.Debug("webhookWorker.process 1")

	wh, err := ParseWebhook(msg.Data)
	if err != nil {
		ww.p.client.Log.Debug("webhookWorker.process 2")
		return err
	}

	if _, _, err = wh.PostNotifications(ww.p, msg.InstanceID); err != nil {
		ww.p.client.Log.Debug("webhookWorker.process 3")
		ww.p.errorf("WebhookWorker id: %d, error posting notifications, err: %v", ww.id, err)
	}

	v := wh.(*webhook)
	if err = v.JiraWebhook.expandIssue(ww.p, msg.InstanceID); err != nil {
		ww.p.client.Log.Debug("webhookWorker.process 4")
		return err
	}

	channelsSubscribed, err := ww.p.getChannelsSubscribed(v, msg.InstanceID)
	if err != nil {
		ww.p.client.Log.Debug("webhookWorker.process 5")
		return err
	}

	ww.p.client.Log.Debug("webhookWorker.process 6", "len_channels_subscribed", len(channelsSubscribed))

	botUserID := ww.p.getUserID()
	for _, channelSubscribed := range channelsSubscribed {
		if _, _, err1 := wh.PostToChannel(ww.p, msg.InstanceID, channelSubscribed.ChannelID, botUserID, channelSubscribed.Name); err1 != nil {
			ww.p.client.Log.Debug("webhookWorker.process 7")
			ww.p.errorf("WebhookWorker id: %d, error posting to channel, err: %v", ww.id, err1)
		}
	}

	ww.p.client.Log.Debug("webhookWorker.process 8")
	return nil
}
