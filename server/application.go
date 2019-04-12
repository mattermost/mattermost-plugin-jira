// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"path"
)

func (p *Plugin) handleHTTPAtlassianConnect(w http.ResponseWriter, r *http.Request) (int, error) {
	vals := map[string]string{
		"BaseURL":     p.externalURL() + "/" + path.Join("plugins", manifest.Id),
		"ExternalURL": p.externalURL(),
	}
	bb := &bytes.Buffer{}
	err := p.atlassianConnectTemplate.ExecuteTemplate(bb, "config", vals)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	io.Copy(w, bytes.NewReader(bb.Bytes()))
	return http.StatusOK, nil
}

func (p *Plugin) handleHTTPInstalled(w http.ResponseWriter, r *http.Request) (int, error) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	var asc AtlassianSecurityContext
	err = json.Unmarshal(body, &asc)
	if err != nil {
		return http.StatusBadRequest, err
	}

	// Create or overwrite the instance record, also store it
	// as current
	jiraInstance := NewJIRACloudInstance(asc.BaseURL, string(body), &asc)
	err = p.StoreJIRAInstance(jiraInstance, true)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	json.NewEncoder(w).Encode([]string{"OK"})
	return http.StatusOK, nil
}

func (p *Plugin) handleHTTPUninstalled(w http.ResponseWriter, r *http.Request) (int, error) {
	json.NewEncoder(w).Encode([]string{"OK"})
	return http.StatusOK, nil
}
