// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/pkg/errors"
)

const userRedirectPageKey = "user-redirect"

var regexpNonAlnum = regexp.MustCompile("[^a-zA-Z0-9]+")

func httpACJSON(p *Plugin, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return http.StatusMethodNotAllowed,
			errors.New("method " + r.Method + " is not allowed, must be GET")
	}

	enc := func(in string) string {
		return regexpNonAlnum.ReplaceAllString(in, "-")
	}

	return respondWithTemplate(w, r, p.templates, "application/json", map[string]string{
		"BaseURL":                      p.GetPluginURL(),
		"RouteACJSON":                  routeACJSON,
		"RouteACInstalled":             routeACInstalled,
		"RouteACUninstalled":           routeACUninstalled,
		"RouteACUserRedirectWithToken": routeACUserRedirectWithToken,
		"UserRedirectPageKey":          userRedirectPageKey,
		"ExternalURL":                  p.GetSiteURL(),
		"Key":                          "mattermost-" + enc(p.GetSiteURL()),
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

	// Create or overwrite the instance record, also store it
	// as current
	jiraInstance := NewJIRACloudInstance(p, asc.BaseURL, string(body), &asc)
	err = p.StoreJIRAInstance(jiraInstance, true)
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
