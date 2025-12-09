// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"strings"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

func (p *Plugin) cacheTeamFieldKeys(instanceID types.ID, keys []string) {
	if len(keys) == 0 {
		return
	}

	normalized := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(strings.ToLower(key))
		if key == "" {
			continue
		}
		normalized[key] = struct{}{}
	}

	if len(normalized) == 0 {
		return
	}

	p.teamFieldCacheLock.Lock()
	defer p.teamFieldCacheLock.Unlock()

	if p.teamFieldCache == nil {
		p.teamFieldCache = make(map[types.ID]map[string]struct{})
	}

	current := p.teamFieldCache[instanceID]
	if current == nil {
		current = make(map[string]struct{}, len(normalized))
	}
	for key := range normalized {
		current[key] = struct{}{}
	}

	p.teamFieldCache[instanceID] = current
}

func (p *Plugin) getTeamFieldKeys(instanceID types.ID) map[string]struct{} {
	p.teamFieldCacheLock.RLock()
	defer p.teamFieldCacheLock.RUnlock()

	cached := p.teamFieldCache[instanceID]
	if len(cached) == 0 {
		return map[string]struct{}{
			defaultTeamFieldKey: {},
		}
	}

	// Copy under the read lock to avoid races if the map is updated concurrently.
	result := make(map[string]struct{}, len(cached))
	for key := range cached {
		result[key] = struct{}{}
	}

	return result
}
