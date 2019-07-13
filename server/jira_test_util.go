// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"io"
	"os"
	"path/filepath"
)

func getJiraTestData(filename string) io.Reader {
	f, err := os.Open(filepath.Join("testdata", filename))

	if err != nil {
		panic(err)
	}

	return f
}

func withExistingChannelSubscriptions(subscriptions []ChannelSubscription) *Subscriptions {
	ret := NewSubscriptions()
	for _, sub := range subscriptions {
		ret.Channel.add(&sub)
	}
	return ret
}
