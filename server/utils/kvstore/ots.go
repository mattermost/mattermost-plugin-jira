// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package kvstore

import (
	"time"

	"github.com/mattermost/mattermost/server/public/pluginapi"
)

// OneTimeStore is a KV store that deletes each record after the first load,
type OneTimeStore KVStore

type ots struct {
	KVStore
}

func NewOneTimePluginStore(client *pluginapi.Client, ttl time.Duration) OneTimeStore {
	return &ots{
		KVStore: NewPluginStoreWithExpiry(client, ttl),
	}
}

func NewOneTimeStore(kv KVStore) OneTimeStore {
	return &ots{
		KVStore: kv,
	}
}

func (s ots) Load(key string) (data []byte, returnErr error) {
	data, err := s.KVStore.Load(key)
	if len(data) == 0 {
		_ = s.Delete(key)
	}
	return data, err
}
