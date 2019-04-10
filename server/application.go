// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"path"

	jira "github.com/andygrunwald/go-jira"
	jwt "github.com/rbriski/atlassian-jwt"
	oauth2_jira "golang.org/x/oauth2/jira"
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
	jiraInstance := JIRAInstance{
		Key:                         asc.BaseURL,
		Type:                        JIRACloudType,
		AtlassianSecurityContextRaw: string(body),
		asc:                         &asc,
	}
	err = p.StoreJIRAInstance(jiraInstance, true)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	// Update the known instances
	known, err := p.LoadKnownJIRAInstances()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	known[jiraInstance.Key] = jiraInstance.Type
	err = p.StoreKnownJIRAInstances(known)
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

// Creates a client for acting on behalf of a user
func (p *Plugin) getJIRAClientForUser(jiraUser string) (*jira.Client, *http.Client, error) {
	jiraInstance, err := p.LoadCurrentJIRAInstance()
	if err != nil {
		return nil, nil, err
	}

	c := oauth2_jira.Config{
		BaseURL: jiraInstance.asc.BaseURL,
		Subject: jiraUser,
	}

	c.Config.ClientID = jiraInstance.asc.OAuthClientId
	c.Config.ClientSecret = jiraInstance.asc.SharedSecret
	c.Config.Endpoint.AuthURL = "https://auth.atlassian.io"
	c.Config.Endpoint.TokenURL = "https://auth.atlassian.io/oauth2/token"

	httpClient := c.Client(context.Background())

	jiraClient, err := jira.NewClient(httpClient, c.BaseURL)
	return jiraClient, httpClient, err
}

// Creates a "bot" client with a JWT
func (p *Plugin) getJIRAClientForServer() (*jira.Client, error) {
	jiraInstance, err := p.LoadCurrentJIRAInstance()
	if err != nil {
		return nil, err
	}

	c := &jwt.Config{
		Key:          jiraInstance.asc.Key,
		ClientKey:    jiraInstance.asc.ClientKey,
		SharedSecret: jiraInstance.asc.SharedSecret,
		BaseURL:      jiraInstance.asc.BaseURL,
	}

	return jira.NewClient(c.Client(), c.BaseURL)
}
