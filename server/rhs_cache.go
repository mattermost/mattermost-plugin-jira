// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"time"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

const rhsStatusCacheTTL = time.Hour

type rhsStatusCacheEntry struct {
	statuses   []*JiraStatus
	categories []*JiraStatusCategory
	fetchedAt  time.Time
}

type rhsStatusLister interface {
	ListStatuses() ([]*JiraStatus, error)
	ListStatusCategories() ([]*JiraStatusCategory, error)
}

func copyRHSStatusCacheEntry(in *rhsStatusCacheEntry) *rhsStatusCacheEntry {
	if in == nil {
		return nil
	}
	out := &rhsStatusCacheEntry{fetchedAt: in.fetchedAt}
	if in.statuses != nil {
		out.statuses = append([]*JiraStatus(nil), in.statuses...)
	}
	if in.categories != nil {
		out.categories = append([]*JiraStatusCategory(nil), in.categories...)
	}
	return out
}

func (p *Plugin) freshRHSStatusCacheLocked(instanceID types.ID) *rhsStatusCacheEntry {
	entry := p.rhsStatusCache[instanceID]
	if entry == nil {
		return nil
	}
	if time.Since(entry.fetchedAt) >= rhsStatusCacheTTL {
		return nil
	}
	return copyRHSStatusCacheEntry(entry)
}

func (p *Plugin) getInstanceStatuses(instanceID types.ID, client rhsStatusLister) (*rhsStatusCacheEntry, error) {
	p.rhsStatusCacheLock.RLock()
	if entry := p.freshRHSStatusCacheLocked(instanceID); entry != nil {
		p.rhsStatusCacheLock.RUnlock()
		return entry, nil
	}
	p.rhsStatusCacheLock.RUnlock()

	statuses, err := client.ListStatuses()
	if err != nil {
		return nil, err
	}
	categories, err := client.ListStatusCategories()
	if err != nil {
		return nil, err
	}

	entry := &rhsStatusCacheEntry{
		statuses:   statuses,
		categories: categories,
		fetchedAt:  time.Now(),
	}

	p.rhsStatusCacheLock.Lock()
	defer p.rhsStatusCacheLock.Unlock()
	if existing := p.freshRHSStatusCacheLocked(instanceID); existing != nil {
		return existing, nil
	}
	if p.rhsStatusCache == nil {
		p.rhsStatusCache = make(map[types.ID]*rhsStatusCacheEntry)
	}
	p.rhsStatusCache[instanceID] = entry
	return copyRHSStatusCacheEntry(entry), nil
}

func (p *Plugin) invalidateRHSStatusCache() {
	p.rhsStatusCacheLock.Lock()
	defer p.rhsStatusCacheLock.Unlock()
	p.rhsStatusCache = make(map[types.ID]*rhsStatusCacheEntry)
}
