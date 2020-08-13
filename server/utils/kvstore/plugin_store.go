// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package kvstore

import (
	"time"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"

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
	data, appErr := s.api.KVGet(key)
	if appErr != nil {
		return nil, errors.WithMessage(appErr, "failed plugin KVGet")
	}
	if data == nil {
		return nil, errors.Wrap(ErrNotFound, key)
	}
	return data, nil
}

func (s *pluginStore) Store(key string, data []byte) error {
	var appErr *model.AppError
	if s.ttlSeconds > 0 {
		appErr = s.api.KVSetWithExpiry(key, data, s.ttlSeconds)
	} else {
		appErr = s.api.KVSet(key, data)
	}
	if appErr != nil {
		return errors.WithMessagef(appErr, "failed plugin KVSet (ttl: %vs) %q", s.ttlSeconds, key)
	}
	return nil
}

func (s *pluginStore) StoreTTL(key string, data []byte, ttlSeconds int64) error {
	appErr := s.api.KVSetWithExpiry(key, data, ttlSeconds)
	if appErr != nil {
		return errors.WithMessagef(appErr, "failed plugin KVSet (ttl: %vs) %q", s.ttlSeconds, key)
	}
	return nil
}

func (s *pluginStore) Delete(key string) error {
	appErr := s.api.KVDelete(key)
	if appErr != nil {
		return errors.WithMessagef(appErr, "failed plugin KVdelete %q", key)
	}
	return nil
}

const listPerPage = 100

func (s *pluginStore) Keys() ([]string, error) {
	keys := []string{}
	for i := 0; ; i++ {
		moreKeys, appErr := s.api.KVList(i, listPerPage)
		if appErr != nil {
			return nil, appErr
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
