// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"encoding/json"
	"strconv"

	jira "github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"
)

type jiraCloudClient struct {
	JiraClient
}

func newCloudClient(jiraClient *jira.Client) Client {
	return &jiraCloudClient{
		JiraClient: JiraClient{
			Jira: jiraClient,
		},
	}
}

// GetCreateMeta returns the metadata needed to implement the UI and validation of
// creating new Jira issues.
func (client jiraCloudClient) GetCreateMeta(options *jira.GetQueryOptions) (*jira.CreateMetaInfo, error) {
	cimd, resp, err := client.Jira.Issue.GetCreateMetaWithOptions(options)
	if err != nil {
		if resp == nil {
			return nil, err
		}

		// returns a different JSON from all other APIs
		result := map[string]string{}
		jsonerr := json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()
		if jsonerr == nil {
			err = errors.New(result["error"])
		}
		return nil, RESTError{err, resp.StatusCode}
	}

	return cimd, nil
}

// SearchUsersAssignableToIssue finds all users that can be assigned to an issue.
func (client jiraCloudClient) SearchUsersAssignableToIssue(issueKey, query string, maxResults int) ([]jira.User, error) {
	return SearchUsersAssignableToIssue(client, issueKey, "query", query, maxResults)
}

// SearchUsersAssignableInProject finds all users that can be assigned to some issue in a given project.
func (client jiraCloudClient) SearchUsersAssignableInProject(projectKey, query string, maxResults int) ([]jira.User, error) {
	return SearchUsersAssignableInProject(client, projectKey, "query", query, maxResults)
}

// GetUserGroups returns the list of groups that a user belongs to.
func (client jiraCloudClient) GetUserGroups(connection *Connection) ([]*jira.UserGroup, error) {
	groups := []*jira.UserGroup{}
	params := map[string]string{
		"accountId": connection.AccountID,
	}
	err := client.RESTGet("3/user/groups", params, &groups)
	if err != nil {
		return nil, err
	}
	return groups, nil
}

func (client jiraCloudClient) ListProjects(query string, limit int) (jira.ProjectList, error) {
	type searchResult struct {
		Values     jira.ProjectList `json:"values"`
		StartAt    int              `json:"startAt"`
		MaxResults int              `json:"maxResults"`
		Total      int              `json:"total"`
		IsLast     bool             `json:"isLast"`
	}

	remaining := 50
	fetchAll := false
	if limit > 0 {
		remaining = limit
	}
	if limit < 0 {
		fetchAll = true
	}

	var out jira.ProjectList
	for {
		opts := map[string]string{
			"startAt":    strconv.Itoa(len(out)),
			"maxResults": strconv.Itoa(remaining),
			"expand":     "issueTypes",
		}
		var result searchResult
		err := client.RESTGet("/3/project/search", opts, &result)
		if err != nil {
			return nil, err
		}
		if len(result.Values) > remaining {
			result.Values = result.Values[:remaining]
		}
		out = append(out, result.Values...)
		remaining -= len(result.Values)

		if !fetchAll && remaining == 0 {
			// Got enough.
			return out, nil
		}
		if len(result.Values) == 0 || result.IsLast {
			// Ran out of results.
			return out, nil
		}
	}
}
