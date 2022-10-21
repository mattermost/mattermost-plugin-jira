// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"fmt"
	"net/http"

	jira "github.com/andygrunwald/go-jira"
	"github.com/hashicorp/go-version"
	"github.com/pkg/errors"
)

const (
	ServerInfoApiEndpoint = "rest/api/2/serverInfo"
	CreateMetaAPIEndpoint = "rest/api/2/issue/createmeta/"
	PivotVersion          = "8.4.0"
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

type IssueInfo struct {
	Values []*jira.MetaIssueType `json:"values,omitempty"`
}

type FieldInfo struct {
	Values []interface{} `json:"values,omitempty"`
}

type FieldValues struct {
	FieldID string `json:"fieldId,omitempty"`
}

type FieldID struct {
	Values []FieldValues `json:"values,omitempty"`
}

type ServerVersion struct {
	VersionInfo string `json:"version,omitempty"`
}

// GetIssueInfo returns the issues information based on project id.
func (client jiraServerClient) GetIssueInfo(projectID string) (*IssueInfo, *jira.Response, error) {
	apiEndpoint := fmt.Sprintf("%s%s/issuetypes", CreateMetaAPIEndpoint, projectID)
	req, err := client.Jira.NewRequest(http.MethodGet, apiEndpoint, nil)
	if err != nil {
		return nil, nil, err
	}

	issues := new(IssueInfo)
	response, err := client.Jira.Do(req, issues)
	return issues, response, err
}

// GetCreateMeta returns the metadata needed to implement the UI and validation of
// creating new Jira issues.
func (client jiraServerClient) GetCreateMeta(options *jira.GetQueryOptions) (*jira.CreateMetaInfo, error) {
	v := new(ServerVersion)
	req, err := client.Jira.NewRequest(http.MethodGet, ServerInfoApiEndpoint, nil)
	if err != nil {
		return nil, err
	}

	if _, err = client.Jira.Do(req, v); err != nil {
		return nil, err
	}

	currentVersion, err := version.NewVersion(v.VersionInfo)
	if err != nil {
		return nil, err
	}

	pivotVersion, err := version.NewVersion(PivotVersion)
	if err != nil {
		return nil, err
	}

	var info *jira.CreateMetaInfo
	var resp *jira.Response
	var issues *IssueInfo
	var projectList *jira.ProjectList
	if currentVersion.LessThan(pivotVersion) {
		info, resp, err = client.Jira.Issue.GetCreateMetaWithOptions(options)
	} else {
		projectList, resp, err = client.Jira.Project.ListWithOptions(options)
		meta := new(jira.CreateMetaInfo)

		if err == nil {
			for _, proj := range *projectList {
				meta.Expand = proj.Expand
				issues, resp, err = client.GetIssueInfo(proj.ID)
				if err != nil {
					break
				}

				project := &jira.MetaProject{
					Expand:     proj.Expand,
					Self:       proj.Self,
					Id:         proj.ID,
					Key:        proj.Key,
					Name:       proj.Name,
					IssueTypes: issues.Values,
				}

				for _, issue := range project.IssueTypes {
					apiEndpoint := fmt.Sprintf("%s%s/issuetypes/%s", CreateMetaAPIEndpoint, proj.ID, issue.Id)
					req, err = client.Jira.NewRequest(http.MethodGet, apiEndpoint, nil)
					if err != nil {
						break
					}

					field := new(FieldInfo)
					resp, err = client.Jira.Do(req, field)
					if err != nil {
						break
					}

					fieldID := new(FieldID)
					resp, err = client.Jira.Do(req, fieldID)
					if err != nil {
						break
					}

					fieldMap := make(map[string]interface{})
					for index, fieldValue := range field.Values {
						fieldMap[fieldID.Values[index].FieldID] = fieldValue
					}
					issue.Fields = fieldMap
				}
				meta.Projects = append(meta.Projects, project)
			}
		}
		info = meta
	}

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
