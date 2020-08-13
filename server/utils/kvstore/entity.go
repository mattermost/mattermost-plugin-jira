// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package kvstore

import (
	"regexp"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
	"github.com/mattermost/mattermost-server/v5/model"
)

type EntityStore interface {
	Delete(types.ID) error
	Load(types.ID, interface{}) error
	NewID(name string) (types.ID, error)
	Store(types.ID, interface{}) error
}

type entityStore struct {
	kv KVStore
}

func (s *store) Entity(prefix string) EntityStore {
	return &entityStore{
		kv: NewHashedKeyStore(s.KVStore, prefix),
	}
}

func (s *entityStore) Load(id types.ID, ref interface{}) error {
	return LoadJSON(s.kv, string(id), ref)
}

func (s *entityStore) Store(id types.ID, ref interface{}) error {
	return StoreJSON(s.kv, string(id), ref)
}

func (s *entityStore) Delete(id types.ID) error {
	return s.kv.Delete(string(id))
}

var ErrTryAgain = errors.New("try again")

func (e *entityStore) NewID(name string) (types.ID, error) {
	for i := 0; i < 5; i++ {
		id := name
		if i > 0 {
			id = name + "-" + model.NewId()[:7]
		}

		dummy := struct{}{}
		err := e.Load(types.ID(id), &dummy)
		if errors.Cause(err) == ErrNotFound {
			return types.ID(id), nil
		}
	}

	return "", ErrTryAgain
}

var reModelID = regexp.MustCompile(`-[a-z0-9]{7}$`)

func NameFromID(id types.ID) string {
	s := string(id)
	if reModelID.MatchString(s) {
		return s[0 : len(id)-8]
	}
	return s
}
