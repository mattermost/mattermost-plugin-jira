// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package kvstore

import (
	"bytes"
	"errors"
)

type cacheKVStore struct {
	upstream KVStore

	// key value  of nil indicated deletion
	Data map[string][]byte

	DirtyKeys map[string]bool
}

var _ KVStore = (*cacheKVStore)(nil)

func NewCacheKVStore(s KVStore) KVStore {
	return &cacheKVStore{
		upstream:  s,
		Data:      map[string][]byte{},
		DirtyKeys: map[string]bool{},
	}
}

func (s *cacheKVStore) Flush() []error {
	if s.upstream == nil {
		return nil
	}

	var errs []error
	for key := range s.DirtyKeys {
		var err error
		data := s.Data[key]
		if data == nil {
			err = s.upstream.Delete(key)
		} else {
			err = s.upstream.Store(key, data)
		}
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func (s *cacheKVStore) Load(key string) ([]byte, error) {
	data, ok := s.Data[key]
	if ok {
		if data == nil {
			return nil, ErrNotFound
		}
		return data, nil
	}

	if s.upstream == nil {
		return nil, ErrNotFound
	}
	data, err := s.upstream.Load(key)
	if err != nil {
		return nil, err
	}

	s.Data[key] = data
	return data, nil
}

func (s *cacheKVStore) Store(key string, data []byte) error {
	prev, ok := s.Data[key]
	if ok && bytes.Equal(data, prev) {
		return nil
	}

	s.Data[key] = data
	s.DirtyKeys[key] = true
	return nil
}

func (s *cacheKVStore) StoreTTL(key string, data []byte, ttlSeconds int64) error {
	if ttlSeconds > 0 {
		return errors.New("TODO: expiry not implemented yet")
	}
	return s.Store(key, data)
}

func (s *cacheKVStore) Delete(key string) error {
	return s.Store(key, nil)
}

func (s *cacheKVStore) Keys() ([]string, error) {
	var err error
	// Get all keys from the upstream
	keys := []string{}
	if s.upstream != nil {
		keys, err = s.upstream.Keys()
		if err != nil {
			return nil, err
		}
	}

	// Merge with any dirty keys we have
	kmap := map[string]bool{}
	for _, key := range keys {
		kmap[key] = true
	}
	for key := range s.DirtyKeys {
		kmap[key] = true
	}

	// return the merged set
	keys = []string{}
	for key := range kmap {
		keys = append(keys, key)
	}
	return keys, nil
}
