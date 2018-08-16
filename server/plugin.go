// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

const PluginId = "jira"

const (
	KEY_SECURITY_CONTEXT = "security_context"
)

type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	securityContext *SecurityContext
}

type SecurityContext struct {
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
}

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	config := p.getConfiguration()
	if config.Secret == "" || config.UserName == "" {
		http.Error(w, "JIRA plugin not configured correctly; must provide Secret and UserName", http.StatusForbidden)
		return
	}

	status, err := p.handleHTTPRequest(c, config, w, r)
	if err != nil {
		// panic(err.Error())
		p.API.LogError(strconv.Itoa(status), "Host", r.Host, "RequestURI", r.RequestURI, "Method", r.Method, "query", r.URL.Query().Encode())
		http.Error(w, err.Error(), status)
	}
	p.API.LogDebug("200", "Host", r.Host, "RequestURI", r.RequestURI, "Method", r.Method, "query", r.URL.Query().Encode())
}

func (p *Plugin) handleHTTPRequest(c *plugin.Context, config *configuration, w http.ResponseWriter, r *http.Request) (int, error) {
	switch r.URL.Path {
	case "/webhook",
		"/issue_event":
		return p.serveWebhook(c, config, w, r)
	case "/atlassian-connect.json":
		return p.serveAtlassianConnect(c, config, w, r)
	case "/installed":
		return p.serveInstalled(c, config, w, r)
	}

	return http.StatusNotFound, fmt.Errorf("Not found")
}

func (p *Plugin) serveAtlassianConnect(c *plugin.Context, config *configuration, w http.ResponseWriter, r *http.Request) (int, error) {
	mmConfig := p.API.GetConfig()
	baseURL := *mmConfig.ServiceSettings.SiteURL + "/" + path.Join("plugins", PluginId)

	lp := filepath.Join(*mmConfig.PluginSettings.Directory, PluginId, "server", "dist", "templates", "atlassian-connect.json")
	vals := map[string]string{
		"BaseURL": baseURL,
	}
	tmpl, err := template.ParseFiles(lp)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	tmpl.ExecuteTemplate(w, "config", vals)
	return http.StatusOK, nil
}

func (p *Plugin) serveInstalled(c *plugin.Context, config *configuration, w http.ResponseWriter, r *http.Request) (int, error) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatalf("Can't read request:%v\n", err)
		return http.StatusInternalServerError, err
	}
	var sc SecurityContext
	json.Unmarshal(body, &sc)

	p.securityContext = &sc

	p.API.KVSet(KEY_SECURITY_CONTEXT, body)

	json.NewEncoder(w).Encode([]string{"OK"})
	return http.StatusOK, nil
}

func (p *Plugin) serveWebhook(c *plugin.Context, config *configuration, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodPost {
		return http.StatusMethodNotAllowed,
			fmt.Errorf("Request: " + r.Method + " is not allowed, must be POST")
	}
	if subtle.ConstantTimeCompare([]byte(r.URL.Query().Get("secret")), []byte(config.Secret)) != 1 {
		return http.StatusForbidden,
			fmt.Errorf("Request URL: secret did not match")
	}

	teamName := r.URL.Query().Get("team")
	if teamName == "" {
		return http.StatusBadRequest,
			fmt.Errorf("Request URL: team is empty")
	}
	channelID := r.URL.Query().Get("channel")
	if channelID == "" {
		return http.StatusBadRequest,
			fmt.Errorf("Request URL: channel is empty")
	}

	user, appErr := p.API.GetUserByUsername(config.UserName)
	if appErr != nil {
		return appErr.StatusCode, fmt.Errorf(appErr.Message)
	}

	channel, appErr := p.API.GetChannelByNameForTeamName(teamName, channelID, false)
	if appErr != nil {
		return appErr.StatusCode, fmt.Errorf(appErr.Message)
	}

	initPost, err := AsSlackAttachment(r.Body)
	if err != nil {
		return http.StatusBadRequest, err
	}

	post := &model.Post{
		ChannelId: channel.Id,
		UserId:    user.Id,
		Props: map[string]interface{}{
			"from_webhook":  "true",
			"use_user_icon": "true",
		},
	}
	initPost(post)

	_, appErr = p.API.CreatePost(post)
	if appErr != nil {
		return appErr.StatusCode, fmt.Errorf(appErr.Message)
	}

	return http.StatusOK, nil
}
