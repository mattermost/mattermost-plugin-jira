// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package kvstore

import (
	"time"

	"github.com/mattermost/mattermost/server/public/pluginapi"

	"github.com/pkg/errors"
)

type pluginStore struct {
	client     *pluginapi.Client
	ttlSeconds int64
}

var _ KVStore = (*pluginStore)(nil)

func NewPluginStore(client *pluginapi.Client) KVStore {
	return NewPluginStoreWithExpiry(client, 0)
}

func NewPluginStoreWithExpiry(client *pluginapi.Client, ttl time.Duration) KVStore {
	return &pluginStore{
		client:     client,
		ttlSeconds: (int64)(ttl / time.Second),
	}
}

func (s *pluginStore) Load(key string) ([]byte, error) {
	var data []byte
	err := s.client.KV.Get(key, &data)
	if err != nil {
		return nil, errors.WithMessage(err, "failed plugin KVGet")
	}
	if len(data) == 0 {
		return nil, errors.Wrap(ErrNotFound, key)
	}
	return data, nil
}

func (s *pluginStore) Store(key string, data []byte) error {
	var err error
	if s.ttlSeconds > 0 {
		_, err = s.client.KV.Set(key, data, pluginapi.SetExpiry(time.Duration(s.ttlSeconds)))
	} else {
		_, err = s.client.KV.Set(key, data)
	}
	if err != nil {
		return errors.WithMessagef(err, "failed plugin KVSet (ttl: %vs) %q", s.ttlSeconds, key)
	}
	return nil
}

func (s *pluginStore) StoreTTL(key string, data []byte, ttlSeconds int64) error {
	_, err := s.client.KV.Set(key, data, pluginapi.SetExpiry(time.Duration(ttlSeconds)))
	if err != nil {
		return errors.WithMessagef(err, "failed plugin KVSet (ttl: %vs) %q", s.ttlSeconds, key)
	}
	return nil
}

func (s *pluginStore) Delete(key string) error {
	err := s.client.KV.Delete(key)
	if err != nil {
		return errors.WithMessagef(err, "failed plugin KVdelete %q", key)
	}
	return nil
}

const listPerPage = 100

func (s *pluginStore) Keys() ([]string, error) {
	keys := []string{}
	for i := 0; ; i++ {
		moreKeys, err := s.client.KV.ListKeys(i, listPerPage)
		if err != nil {
			return nil, err
		}
		if len(moreKeys) < listPerPage {
			break
		}
		keys = append(keys, moreKeys...)
	}
	return keys, nil
}

func (s *pluginStore) Flush() []error {
	return nil
}
