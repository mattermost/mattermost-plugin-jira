// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

const userRedirectPageKey = "user-redirect"

func (p *Plugin) httpACJSON(w http.ResponseWriter, r *http.Request, instanceID types.ID) (int, error) {
	return p.respondTemplate(w, r, "application/json", map[string]string{
		"BaseURL":                      p.GetPluginURL(),
		"RouteACJSON":                  instancePath(routeACJSON, instanceID),
		"RouteACInstalled":             routeACInstalled,
		"RouteACUninstalled":           routeACUninstalled,
		"RouteACUserRedirectWithToken": instancePath(routeACUserRedirectWithToken, instanceID),
		"UserRedirectPageKey":          userRedirectPageKey,
		"ExternalURL":                  p.GetSiteURL(),
		"PluginKey":                    p.GetPluginKey(),
	})
}

func (p *Plugin) httpACInstalled(w http.ResponseWriter, r *http.Request) (int, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.WithMessage(err, "failed to decode request"))
	}

	var asc AtlassianSecurityContext
	err = json.Unmarshal(body, &asc)
	if err != nil {
		return respondErr(w, http.StatusBadRequest,
			errors.WithMessage(err, "failed to unmarshal request"))
	}
	instanceID := types.ID(asc.BaseURL)

	// Only allow this operation once, a JIRA instance must already exist
	// for asc.BaseURL, but not already installed.
	instance, err := p.instanceStore.LoadInstance(instanceID)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.WithMessage(err, "failed to load instance "+asc.BaseURL))
	}
	if instance == nil {
		return respondErr(w, http.StatusNotFound,
			errors.Errorf("Jira instance %s must first be added to Mattermost", asc.BaseURL))
	}
	ci, ok := instance.(*cloudInstance)
	if !ok {
		return respondErr(w, http.StatusBadRequest,
			errors.Errorf("Must be a JIRA Cloud instance, is %s", instance.Common().Type))
	}
	if ci.Installed {
		return respondErr(w, http.StatusForbidden,
			errors.Errorf("Jira instance %s is already installed", asc.BaseURL))
	}

	// Create a permanent instance record, also store it as current
	newInstance := newCloudInstance(p, instanceID, true, string(body), &asc)
	err = p.InstallInstance(newInstance)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	// Setup autolink
	err = p.AddAutolinksForCloudInstance(newInstance)
	if err != nil {
		p.client.Log.Info("could not install autolinks for cloud instance", "instance", ci.BaseURL, "err", err)
	}

	_ = p.setupFlow.ForUser(ci.SetupWizardUserID).Go(stepInstalledJiraApp)

	return respondJSON(w, []string{"OK"})
}

func (p *Plugin) httpACUninstalled(w http.ResponseWriter, r *http.Request) (int, error) {
	// Just send an ok to the Jira server, even though we're not doing anything.
	return respondJSON(w, []string{"OK"})
}
