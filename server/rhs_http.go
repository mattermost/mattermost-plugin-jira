// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"encoding/json"
	"net/http"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

const (
	rhsErrNotConnected   = "not_connected"
	rhsErrRateLimited    = "rate_limited"
	rhsErrNotAuthorized  = "not_authorized"
	rhsErrNotCloud       = "not_cloud"
	rhsErrInvalidRequest = "invalid_request"
	rhsErrInternal       = "internal_error"

	queryRHSTabKind       = "tab_kind"
	queryRHSTabKey        = "tab_key"
	queryRHSTabID         = "tab_id"
	queryRHSSort          = "sort"
	queryRHSNextPageToken = "next_page_token"

	rhsDefaultSort = "updated"
)

type rhsJSONError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func respondJSONErr(w http.ResponseWriter, status int, errorCode, message string) (int, error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	encErr := json.NewEncoder(w).Encode(rhsJSONError{Error: errorCode, Message: message})
	if encErr != nil {
		return status, errors.Wrap(encErr, errorCode+": "+message)
	}
	return status, errors.Errorf("%s: %s", errorCode, message)
}

func (p *Plugin) checkSystemAdmin(userID string) bool {
	return p.client.User.HasPermissionTo(userID, model.PermissionManageSystem)
}

func respondRHSErr(w http.ResponseWriter, err error) (int, error) {
	switch {
	case errors.Is(err, kvstore.ErrNotFound):
		return respondJSONErr(w, http.StatusUnauthorized, rhsErrNotConnected, "Jira account is not connected")
	case errors.Is(err, ErrRateLimited):
		return respondJSONErr(w, http.StatusTooManyRequests, rhsErrRateLimited, "Jira is rate limiting requests, try again shortly")
	case errors.Is(err, ErrRHSNotCloud):
		return respondJSONErr(w, http.StatusBadRequest, rhsErrNotCloud, "Jira RHS is available for Jira Cloud only")
	case errors.Is(err, ErrInvalidStatusCategory),
		errors.Is(err, ErrInvalidRHSSort),
		errors.Is(err, ErrUnknownRHSTabKind),
		errors.Is(err, ErrInvalidRHSTab):
		return respondJSONErr(w, http.StatusBadRequest, rhsErrInvalidRequest, err.Error())
	default:
		status, _ := respondJSONErr(w, http.StatusInternalServerError, rhsErrInternal, "internal error")
		return status, err
	}
}

func (p *Plugin) httpRHSGetIssues(w http.ResponseWriter, r *http.Request) (int, error) {
	userID := r.Header.Get(HeaderMattermostUserID)
	instanceID := types.ID(r.FormValue(ParamInstanceID))
	result, err := p.getRHSIssues(
		instanceID,
		types.ID(userID),
		r.FormValue(queryRHSTabKind),
		r.FormValue(queryRHSTabKey),
		r.FormValue(queryRHSTabID),
		r.FormValue(queryRHSSort),
		r.FormValue(queryRHSNextPageToken),
	)
	if err != nil {
		return respondRHSErr(w, err)
	}
	return respondJSON(w, result)
}

func (p *Plugin) httpRHSListStatuses(w http.ResponseWriter, r *http.Request) (int, error) {
	userID := r.Header.Get(HeaderMattermostUserID)
	if !p.checkSystemAdmin(userID) {
		return respondJSONErr(w, http.StatusForbidden, rhsErrNotAuthorized, "not authorized")
	}
	instanceID := types.ID(r.FormValue(ParamInstanceID))
	result, err := p.getRHSStatuses(instanceID, types.ID(userID))
	if err != nil {
		return respondRHSErr(w, err)
	}
	return respondJSON(w, result)
}
