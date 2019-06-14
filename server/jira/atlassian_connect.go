// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"encoding/json"
	"net/http"
)

const userRedirectPageKey = "user-redirect"

var httpACJSON = []ActionFunc{
	RequireHTTPGet,
	handleACInstalled,
}

func handleACJSON(a Action, ac *ActionContext) error {
	return httpRespondTemplateForPath(a,
		"application/json",
		map[string]string{
			"BaseURL":                      ac.PluginConfig.PluginURL,
			"RouteACJSON":                  routeACJSON,
			"RouteACInstalled":             routeACInstalled,
			"RouteACUninstalled":           routeACUninstalled,
			"RouteACUserRedirectWithToken": routeACUserRedirectWithToken,
			"UserRedirectPageKey":          userRedirectPageKey,
			"ExternalURL":                  ac.PluginConfig.SiteURL,
			"PluginKey":                    ac.PluginConfig.PluginKey,
		})
}

var httpACInstalled = []ActionFunc{
	RequireHTTPPost,
	handleACInstalled,
}

func handleACInstalled(a Action, ac *ActionContext) error {
	body, err := httpReadRequestBody(a, ac)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err,
			"failed to decode request")
	}

	var asc AtlassianSecurityContext
	err = json.Unmarshal(body, &asc)
	if err != nil {
		return a.RespondError(http.StatusBadRequest, err,
			"failed to unmarshal request")
	}

	// Only allow this operation once, a Jira instance must already exist
	// for asc.BaseURL but not Installed.
	instance, err := ac.InstanceStore.LoadInstance(asc.BaseURL)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err,
			"failed to load instance %q", asc.BaseURL)
	}
	if instance == nil {
		return a.RespondError(http.StatusNotFound, nil,
			"Jira instance %q must first be added to Mattermost", asc.BaseURL)
	}
	cloudInstance, ok := instance.(*jiraCloudInstance)
	if !ok {
		return a.RespondError(http.StatusBadRequest, nil,
			"Must be a Jira Cloud instance, is %q", instance.GetType())
	}
	if cloudInstance.Installed {
		return a.RespondError(http.StatusForbidden, nil,
			"Jira instance %q is already installed", asc.BaseURL)
	}

	// Create a permanent instance record, also store it as current
	jiraInstance := NewCloudInstance(asc.BaseURL, true, string(body), &asc)
	// StoreInstance also updates the list of known Jira instances
	err = ac.InstanceStore.StoreInstance(jiraInstance)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}
	err = StoreCurrentInstanceAndNotify(ac.API, ac.CurrentInstanceStore, jiraInstance)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}

	return a.RespondJSON([]string{"OK"})
}
