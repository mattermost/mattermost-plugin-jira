// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

func (p *Plugin) httpGetAutoCompleteFields(w http.ResponseWriter, r *http.Request) (int, error) {
	mattermostUserID := r.Header.Get("Mattermost-User-Id")
	instanceID := r.FormValue("instance_id")
	params := map[string]string{
		"fieldName":  r.FormValue("fieldName"),
		"fieldValue": r.FormValue("fieldValue"),
	}

	client, _, _, err := p.getClient(types.ID(instanceID), types.ID(mattermostUserID))
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	results, err := client.SearchAutoCompleteFields(params)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	if results == nil {
		return respondErr(w, http.StatusInternalServerError, errors.New("failed to return any results"))
	}

	bb, err := json.Marshal(results)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, errors.WithMessage(err, "failed to marshal response"))
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(bb)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, errors.WithMessage(err, "failed to write response"))
	}
	return http.StatusOK, nil
}

func (p *Plugin) httpGetSearchUsers(w http.ResponseWriter, r *http.Request) (int, error) {
	mattermostUserID := r.Header.Get("Mattermost-User-Id")
	instanceID := r.FormValue("instance_id")
	projectKey := r.FormValue("project")
	userSearch := r.FormValue("q")

	client, _, _, err := p.getClient(types.ID(instanceID), types.ID(mattermostUserID))
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	// Get list of assignable users
	jiraUsers, err := client.SearchUsersAssignableInProject(projectKey, userSearch, 10)
	if StatusCode(err) == 401 {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	if jiraUsers == nil {
		return respondErr(w, http.StatusInternalServerError, errors.New("failed to return any results"))
	}

	bb, err := json.Marshal(jiraUsers)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, errors.WithMessage(err, "failed to marshal response"))
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(bb)
	if err != nil {
		return http.StatusInternalServerError, errors.WithMessage(err, "failed to write response")
	}
	return http.StatusOK, nil
}
