// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"fmt"
	"net/http"

	"github.com/pkg/errors"
)

// cloudInstance is a Jira Cloud instance installed through the Atlassian
// Connect app. Connect is no longer supported, so these instances are only
// loaded to tell admins to reinstall them with OAuth 2.0.
type cloudInstance struct {
	*InstanceCommon

	RawAtlassianSecurityContext string
	*AtlassianSecurityContext   `json:"-"`
}

var _ Instance = (*cloudInstance)(nil)

type AtlassianSecurityContext struct {
	Key            string `json:"key"`
	ClientKey      string `json:"clientKey"`
	SharedSecret   string `json:"sharedSecret"`
	ServerVersion  string `json:"serverVersion"`
	PluginsVersion string `json:"pluginsVersion"`
	BaseURL        string `json:"baseUrl"`
	ProductType    string `json:"productType"`
	Description    string `json:"description"`
	EventType      string `json:"eventType"`
	OAuthClientID  string `json:"oauthClientId"`
}

var errConnectInstanceUnsupported = errors.New("this Jira Cloud instance was installed with the Atlassian Connect app, which is no longer supported. " +
	"Ask a Mattermost system admin to reinstall it by running `/jira setup` and choosing Jira Cloud (OAuth 2.0)")

func (ci *cloudInstance) GetMattermostKey() string {
	if ci.AtlassianSecurityContext == nil {
		return ""
	}
	return ci.AtlassianSecurityContext.Key
}

func (ci *cloudInstance) GetDisplayDetails() map[string]string {
	return map[string]string{
		"Status": "Installed with Atlassian Connect, which is no longer supported. Reinstall with `/jira setup`.",
	}
}

func (ci *cloudInstance) GetUserConnectURL(string) (string, *http.Cookie, error) {
	return "", nil, errConnectInstanceUnsupported
}

func (ci *cloudInstance) GetURL() string {
	if ci.AtlassianSecurityContext == nil {
		return ci.InstanceID.String()
	}
	return ci.AtlassianSecurityContext.BaseURL
}

func (ci *cloudInstance) GetJiraBaseURL() string {
	return normalizeJiraBaseURL(ci.GetURL())
}

func (ci *cloudInstance) GetManageAppsURL() string {
	return fmt.Sprintf("%s/plugins/servlet/upm", ci.GetJiraBaseURL())
}

func (ci *cloudInstance) GetManageWebhooksURL() string {
	return cloudManageWebhooksURL(ci.GetJiraBaseURL())
}

func cloudManageWebhooksURL(jiraURL string) string {
	return fmt.Sprintf("%s/plugins/servlet/webhooks", jiraURL)
}

func (ci *cloudInstance) GetClient(*Connection) (Client, error) {
	return nil, errConnectInstanceUnsupported
}
