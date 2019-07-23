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
		wh, err := ParseWebhook(bb)
		if err != nil {
			// Don't log an error if we're just ignoring the webhook.
			if err != ErrWebhookIgnored {
				ww.p.errorf("webhookWorker id: %d, error parsing webhook, err: %v", ww.id, err)
			}
			continue
		}

		_, _, err = wh.PostNotifications(ww.p)
		if err != nil {
			ww.p.errorf("WebhookWorker id: %d, error posting notifications, err: %v", ww.id, err)
		}

		channelIds, err := ww.p.getChannelsSubscribed(wh.(*webhook))
		if err != nil {
			ww.p.errorf("webhookWorker id: %d, error getting channel's subscribed, err: %v", ww.id, err)
			continue
		}

		botUserId := ww.p.getUserID()

		for _, channelId := range channelIds {
			if _, _, err1 := wh.PostToChannel(ww.p, channelId, botUserId); err1 != nil {
				ww.p.errorf("WebhookWorker id: %d, error posting to channel, err: %v", ww.id, err)
			}
		}
	}
}
