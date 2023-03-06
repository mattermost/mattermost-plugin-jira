// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package kvstore

import (
	"time"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"

	"github.com/pkg/errors"
)

type pluginStore struct {
	api        plugin.API
	ttlSeconds int64
}

var _ KVStore = (*pluginStore)(nil)

func NewPluginStore(api plugin.API) KVStore {
	return NewPluginStoreWithExpiry(api, 0)
}

func NewPluginStoreWithExpiry(api plugin.API, ttl time.Duration) KVStore {
	return &pluginStore{
		api:        api,
		ttlSeconds: (int64)(ttl / time.Second),
	}
}

func (s *pluginStore) Load(key string) ([]byte, error) {
	data, err := s.client.KV.Get(key)
	if err != nil {
		return nil, errors.WithMessage(err, "failed plugin KVGet")
	}
	if data == nil {
		return nil, errors.Wrap(ErrNotFound, key)
	}
	return data, nil
}

func (s *pluginStore) Store(key string, data []byte) error {
	if s.ttlSeconds > 0 {
		err := s.client.KV.SetWithExpiry(key, data, s.ttlSeconds)
	} else {
		err := s.client.KV.Set(key, data)
	}
	if err != nil {
		return errors.WithMessagef(err, "failed plugin KVSet (ttl: %vs) %q", s.ttlSeconds, key)
	}
	return nil
}

func (s *pluginStore) StoreTTL(key string, data []byte, ttlSeconds int64) error {
	err := s.client.KV.SetWithExpiry(key, data, ttlSeconds)
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
		moreKeys, err := s.client.KV.List(i, listPerPage)
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
