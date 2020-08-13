// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"time"

	"github.com/mattermost/mattermost-plugin-jira/server/utils"
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
	conf := ww.p.getConfig()
	start := time.Now()
	defer func() {
		isError, isIgnored := false, false
		switch err {
		case nil:
			break
		case ErrWebhookIgnored:
			// ignore ErrWebhookIgnored - from here up it's a success
			isIgnored = true
			err = nil
		default:
			// TODO save the payload here
			isError = true
		}
		if conf.stats != nil {
			path := instancePath("jira/subscribe/processing", msg.InstanceID)
			conf.stats.EnsureEndpoint(path).Record(utils.ByteSize(len(msg.Data)), 0, time.Since(start), isError, isIgnored)
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

	channelIds, err := ww.p.getChannelsSubscribed(v, msg.InstanceID)
	if err != nil {
		return err
	}

	botUserId := ww.p.getUserID()
	for _, channelId := range channelIds.Elems() {
		if _, _, err1 := wh.PostToChannel(ww.p, msg.InstanceID, channelId, botUserId); err1 != nil {
			ww.p.errorf("WebhookWorker id: %d, error posting to channel, err: %v", ww.id, err1)
		}
	}

	if err := ww.p.NotifyWorkflow(wh.(*webhook)); err != nil {
		ww.p.errorf("WebhookWorker id: %d, error notifying workflow, err: %v", ww.id, err)
	}

	return nil
}
