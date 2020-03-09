// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
)

type Label struct {
	Value       string `json:"value"`
	DisplayName string `json:"displayName"`
}

type LabelResult struct {
	Results []Label `json:"results"`
}

func httpAPIGetLabels(ji Instance, w http.ResponseWriter, r *http.Request) (int, error) {
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

	val := r.FormValue("fieldValue")

	labels, err := client.GetLabels(val)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	bb, err := json.Marshal(labels)
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
