// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"io"
	"os"
	"path/filepath"
)

const someSecret = "somesecret"

func getJiraTestData(filename string) ([]byte, error) {
	f, err := os.Open(filepath.Join("testdata", filename))

	if err != nil {
		panic(err)
	}

	defer f.Close()
	return io.ReadAll(f)
}

func withExistingChannelSubscriptions(subscriptions []ChannelSubscription) *Subscriptions {
	ret := NewSubscriptions()
	for i := range subscriptions {
		subscriptions[i].InstanceID = testInstance1.GetID()
		ret.Channel.add(&subscriptions[i])
	}
	return ret
}
