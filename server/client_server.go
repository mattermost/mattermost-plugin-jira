// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"strconv"

	jira "github.com/andygrunwald/go-jira"
)

type jiraServerClient struct {
	JiraClient
}

func newServerClient(jiraClient *jira.Client) Client {
	return &jiraServerClient{
		JiraClient: JiraClient{
			Jira: jiraClient,
		},
	}
}

// SearchUsersAssignableToIssue finds all users that can be assigned to an issue.
func (client JiraClient) SearchUsersAssignableToIssue(issueKey, query string, maxResults int) ([]jira.User, error) {
	users := []jira.User{}
	params := map[string]string{
		"issueKey": issueKey,
		"username": query,
	}
	if maxResults > 0 {
		params["maxResults"] = strconv.Itoa(maxResults)
	}
	err := client.RESTGet("2/user/assignable/search", params, &users)
	if err != nil {
		return nil, err
	}
	return users, nil
}

// GetUserGroups returns the list of groups that a user belongs to.
func (client jiraServerClient) GetUserGroups(user JIRAUser) ([]*jira.UserGroup, error) {
	var result struct {
		Groups struct {
			Items []*jira.UserGroup
		}
	}
	err := client.RESTGet("2/myself", map[string]string{"expand": "groups"}, &result)
	if err != nil {
		return nil, err
	}
	return result.Groups.Items, nil
}
