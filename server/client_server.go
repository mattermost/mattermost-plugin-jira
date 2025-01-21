// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"fmt"
	"net/http"

	jira "github.com/andygrunwald/go-jira"
	"github.com/blang/semver/v4"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/pkg/errors"
)

const (
	APIEndpointGetServerInfo           = "rest/api/2/serverInfo"
	APIEndpointCreateIssueMeta         = "rest/api/2/issue/createmeta/"
	JiraVersionWithOldIssueAPIBreaking = "9.0.0"
	QueryParamIssueTypes               = "issueTypes"
)

type jiraServerClient struct {
	JiraClient
}

type searchResult struct {
	IssueTypes []jira.IssueType `json:"issueTypes"`
}

func newServerClient(jiraClient *jira.Client) Client {
	return &jiraServerClient{
		JiraClient: JiraClient{
			Jira: jiraClient,
		},
	}
}

type ProjectIssueInfo struct {
	Values []*jira.MetaIssueType `json:"values"`
}

type FieldInfo struct {
	Values []map[string]interface{} `json:"values"`
}

type JiraServerVersionInfo struct {
	Version string `json:"version"`
}

// GetIssueInfo returns the issues information based on project id.
func (client jiraServerClient) GetIssueInfo(projectID string) (*ProjectIssueInfo, *jira.Response, error) {
	apiEndpoint := fmt.Sprintf("%s%s/issuetypes", APIEndpointCreateIssueMeta, projectID)
	req, err := client.Jira.NewRequest(http.MethodGet, apiEndpoint, nil)
	if err != nil {
		return nil, nil, err
	}

	issues := ProjectIssueInfo{}
	response, err := client.Jira.Do(req, &issues)
	return &issues, response, err
}

func (client jiraServerClient) GetCreateMetaInfoForPivotJiraVersion(api plugin.API, options *jira.GetQueryOptions) (*jira.CreateMetaInfo, *jira.Response, error) {
	projectID := options.ProjectKeys
	proj, resp, err := client.Jira.Project.Get(projectID)
	if err != nil {
		return nil, resp, errors.Wrap(err, "failed to get project for CreateMetaInfo")
	}

	issueInfo, resp, err := client.GetIssueInfo(proj.ID)
	if err != nil {
		return nil, resp, errors.Wrap(err, "failed to get create meta info")
	}

	for _, issueType := range issueInfo.Values {
		apiEndpoint := fmt.Sprintf("%s%s/issuetypes/%s", APIEndpointCreateIssueMeta, proj.ID, issueType.Id)
		req, rErr := client.Jira.NewRequest(http.MethodGet, apiEndpoint, nil)
		if rErr != nil {
			api.LogDebug("Failed to get the issue type.", "IssueType", issueType, "Error", rErr.Error())
			continue
		}

		fieldInfo := FieldInfo{}
		resp, err = client.Jira.Do(req, &fieldInfo)
		if err != nil {
			api.LogDebug("Failed to get the response for field info.", "Error", err.Error())
			continue
		}

		fieldMap := make(map[string]interface{})
		for _, fieldValue := range fieldInfo.Values {
			fieldMap[fmt.Sprintf("%v", fieldValue["fieldId"])] = fieldValue
		}
		issueType.Fields = fieldMap
	}
	project := &jira.MetaProject{
		Expand:     proj.Expand,
		Self:       proj.Self,
		Id:         proj.ID,
		Key:        proj.Key,
		Name:       proj.Name,
		IssueTypes: issueInfo.Values,
	}

	meta := new(jira.CreateMetaInfo)
	meta.Projects = append(meta.Projects, project)
	return meta, resp, err
}

func (client jiraServerClient) GetCreateMetaInfoForSpecificJiraVersion(api plugin.API, currentVersion, pivotVersion semver.Version, options *jira.GetQueryOptions) (*jira.CreateMetaInfo, *jira.Response, error) {
	if currentVersion.LT(pivotVersion) {
		return client.Jira.Issue.GetCreateMetaWithOptions(options)
	}
	return client.GetCreateMetaInfoForPivotJiraVersion(api, options)
}

// GetCreateMetaInfo returns the metadata needed to implement the UI and validation of
// creating new Jira issues.
func (client jiraServerClient) GetCreateMetaInfo(api plugin.API, options *jira.GetQueryOptions) (*jira.CreateMetaInfo, error) {
	v := new(JiraServerVersionInfo)
	req, err := client.Jira.NewRequest(http.MethodGet, APIEndpointGetServerInfo, nil)
	if err != nil {
		return nil, err
	}

	if _, err = client.Jira.Do(req, v); err != nil {
		return nil, errors.Wrap(err, "failed to fetch Jira server version")
	}

	currentVersion, err := semver.Make(v.Version)
	if err != nil {
		return nil, errors.Wrap(err, "error while parsing version")
	}

	pivotVersion, err := semver.Make(JiraVersionWithOldIssueAPIBreaking)
	if err != nil {
		return nil, errors.Wrap(err, "error while parsing version")
	}

	info, resp, err := client.GetCreateMetaInfoForSpecificJiraVersion(api, currentVersion, pivotVersion, options)
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
	return info, nil
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

func (client jiraServerClient) ListProjects(query string, limit int, expandIssueTypes bool) (jira.ProjectList, error) {
	queryOptions := &jira.GetQueryOptions{}
	if expandIssueTypes {
		queryOptions.Expand = QueryParamIssueTypes
	}

	pList, resp, err := client.Jira.Project.ListWithOptions(queryOptions)
	if err != nil {
		return nil, userFriendlyJiraError(resp, err)
	}
	if pList == nil {
		return jira.ProjectList{}, nil
	}
	result := *pList
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (client jiraServerClient) GetIssueTypes(projectID string) ([]jira.IssueType, error) {
	var result searchResult
	opts := map[string]string{
		"expand": "issueTypes",
	}

	if err := client.RESTGet(fmt.Sprintf("2/project/%s", projectID), opts, &result); err != nil {
		return nil, err
	}

	return result.IssueTypes, nil
}

func (client jiraServerClient) ListProjectStatuses(projectID string) ([]*IssueTypeWithStatuses, error) {
	var result []*IssueTypeWithStatuses
	if err := client.RESTGet(fmt.Sprintf("2/project/%s/statuses", projectID), nil, &result); err != nil {
		return nil, err
	}

	return result, nil
}
