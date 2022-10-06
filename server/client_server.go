// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"net/http"

	jira "github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"
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

// GetCreateMeta returns the metadata needed to implement the UI and validation of
// creating new Jira issues.
func (client jiraServerClient) GetCreateMeta(options *jira.GetQueryOptions) (*jira.CreateMetaInfo, error) {
	cimd, resp, err := client.Jira.Issue.GetCreateMetaWithOptions(options)
	if err != nil {
		if resp == nil {
			return nil, err
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
			err = errors.New("not authorized to create issues")
		}
		return nil, RESTError{err, resp.StatusCode}
	}
	return cimd, nil
}

// SearchUsersAssignableToIssue finds all users that can be assigned to an issue.
func (client jiraServerClient) SearchUsersAssignableToIssue(issueKey, query string, maxResults int) ([]jira.User, error) {
	return SearchUsersAssignableToIssue(client, issueKey, "username", query, maxResults)
}

// SearchUsersAssignableInProject finds all users that can be assigned to some issue in a given project.
func (client jiraServerClient) SearchUsersAssignableInProject(projectKey, query string, maxResults int) ([]jira.User, error) {
	return SearchUsersAssignableInProject(client, projectKey, "username", query, maxResults)
}

// GetUserGroups returns the list of groups that a user belongs to.
func (client jiraServerClient) GetUserGroups(connection *Connection) ([]*jira.UserGroup, error) {
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

func (client jiraServerClient) ListProjects(query string, limit int) (jira.ProjectList, error) {
	plist, resp, err := client.Jira.Project.GetList()
	if err != nil {
		return nil, userFriendlyJiraError(resp, err)
	}
	if plist == nil {
		return jira.ProjectList{}, nil
	}
	result := *plist
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}
