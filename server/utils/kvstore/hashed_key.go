// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package kvstore

import (
	"crypto/md5"
	"fmt"
	"strings"
)

type hashedKeyStore struct {
	store  KVStore
	prefix string
}

var _ KVStore = (*hashedKeyStore)(nil)

func NewHashedKeyStore(s KVStore, prefix string) KVStore {
	return &hashedKeyStore{
		store:  s,
		prefix: prefix,
	}
}

func (s *hashedKeyStore) Load(key string) ([]byte, error) {
	return s.store.Load(hashKey(s.prefix, key))
}

func (s *hashedKeyStore) Store(key string, data []byte) error {
	return s.store.Store(hashKey(s.prefix, key), data)
}

func (s *hashedKeyStore) StoreTTL(key string, data []byte, ttlSeconds int64) error {
	return s.store.StoreTTL(hashKey(s.prefix, key), data, ttlSeconds)
}

func (s *hashedKeyStore) Delete(key string) error {
	return s.store.Delete(hashKey(s.prefix, key))
}

func (s *hashedKeyStore) Keys() ([]string, error) {
	all, err := s.store.Keys()
	if err != nil {
		return nil, err
	}

	matched := []string{}
	for _, key := range all {
		if strings.HasPrefix(key, s.prefix) {
			matched = append(matched, key)
		}
	}
	return matched, nil
}

func (s *hashedKeyStore) Flush() []error {
	return nil
}

func hashKey(prefix, hashableKey string) string {
	if hashableKey == "" {
		return prefix
	}

	h := md5.New()
	_, _ = h.Write([]byte(hashableKey))
	return fmt.Sprintf("%s%x", prefix, h.Sum(nil))
}
