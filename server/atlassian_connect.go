// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
)

var regexpNonAlnum = regexp.MustCompile("[^a-zA-Z0-9]+")

func (p *Plugin) handleHTTPAtlassianConnect(w http.ResponseWriter, r *http.Request) (int, error) {
	enc := func(in string) string {
		return regexpNonAlnum.ReplaceAllString(in, "-")
	}

	vals := map[string]string{
		"BaseURL":     p.GetPluginURL(),
		"ExternalURL": p.GetSiteURL(),
		"Key":         "mattermost-" + enc(p.GetSiteURL()),
	}
	bb := &bytes.Buffer{}
	err := p.atlassianConnectTemplate.ExecuteTemplate(bb, "config", vals)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	io.Copy(w, bytes.NewReader(bb.Bytes()))
	p.debugf("Served atlassian-connect.json:\n%s", bb.String())
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
	jiraInstance := NewJIRACloudInstance(p, asc.BaseURL, string(body), &asc)
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
