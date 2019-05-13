// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
)

const userRedirectPageKey = "user-redirect"

func httpACJSON(p *Plugin, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return http.StatusMethodNotAllowed,
			errors.New("method " + r.Method + " is not allowed, must be GET")
	}

	return p.respondWithTemplate(w, r, "application/json", map[string]string{
		"BaseURL":                      p.GetPluginURL(),
		"RouteACJSON":                  routeACJSON,
		"RouteACInstalled":             routeACInstalled,
		"RouteACUninstalled":           routeACUninstalled,
		"RouteACUserRedirectWithToken": routeACUserRedirectWithToken,
		"UserRedirectPageKey":          userRedirectPageKey,
		"ExternalURL":                  p.GetSiteURL(),
		"PluginKey":                    p.GetPluginKey(),
	})
}

func httpACInstalled(p *Plugin, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodPost {
		return http.StatusMethodNotAllowed,
			errors.New("method " + r.Method + " is not allowed, must be POST")
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to decode request")
	}

	var asc AtlassianSecurityContext
	err = json.Unmarshal(body, &asc)
	if err != nil {
		return http.StatusBadRequest,
			errors.WithMessage(err, "failed to unmarshal request")
	}

	// Only allow this operation once, a JIRA instance must already exist
	// for asc.BaseURL but its EventType would be empty.
	ji, err := p.LoadJIRAInstance(asc.BaseURL)
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to load instance "+asc.BaseURL)
	}
	if ji == nil {
		return http.StatusNotFound,
			errors.Errorf("Jira instance %s must first be added to Mattermost", asc.BaseURL)
	}
	jci, ok := ji.(*jiraCloudInstance)
	if !ok {
		return http.StatusBadRequest, errors.New("Must be a JIRA Cloud instance, is " + ji.GetType())
	}
	if jci.Installed {
		return http.StatusForbidden,
			errors.Errorf("Jira instance %s is already installed", asc.BaseURL)
	}

	// Create a permanent instance record, also store it as current
	jiraInstance := NewJIRACloudInstance(p, asc.BaseURL, true, string(body), &asc)
	err = p.StoreJIRAInstance(jiraInstance)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	err = p.StoreCurrentJIRAInstance(jiraInstance)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	err = json.NewEncoder(w).Encode([]string{"OK"})
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to encode response")
	}
	return http.StatusOK, nil
}

func httpACUninstalled(p *Plugin, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodPost {
		return http.StatusMethodNotAllowed,
			errors.New("method " + r.Method + " is not allowed, must be POST")
	}

	_ = json.NewEncoder(w).Encode([]string{"OK"})
	return http.StatusOK, nil
}
