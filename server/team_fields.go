// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"sort"
	"strings"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

const teamFieldKeysKey = "team_field_keys"

func normalizeTeamFieldKeys(keys []string) map[string]struct{} {
	normalized := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(strings.ToLower(key))
		if key == "" {
			continue
		}
		normalized[key] = struct{}{}
	}

	return normalized
}

func copyTeamFieldKeys(keys map[string]struct{}) map[string]struct{} {
	result := make(map[string]struct{}, len(keys))
	for key := range keys {
		result[key] = struct{}{}
	}

	return result
}

func (p *Plugin) cacheTeamFieldKeys(instanceID types.ID, keys []string) {
	normalized := normalizeTeamFieldKeys(keys)
	if len(normalized) == 0 {
		return
	}

	p.teamFieldCacheLock.Lock()
	if p.teamFieldCache == nil {
		p.teamFieldCache = make(map[types.ID]map[string]struct{})
	}

	merged := copyTeamFieldKeys(p.teamFieldCache[instanceID])
	added := false
	for key := range normalized {
		if _, ok := merged[key]; !ok {
			merged[key] = struct{}{}
			added = true
		}
	}
	p.teamFieldCache[instanceID] = merged
	p.teamFieldCacheLock.Unlock()

	if !added {
		return
	}

	if err := p.storeTeamFieldKeys(instanceID, merged); err != nil {
		p.client.Log.Warn("Failed to persist Jira team field keys",
			"instance_id", string(instanceID), "error", err.Error())
	}
}

func (p *Plugin) getTeamFieldKeys(instanceID types.ID) map[string]struct{} {
	p.teamFieldCacheLock.RLock()
	cached, loaded := p.teamFieldCache[instanceID]
	p.teamFieldCacheLock.RUnlock()

	if loaded {
		return copyTeamFieldKeys(cached)
	}

	stored, err := p.loadTeamFieldKeys(instanceID)
	if err != nil {
		p.client.Log.Warn("Failed to load Jira team field keys",
			"instance_id", string(instanceID), "error", err.Error())
		return map[string]struct{}{}
	}

	p.teamFieldCacheLock.Lock()
	if p.teamFieldCache == nil {
		p.teamFieldCache = make(map[types.ID]map[string]struct{})
	}
	if current, ok := p.teamFieldCache[instanceID]; ok {
		stored = current
	} else {
		p.teamFieldCache[instanceID] = stored
	}
	p.teamFieldCacheLock.Unlock()

	return copyTeamFieldKeys(stored)
}

func (p *Plugin) storeTeamFieldKeys(instanceID types.ID, keys map[string]struct{}) error {
	sorted := make([]string, 0, len(keys))
	for key := range keys {
		sorted = append(sorted, key)
	}
	sort.Strings(sorted)

	_, err := p.client.KV.Set(keyWithInstanceID(instanceID, teamFieldKeysKey), sorted)
	return err
}

func (p *Plugin) loadTeamFieldKeys(instanceID types.ID) (map[string]struct{}, error) {
	var keys []string
	if err := p.client.KV.Get(keyWithInstanceID(instanceID, teamFieldKeysKey), &keys); err != nil {
		return nil, err
	}

	return normalizeTeamFieldKeys(keys), nil
}

// discoverTeamFieldKeys resolves the instance's team field keys from Jira's field
// metadata, which covers fields that never appear on a project's create screen.
func (p *Plugin) discoverTeamFieldKeys(instanceID types.ID, client Client) {
	fields, err := client.ListFields()
	if err != nil {
		p.client.Log.Debug("Failed to list Jira fields for team field discovery",
			"instance_id", string(instanceID), "error", err.Error())
		return
	}

	keys := make([]string, 0, 1)
	for _, field := range fields {
		if isTeamFieldSchema(field.Schema.Custom) {
			keys = append(keys, field.ID)
		}
	}

	p.cacheTeamFieldKeys(instanceID, keys)
}
