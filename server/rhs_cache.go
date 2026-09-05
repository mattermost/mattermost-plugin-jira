// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"time"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

const rhsStatusCacheTTL = time.Hour

// rhsStatusCacheBotUser keys statuses fetched with the Cloud JWT bot client.
// GET /3/status is visibility-scoped; bot and user results must not share a slot.
const rhsStatusCacheBotUser types.ID = "bot"

type rhsStatusCacheKey struct {
	instanceID types.ID
	userID     types.ID
}

type rhsStatusCacheEntry struct {
	statuses   []*JiraStatus
	categories []*JiraStatusCategory
	fetchedAt  time.Time
}

type rhsStatusFlight struct {
	done  chan struct{}
	entry *rhsStatusCacheEntry
	err   error
}

type rhsStatusLister interface {
	ListStatuses() ([]*JiraStatus, error)
	ListStatusCategories() ([]*JiraStatusCategory, error)
}

// statusProjectLookup is implemented by the Cloud client. Test listers omit it
// so existing mocks do not have to stub project search. Do not type-assert
// ListProjects — testClient embeds a nil ProjectService and would panic.
type statusProjectLookup interface {
	lookupStatusProjects(ids []string) (map[string]JiraStatusProject, error)
}

func rhsAdminStatusCacheUserID(instance Instance, adminUserID types.ID) types.ID {
	if _, ok := instance.(*cloudInstance); ok {
		return rhsStatusCacheBotUser
	}
	return adminUserID
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

func (p *Plugin) freshRHSStatusCacheLocked(key rhsStatusCacheKey) *rhsStatusCacheEntry {
	entry := p.rhsStatusCache[key]
	if entry == nil {
		return nil
	}
	if time.Since(entry.fetchedAt) >= rhsStatusCacheTTL {
		return nil
	}
	return copyRHSStatusCacheEntry(entry)
}

func (p *Plugin) fetchRHSStatuses(client rhsStatusLister) (*rhsStatusCacheEntry, error) {
	statuses, err := client.ListStatuses()
	if err != nil {
		return nil, err
	}
	categories, err := client.ListStatusCategories()
	if err != nil {
		return nil, err
	}
	return &rhsStatusCacheEntry{
		statuses:   statuses,
		categories: categories,
		fetchedAt:  time.Now(),
	}, nil
}

// getInstanceStatuses returns statuses visible to userID's Jira credentials.
// Team-managed project statuses are omitted for users without project access,
// so the cache is keyed by instance and credential owner.
func (p *Plugin) getInstanceStatuses(instanceID, userID types.ID, client rhsStatusLister) (*rhsStatusCacheEntry, error) {
	key := rhsStatusCacheKey{instanceID: instanceID, userID: userID}

	p.rhsStatusCacheLock.RLock()
	if entry := p.freshRHSStatusCacheLocked(key); entry != nil {
		p.rhsStatusCacheLock.RUnlock()
		return entry, nil
	}
	p.rhsStatusCacheLock.RUnlock()

	p.rhsStatusCacheLock.Lock()
	if entry := p.freshRHSStatusCacheLocked(key); entry != nil {
		p.rhsStatusCacheLock.Unlock()
		return entry, nil
	}
	if flight := p.rhsStatusFlights[key]; flight != nil {
		p.rhsStatusCacheLock.Unlock()
		<-flight.done
		if flight.err != nil {
			return nil, flight.err
		}
		return copyRHSStatusCacheEntry(flight.entry), nil
	}
	flight := &rhsStatusFlight{done: make(chan struct{})}
	if p.rhsStatusFlights == nil {
		p.rhsStatusFlights = make(map[rhsStatusCacheKey]*rhsStatusFlight)
	}
	p.rhsStatusFlights[key] = flight
	p.rhsStatusCacheLock.Unlock()

	entry, err := p.fetchRHSStatuses(client)

	p.rhsStatusCacheLock.Lock()
	if err == nil {
		if p.rhsStatusCache == nil {
			p.rhsStatusCache = make(map[rhsStatusCacheKey]*rhsStatusCacheEntry)
		}
		p.rhsStatusCache[key] = entry
	}
	flight.entry = entry
	flight.err = err
	delete(p.rhsStatusFlights, key)
	close(flight.done)
	p.rhsStatusCacheLock.Unlock()

	if err != nil {
		return nil, err
	}
	return copyRHSStatusCacheEntry(entry), nil
}

func (p *Plugin) enrichStatusesWithProjects(client rhsStatusLister, statuses []*JiraStatus) {
	ids := uniqueStatusProjectIDs(statuses)
	if len(ids) == 0 {
		return
	}
	lookup, ok := client.(statusProjectLookup)
	if !ok {
		return
	}
	byID, err := lookup.lookupStatusProjects(ids)
	if err != nil {
		p.client.Log.Warn("Failed to load Jira projects for RHS status tabs", "error", err.Error())
		return
	}
	applyStatusProjects(statuses, byID)
}

func (p *Plugin) invalidateRHSStatusCache() {
	p.rhsStatusCacheLock.Lock()
	defer p.rhsStatusCacheLock.Unlock()
	p.rhsStatusCache = make(map[rhsStatusCacheKey]*rhsStatusCacheEntry)
}
