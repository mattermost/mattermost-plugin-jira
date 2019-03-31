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

type AtlassianSecurityContext struct {
	Key            string `json:"key"`
	ClientKey      string `json:"clientKey"`
	PublicKey      string `json:"publicKey"`
	SharedSecret   string `json:"sharedSecret"`
	ServerVersion  string `json:"serverVersion"`
	PluginsVersion string `json:"pluginsVersion"`
	BaseURL        string `json:"baseUrl"`
	ProductType    string `json:"productType"`
	Description    string `json:"description"`
	EventType      string `json:"eventType"`
	OAuthClientId  string `json:"oauthClientId"`
}

func (p *Plugin) handleHTTPAtlassianConnect(w http.ResponseWriter, r *http.Request) (int, error) {
	vals := map[string]string{
		"BaseURL": p.externalURL() + "/" + path.Join("plugins", manifest.Id),
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

	var sc AtlassianSecurityContext
	err = json.Unmarshal(body, &sc)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	p.StoreSecurityContext(body)
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
	sc, err := p.LoadSecurityContext()
	if err != nil {
		return nil, nil, err
	}

	c := oauth2_jira.Config{
		BaseURL: sc.BaseURL,
		Subject: jiraUser,
	}

	c.Config.ClientID = sc.OAuthClientId
	c.Config.ClientSecret = sc.SharedSecret
	c.Config.Endpoint.AuthURL = "https://auth.atlassian.io"
	c.Config.Endpoint.TokenURL = "https://auth.atlassian.io/oauth2/token"

	httpClient := c.Client(context.Background())

	jiraClient, err := jira.NewClient(httpClient, c.BaseURL)
	return jiraClient, httpClient, err
}

// Creates a client with a JWT
func (p *Plugin) getJIRAClientForServer() (*jira.Client, error) {
	sc, err := p.LoadSecurityContext()
	if err != nil {
		return nil, err
	}

	c := &jwt.Config{
		Key:          sc.Key,
		ClientKey:    sc.ClientKey,
		SharedSecret: sc.SharedSecret,
		BaseURL:      sc.BaseURL,
	}

	return jira.NewClient(c.Client(), c.BaseURL)
}
