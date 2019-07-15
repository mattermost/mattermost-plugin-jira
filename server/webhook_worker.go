// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

type webhookWorker struct {
	id        int
	p         *Plugin
	workQueue <-chan []byte
}

func (ww webhookWorker) work() {
	for bb := range ww.workQueue {
		wh, jwh, err := ParseWebhook(bb)
		if err != nil {
			ww.p.errorf("webhookWorker id: %d, error parsing webhook, err: %v", ww.id, err)
		}

		channelIds, err := ww.p.getChannelsSubscribed(jwh)
		if err != nil {
			ww.p.errorf("webhookWorker id: %d, error getting channel's subscribed, err: %v", ww.id, err)
		}

		botUserId := ww.p.getUserID()

		for _, channelId := range channelIds {
			if _, _, err1 := wh.PostToChannel(ww.p, channelId, botUserId); err1 != nil {
				ww.p.errorf("WebhookWorker id: %d, error posting to channel, err: %v", ww.id, err)
			}
		}

		_, _, err = wh.PostNotifications(ww.p)
		if err != nil {
			ww.p.errorf("WebhookWorker id: %d, error posting notifications, err: %v", ww.id, err)
		}
	}
}
