// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"time"

	"github.com/mattermost/mattermost-plugin-jira/server/utils"
)

type webhookWorker struct {
	id        int
	p         *Plugin
	workQueue <-chan []byte
}

func (ww webhookWorker) work() {
	for rawData := range ww.workQueue {
		err := ww.process(rawData)
		if err != nil {
			ww.p.errorf("WebhookWorker id: %d, error processing, err: %v", ww.id, err)
		}

	}
}

func (ww webhookWorker) process(rawData []byte) (err error) {
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
			conf.stats.EnsureEndpoint("jira/subscribe/processing").Record(utils.ByteSize(len(rawData)), 0, time.Since(start), isError, isIgnored)
		}
	}()

	wh, err := ParseWebhook(rawData)
	if err != nil {
		return err
	}

	if _, _, err = wh.PostNotifications(ww.p); err != nil {
		ww.p.errorf("WebhookWorker id: %d, error posting notifications, err: %v", ww.id, err)
	}

	v := wh.(*webhook)
	if err = v.JiraWebhook.expandIssue(ww.p, v.instanceID); err != nil {
		return err
	}

	channelIds, err := ww.p.getChannelsSubscribed(v)
	if err != nil {
		return err
	}
	botUserId := ww.p.getUserID()
	for _, channelId := range channelIds.Elems() {
		if _, _, err1 := wh.PostToChannel(ww.p, v.instanceID, channelId, botUserId); err1 != nil {
			ww.p.errorf("WebhookWorker id: %d, error posting to channel, err: %v", ww.id, err)
		}
	}

	if err := ww.p.NotifyWorkflow(wh.(*webhook)); err != nil {
		ww.p.errorf("WebhookWorker id: %d, error notifying workflow, err: %v", ww.id, err)
	}

	return nil
}
