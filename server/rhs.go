// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"strings"
	"time"

	jira "github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

const rhsIssuesPageSize = 20

var rhsSearchFields = []string{
	"summary", "status", "priority", "issuetype", "assignee", "reporter",
	"created", "updated", "duedate", "project", "labels",
}

// rhsCloudClient is the Cloud-only surface the HTTP layer type-asserts to.
// Do not add these methods to the shared Client / SearchService tree.
type rhsCloudClient interface {
	rhsStatusLister
	SearchJQL(params CloudSearchParams) (*CloudSearchResult, error)
}

func (p *Plugin) resolveRHSUserClient(instanceID, mattermostUserID types.ID) (Instance, rhsCloudClient, error) {
	if instanceID == "" {
		return nil, nil, errors.Wrap(ErrInvalidRHSTab, "instance_id is required")
	}
	instance, err := p.instanceStore.LoadInstance(instanceID)
	if err != nil {
		if errors.Is(err, kvstore.ErrNotFound) {
			return nil, nil, errors.Wrap(ErrInvalidRHSTab, "unknown instance_id")
		}
		return nil, nil, err
	}

	client, instance, _, err := p.getClient(instance.GetID(), mattermostUserID)
	if err != nil {
		return nil, nil, err
	}
	rhs, ok := client.(rhsCloudClient)
	if !ok {
		return nil, nil, ErrRHSNotCloud
	}
	return instance, rhs, nil
}

func (p *Plugin) resolveRHSAdminClient(instance Instance, adminUserID types.ID) (rhsCloudClient, error) {
	switch inst := instance.(type) {
	case *cloudInstance:
		raw, err := inst.getClientForBot()
		if err != nil {
			return nil, err
		}
		rhs, ok := newCloudClient(raw).(rhsCloudClient)
		if !ok {
			return nil, ErrRHSNotCloud
		}
		return rhs, nil

	case *cloudOAuthInstance:
		connection, err := p.userStore.LoadConnection(inst.GetID(), adminUserID)
		if err != nil {
			return nil, err
		}
		client, err := inst.GetClient(connection)
		if err != nil {
			return nil, err
		}
		rhs, ok := client.(rhsCloudClient)
		if !ok {
			return nil, ErrRHSNotCloud
		}
		return rhs, nil

	default:
		return nil, ErrRHSNotCloud
	}
}

func rhsBrowseURL(instance Instance, issueKey string) string {
	return strings.TrimRight(instance.GetJiraBaseURL(), "/") + "/browse/" + issueKey
}

func rhsTimeRFC3339(t jira.Time) string {
	tt := time.Time(t)
	if tt.IsZero() {
		return ""
	}
	return tt.UTC().Format(time.RFC3339)
}

func rhsDate(d jira.Date) string {
	tt := time.Time(d)
	if tt.IsZero() {
		return ""
	}
	return tt.Format("2006-01-02")
}

func normalizeRHSIssue(instance Instance, issue jira.Issue) RHSIssue {
	out := RHSIssue{
		Key:       issue.Key,
		BrowseURL: rhsBrowseURL(instance, issue.Key),
		Labels:    []string{},
	}
	if issue.Fields == nil {
		return out
	}
	f := issue.Fields
	out.Summary = f.Summary
	out.IssueType = f.Type.Name
	out.Project = f.Project.Key
	out.Created = rhsTimeRFC3339(f.Created)
	out.Updated = rhsTimeRFC3339(f.Updated)
	out.DueDate = rhsDate(f.Duedate)
	if f.Labels != nil {
		out.Labels = f.Labels
	}
	if f.Status != nil {
		out.Status = RHSIssueStatus{
			Name:        f.Status.Name,
			CategoryKey: f.Status.StatusCategory.Key,
		}
	}
	if f.Priority != nil {
		out.Priority = f.Priority.Name
	}
	if f.Assignee != nil {
		out.Assignee = f.Assignee.DisplayName
	}
	if f.Reporter != nil {
		out.Reporter = f.Reporter.DisplayName
	}
	return out
}

func normalizeRHSIssues(instance Instance, issues []jira.Issue) []RHSIssue {
	out := make([]RHSIssue, 0, len(issues))
	for _, issue := range issues {
		out = append(out, normalizeRHSIssue(instance, issue))
	}
	return out
}

func pickRHSTab(tabs []RHSTabEntry, kind, key, id string) (RHSTabEntry, error) {
	if kind == "" {
		kind = string(RHSTabKindAssigned)
	}
	for _, tab := range tabs {
		if string(tab.Kind) != kind {
			continue
		}
		switch tab.Kind {
		case RHSTabKindAssigned:
			return tab, nil
		case RHSTabKindCategory:
			if tab.Key == key {
				return tab, nil
			}
		case RHSTabKindStatus:
			if tab.ID == id {
				return tab, nil
			}
		}
	}
	return RHSTabEntry{}, errors.Wrapf(ErrInvalidRHSTab, "tab %s/%s/%s is not in the resolved list", kind, key, id)
}

type rhsIssuesResult struct {
	Issues        []RHSIssue    `json:"issues"`
	Tabs          []RHSTabEntry `json:"tabs"`
	NextPageToken string        `json:"nextPageToken"`
	IsLast        bool          `json:"isLast"`
}

func (p *Plugin) getRHSIssues(instanceID, mattermostUserID types.ID, tabKind, tabKey, tabID, sort, nextPageToken string) (*rhsIssuesResult, error) {
	instance, client, err := p.resolveRHSUserClient(instanceID, mattermostUserID)
	if err != nil {
		return nil, err
	}

	entry, err := p.getInstanceStatuses(instance.GetID(), client)
	if err != nil {
		return nil, err
	}
	// Read-only: do not mutate entry.statuses / entry.categories.

	conf := p.getConfig()
	configured := conf.RHSStatusTabs[string(instance.GetID())]
	tabs := resolveTabs(configured, entry.statuses, entry.categories)

	selected, err := pickRHSTab(tabs, tabKind, tabKey, tabID)
	if err != nil {
		return nil, err
	}
	if sort == "" {
		sort = rhsDefaultSort
	}

	// RE3: cached keys only. Do not substitute a hardcoded four-key map.
	jql, err := buildTabJQL(selected, sort, validCategoryKeysFrom(entry.categories))
	if err != nil {
		return nil, err
	}

	search, err := client.SearchJQL(CloudSearchParams{
		JQL:           jql,
		Fields:        rhsSearchFields,
		MaxResults:    rhsIssuesPageSize,
		NextPageToken: nextPageToken,
	})
	if err != nil {
		return nil, err
	}

	return &rhsIssuesResult{
		Issues:        normalizeRHSIssues(instance, search.Issues),
		Tabs:          tabs,
		NextPageToken: search.NextPageToken,
		IsLast:        search.IsLast,
	}, nil
}

type rhsStatusesResult struct {
	Statuses   []*JiraStatus         `json:"statuses"`
	Categories []*JiraStatusCategory `json:"categories"`
}

func (p *Plugin) getRHSStatuses(instanceID, adminUserID types.ID) (*rhsStatusesResult, error) {
	if instanceID == "" {
		return nil, errors.Wrap(ErrInvalidRHSTab, "instance_id is required")
	}
	instance, err := p.instanceStore.LoadInstance(instanceID)
	if err != nil {
		if errors.Is(err, kvstore.ErrNotFound) {
			return nil, errors.Wrap(ErrInvalidRHSTab, "unknown instance_id")
		}
		return nil, err
	}

	client, err := p.resolveRHSAdminClient(instance, adminUserID)
	if err != nil {
		return nil, err
	}

	entry, err := p.getInstanceStatuses(instance.GetID(), client)
	if err != nil {
		return nil, err
	}
	return &rhsStatusesResult{
		Statuses:   entry.statuses,
		Categories: entry.categories,
	}, nil
}
