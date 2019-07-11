// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func getJiraTestData(filename string) *JiraWebhook {
	f, err := os.Open(filepath.Join("testdata", filename))

	if err != nil {
		panic(err)
	}

	jwh := &JiraWebhook{}
	err = json.NewDecoder(f).Decode(&jwh)
	if err != nil {
		panic(err)
	}

	return jwh
}

func withExistingChannelSubscriptions(subscriptions []ChannelSubscription) *Subscriptions {
	ret := NewSubscriptions()
	for _, sub := range subscriptions {
		ret.Channel.add(&sub)
	}
	return ret
}
