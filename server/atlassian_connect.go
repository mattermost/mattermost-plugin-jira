// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

const userRedirectPageKey = "user-redirect"

var httpACJSON = []ActionFunc{
	RequireHTTPGet,
	handleACInstalled,
}

func handleACJSON(a *Action) error {
	return a.RespondTemplate(
		a.HTTPRequest.URL.Path,
		"application/json",
		map[string]string{
			"BaseURL":                      a.PluginConfig.PluginURL,
			"RouteACJSON":                  routeACJSON,
			"RouteACInstalled":             routeACInstalled,
			"RouteACUninstalled":           routeACUninstalled,
			"RouteACUserRedirectWithToken": routeACUserRedirectWithToken,
			"UserRedirectPageKey":          userRedirectPageKey,
			"ExternalURL":                  a.PluginConfig.SiteURL,
			"PluginKey":                    a.PluginConfig.PluginKey,
		})
}

var httpACInstalled = []ActionFunc{
	RequireHTTPPost,
	handleACInstalled,
}

func handleACInstalled(a *Action) error {
	body, err := ioutil.ReadAll(a.HTTPRequest.Body)
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

	// Only allow this operation once, a JIRA instance must already exist
	// for asc.BaseURL but not Installed.
	instance, err := a.InstanceStore.LoadJIRAInstance(asc.BaseURL)
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
			"Must be a JIRA Cloud instance, is %q", instance.GetType())
	}
	if cloudInstance.Installed {
		return a.RespondError(http.StatusForbidden, nil,
			"Jira instance %q is already installed", asc.BaseURL)
	}

	// Create a permanent instance record, also store it as current
	jiraInstance := NewJIRACloudInstance(asc.BaseURL, true, string(body), &asc)
	// StoreJIRAInstance also updates the list of known Jira instances
	err = a.InstanceStore.StoreJIRAInstance(jiraInstance)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}
	err = StoreCurrentJIRAInstanceAndNotify(a.API, a.CurrentInstanceStore, jiraInstance)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}

	return a.RespondJSON([]string{"OK"})
}
