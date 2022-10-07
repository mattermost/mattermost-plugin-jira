// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package kvstore

import (
	"crypto/md5" // #nosec G501
	"fmt"
	"strings"

	"github.com/pkg/errors"
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
	data, err := s.store.Load(hashKey(s.prefix, key))
	if err != nil {
		return nil, errors.Wrap(err, key)
	}
	return data, nil
}

func (s *hashedKeyStore) Store(key string, data []byte) error {
	err := s.store.Store(hashKey(s.prefix, key), data)
	if err != nil {
		return errors.Wrap(err, key)
	}
	return nil
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

	h := md5.New() // #nosec G401
	_, _ = h.Write([]byte(hashableKey))
	return fmt.Sprintf("%s%x", prefix, h.Sum(nil))
}
