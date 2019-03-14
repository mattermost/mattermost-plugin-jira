// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"path"
	"path/filepath"

	jira "github.com/andygrunwald/go-jira"
	jwt "github.com/rbriski/atlassian-jwt"
	oauth2_jira "golang.org/x/oauth2/jira"
)

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
	OAuthClientId  string `json:"oauthClientId"`
}

func (p *Plugin) handleHTTPAtlassianConnect(w http.ResponseWriter, r *http.Request) (int, error) {
	baseURL := p.externalURL() + "/" + path.Join("plugins", manifest.Id)

	lp := filepath.Join(*p.API.GetConfig().PluginSettings.Directory, manifest.Id, "server", "dist", "templates", "atlassian-connect.json")
	vals := map[string]string{
		"BaseURL": baseURL,
	}
	tmpl, err := template.ParseFiles(lp)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	bb := &bytes.Buffer{}
	err = tmpl.ExecuteTemplate(bb, "config", vals)
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

	var sc SecurityContext
	err = json.Unmarshal(body, &sc)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	p.sc = sc

	// TODO in a cluster situation, other instances should be notified and re-configure
	// themselves
	appErr := p.API.KVSet(KEY_SECURITY_CONTEXT, body)
	fmt.Printf("<><> SecurityContext payload (%v) (%v): %v\n", appErr, len(body), string(body))

	// Attempted to auto load the project keys but the jira client was failing for some reason
	// Need to look into it some more later

	/*if jiraClient, _ := p.getJIRAClientForServer(); jiraClient != nil {
	        fmt.Println("HIT0")
	        req, _ := jiraClient.NewRawRequest(http.MethodGet, "/rest/api/2/project", nil)
	        list1 := jira.ProjectList{}
	        _, err1 := jiraClient.Do(req, &list1)
	        if err1 != nil {
	                fmt.Println(err1.Error())
	        }

	        fmt.Println(list1)

	        if list, resp, err := jiraClient.Project.GetList(); err == nil {
	                fmt.Println("HIT1")
	                keys := []string{}
	                for _, proj := range *list {
	                        keys = append(keys, proj.Key)
	                }
	                p.projectKeys = keys
	                fmt.Println(p.projectKeys)
	        } else {
	                body, _ := ioutil.ReadAll(resp.Body)
	                fmt.Println(string(body))
	                fmt.Println(err.Error())
	        }
	}*/

	json.NewEncoder(w).Encode([]string{"OK"})
	return http.StatusOK, nil
}

func (p *Plugin) handleHTTPUninstalled(w http.ResponseWriter, r *http.Request) (int, error) {
	json.NewEncoder(w).Encode([]string{"OK"})
	return http.StatusOK, nil
}

func (p *Plugin) loadSecurityContext() error {
	// Since .sc is not a pointer, use .Key to check if it's already loaded
	if p.sc.Key != "" {
		return nil
	}

	b, apperr := p.API.KVGet(KEY_SECURITY_CONTEXT)
	if apperr != nil {
		return apperr
	}
	var sc SecurityContext
	err := json.Unmarshal(b, &sc)
	if err != nil {
		return err
	}
	p.sc = sc
	return nil
}

// Creates a client for acting on behalf of a user
func (p *Plugin) getJIRAClientForUser(jiraUser string) (*jira.Client, *http.Client, error) {
	err := p.loadSecurityContext()
	if err != nil {
		return nil, nil, err
	}

	c := oauth2_jira.Config{
		BaseURL: p.sc.BaseURL,
		Subject: jiraUser,
	}

	c.Config.ClientID = p.sc.OAuthClientId
	c.Config.ClientSecret = p.sc.SharedSecret
	c.Config.Endpoint.AuthURL = "https://auth.atlassian.io"
	c.Config.Endpoint.TokenURL = "https://auth.atlassian.io/oauth2/token"

	httpClient := c.Client(context.Background())

	jiraClient, err := jira.NewClient(httpClient, c.BaseURL)
	return jiraClient, httpClient, err
}

// Creates a client with a JWT
func (p *Plugin) getJIRAClientForServer() (*jira.Client, error) {
	err := p.loadSecurityContext()
	if err != nil {
		return nil, err
	}

	c := &jwt.Config{
		Key:          p.sc.Key,
		ClientKey:    p.sc.ClientKey,
		SharedSecret: p.sc.SharedSecret,
		BaseURL:      p.sc.BaseURL,
	}

	return jira.NewClient(c.Client(), c.BaseURL)
}
