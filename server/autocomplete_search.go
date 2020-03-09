// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
)

type Result struct {
	Value       string `json:"value"`
	DisplayName string `json:"displayName"`
}

type AutoCompleteResult struct {
	Results []Result `json:"results"`
}

func httpAPIGetAutoCompleteFields(ji Instance, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return http.StatusMethodNotAllowed,
			errors.New("Request: " + r.Method + " is not allowed, must be GET")
	}

	mattermostUserID := r.Header.Get("Mattermost-User-Id")
	if mattermostUserID == "" {
		return http.StatusUnauthorized, errors.New("not authorized")
	}

	jiraUser, err := ji.GetPlugin().userStore.LoadJIRAUser(ji, mattermostUserID)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	client, err := ji.GetClient(jiraUser)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	params := map[string]string{
		"fieldName":  r.FormValue("fieldName"),
		"fieldValue": r.FormValue("fieldValue"),
	}

	results, err := client.SearchAutoCompleteFields(&AutoCompleteResult{}, params)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	if results == nil {
		return http.StatusInternalServerError, errors.New("failed to return any results")
	}

	bb, err := json.Marshal(results)
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to marshal response")
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(bb)
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to write response")
	}
	return http.StatusOK, nil
}
