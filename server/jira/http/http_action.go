// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/mattermost/mattermost-server/plugin"
	"github.com/pkg/errors"
)

type HTTPAction struct {
	action

	Request            *http.Request
	ResponseWriter     http.ResponseWriter
	ResponseStatusCode int
}

var _ Action = (*HTTPAction)(nil)

func NewHTTPActtion(p *Plugin, c *plugin.Context, r *http.Request, w http.ResponseWriter) *HTTPAction {
	mattermostUserId := r.Header.Get("Mattermost-User-Id")

	return &HTTPAction{
		action:         *newAction(p, c, mattermostUserId),
		Request:        r,
		ResponseWriter: w,
	}
}

func (httpAction HTTPAction) FormValue(key string) string {
	return httpAction.Request.FormValue(key)
}

func (httpAction HTTPAction) RespondError(code int, err error, wrap ...interface{}) error {
	if len(wrap) > 0 {
		fmt := wrap[0].(string)
		if err != nil {
			err = errors.WithMessagef(err, fmt, wrap[1:]...)
		} else {
			err = errors.Errorf(fmt, wrap[1:]...)
		}
	}

	if err == nil {
		return nil
	}

	httpAction.ResponseStatusCode = code
	http.Error(httpAction.ResponseWriter, err.Error(), code)
	return err
}

func (httpAction HTTPAction) RespondPrintf(format string, args ...interface{}) error {
	text := fmt.Sprintf(format, args...)
	httpAction.ResponseWriter.Header().Set("Content-Type", "text/plain")
	_, err := httpAction.ResponseWriter.Write([]byte(text))
	if err != nil {
		return httpAction.RespondError(http.StatusInternalServerError, err,
			"failed to write response")
	}
	return nil
}

func (httpAction HTTPAction) RespondRedirect(redirectURL string) error {
	status := http.StatusFound
	if httpAction.Request.Method != http.MethodGet {
		status = http.StatusTemporaryRedirect
	}
	http.Redirect(httpAction.ResponseWriter, httpAction.Request, redirectURL, status)
	httpAction.ResponseStatusCode = status
	return nil
}

func (httpAction HTTPAction) RespondTemplate(templateKey, contentType string, values interface{}) error {
	t := httpAction.PluginConfig.Templates[templateKey]
	if t == nil {
		return httpAction.RespondError(http.StatusInternalServerError, nil,
			"no template found for %q", templateKey)
	}
	httpAction.ResponseWriter.Header().Set("Content-Type", contentType)
	err := t.Execute(httpAction.ResponseWriter, values)
	if err != nil {
		return httpAction.RespondError(http.StatusInternalServerError, err,
			"failed to write response")
	}
	return nil
}

func (httpAction HTTPAction) RespondJSON(value interface{}) error {
	httpAction.ResponseWriter.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(httpAction.ResponseWriter).Encode(value)
	if err != nil {
		return httpAction.RespondError(http.StatusInternalServerError, err,
			"failed to write response")
	}
	return nil
}

func RequireHTTPGet(a Action, ac *ActionContext) error {
	return httpRequireMethod(a, ac, http.MethodGet)
}

func RequireHTTPPost(a Action, ac *ActionContext) error {
	return httpRequireMethod(a, ac, http.MethodPost)
}

func httpRequireMethod(a Action, ac *ActionContext, method string) error {
	httpAction, ok := a.(*HTTPAction)
	if !ok {
		a.RespondError(http.StatusInternalServerError, nil, "Wrong action type %T, eexpected HTTPAction", a)
	}
	if httpAction.Request.Method != method {
		return a.RespondError(http.StatusMethodNotAllowed, nil,
			"method %s is not allowed, must be %s", httpAction.Request.Method, method)
	}
	return nil
}

func httpRespondTemplateForPath(a Action, contentType string, values interface{}) error {
	httpAction, ok := a.(HTTPAction)
	if !ok {
		a.RespondError(http.StatusInternalServerError, nil, "Wrong action type %T, eexpected HTTPAction", a)
	}
	return a.RespondTemplate(httpAction.Request.URL.Path, contentType, values)
}

func httpReadRequestBody(a Action, ac *ActionContext) ([]byte, error) {
	httpAction, ok := a.(HTTPAction)
	if !ok {
		return nil, errors.Errorf("Wrong action type %T, eexpected HTTPAction", a)
	}
	body, err := ioutil.ReadAll(httpAction.Request.Body)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to read request")
	}
	return body, nil
}
