// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

// TestMapUsers_PageBoundarySkipRegression reproduces the original MapUsers
// bug: deleting a KV key that sorts before "user_" keys while iterating
// offset-based pages shifts every subsequent page and silently skips users.
// It backs KVList/KVGet/KVSetWithOptions with a real, mutating in-memory
// key list so a delete from inside the callback has the same page-shifting
// effect a real KV store would have.
func TestMapUsers_PageBoundarySkipRegression(t *testing.T) {
	origPerPage := listPerPage
	listPerPage = 10 // small pages make the boundary easy to cross deterministically
	defer func() { listPerPage = origPerPage }()

	const numUsers = 37 // several times the page size, and not a multiple of it
	const numBareKeys = 6

	userIDs := make([]types.ID, 0, numUsers)
	values := map[string][]byte{}
	var keys []string

	for i := 0; i < numUsers; i++ {
		id := types.ID(fmt.Sprintf("user-%02d", i))
		userIDs = append(userIDs, id)
		key := hashkey(prefixUser, id.String())
		keys = append(keys, key)

		data, err := json.Marshal(NewUser(id))
		require.NoError(t, err)
		values[key] = data
	}
	// Bare 32-char-hex keys, like the connection and reverse-index keys
	// disconnectUser deletes. These sort before every "user_" key.
	for i := 0; i < numBareKeys; i++ {
		keys = append(keys, hashkey("", fmt.Sprintf("bare-key-%d", i)))
	}
	sort.Strings(keys)

	var mu sync.Mutex
	api := &plugintest.API{}

	api.On("KVList", mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(
		func(page, count int) ([]string, *model.AppError) {
			mu.Lock()
			defer mu.Unlock()
			start := page * count
			if start >= len(keys) {
				return []string{}, nil
			}
			end := start + count
			if end > len(keys) {
				end = len(keys)
			}
			out := make([]string, end-start)
			copy(out, keys[start:end])
			return out, nil
		},
	)
	api.On("KVGet", mock.AnythingOfType("string")).Return(
		func(key string) ([]byte, *model.AppError) {
			mu.Lock()
			defer mu.Unlock()
			return values[key], nil
		},
	)
	api.On("KVSetWithOptions", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(
		func(key string, _ []byte, _ model.PluginKVSetOptions) (bool, *model.AppError) {
			mu.Lock()
			defer mu.Unlock()
			for i, k := range keys {
				if k == key {
					keys = append(keys[:i], keys[i+1:]...)
					break
				}
			}
			return true, nil
		},
	)

	p := &Plugin{}
	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)
	s := NewStore(p)

	var visited []types.ID
	failedReads, err := s.MapUsers(func(user *User) error {
		visited = append(visited, user.MattermostUserID)

		// Mirror disconnectUser deleting a bare key while MapUsers is
		// iterating, which is what shifted the offset window in the
		// original implementation.
		mu.Lock()
		var toDelete string
		if len(keys) > 0 && !strings.HasPrefix(keys[0], prefixUser) {
			toDelete = keys[0]
		}
		mu.Unlock()
		if toDelete != "" {
			require.NoError(t, p.client.KV.Delete(toDelete))
		}
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 0, failedReads)
	assert.ElementsMatch(t, userIDs, visited, "every user must be visited exactly once, even though the callback deletes keys that sort before user_ keys")
}
