// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"io/ioutil"
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
	return ioutil.ReadAll(f)
}

func withExistingChannelSubscriptions(subscriptions []ChannelSubscription) *Subscriptions {
	ret := NewSubscriptions()
	for i, sub := range subscriptions {
		sub.InstanceID = testInstance1.GetID()
		ret.Channel.add(&subscriptions[i])
	}
	return ret
}
