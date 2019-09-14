// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"time"
)

type webhookWorker struct {
	id        int
	p         *Plugin
	workQueue <-chan []byte
}

func (ww webhookWorker) process(rawData []byte) (err error) {
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

		if ww.p.Stats != nil {
			ww.p.Stats.SubscribeWebhook.Processing("",
				time.Since(start), isError, isIgnored)
		}
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
