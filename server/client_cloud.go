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
	users := []jira.User{}
	params := map[string]string{
		"issueKey": issueKey,
		"query":    query,
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
func (client jiraCloudClient) GetUserGroups(user JIRAUser) ([]*jira.UserGroup, error) {
	groups := []*jira.UserGroup{}
	params := map[string]string{
		"accountId": user.AccountID,
	}
	err := client.RESTGet("3/user/groups", params, &groups)
	if err != nil {
		return nil, err
	}
	return groups, nil
}
