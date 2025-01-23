// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package kvstore

import "github.com/mattermost/mattermost-plugin-jira/server/utils/types"

type Store interface {
	KVStore

	Entity(string) EntityStore
	ValueIndex(string, types.ValueArray) ValueIndexStore
	IDIndex(string) IDIndexStore
}

type store struct {
	KVStore
}

func NewStore(kv KVStore) Store {
	return &store{
		KVStore: kv,
	}
}
