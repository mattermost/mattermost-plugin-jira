// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"time"

	"github.com/mattermost/mattermost-plugin-jira/server/stats"
)

type webhookWorker struct {
	id        int
	p         *Plugin
	workQueue <-chan []byte
}

func (ww webhookWorker) process(rawData []byte) (err error) {
	startTime := time.Now()
	defer func() {
		isError, isIgnored := false, false
		if err != nil {
			if err == ErrWebhookIgnored {
				isIgnored = true
			} else {
				// TODO save the payload here
				isError = true
			}
		}
		stats.RecordWebhookProcessed(stats.WebhookSubscribe, isError, isIgnored, time.Since(startTime))
	}()

	wh, err := ParseWebhook(rawData)
	if err != nil {
		return err
	}

	_, _, err = wh.PostNotifications(ww.p)
	if err != nil {
		ww.p.errorf("WebhookWorker id: %d, error posting notifications, err: %v", ww.id, err)
	}

	channelIds, err := ww.p.getChannelsSubscribed(wh.(*webhook))
	if err != nil {
		return err
	}
	botUserId := ww.p.getUserID()
	for _, channelId := range channelIds {
		if _, _, err1 := wh.PostToChannel(ww.p, channelId, botUserId); err1 != nil {
			ww.p.errorf("WebhookWorker id: %d, error posting to channel, err: %v", ww.id, err)
		}
	}

	return nil
}

func (ww webhookWorker) work() {
	for rawData := range ww.workQueue {
		err := ww.process(rawData)
		if err != nil {
			ww.p.errorf("WebhookWorker id: %d, error processing, err: %v", ww.id, err)
		}

	}
}
