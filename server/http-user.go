// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	jira "github.com/andygrunwald/go-jira"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost-server/model"
)

const (
	WS_EVENT_CONNECT = "connect"

	argMMToken            = "mm_token"
	argAtlassianAccountID = "atlassian_account_id"
	argJIRAUserKey        = "jira_user_key"
	argJIRAUserName       = "jira_user_name"
)

type JIRAUserInfo struct {
	// These fields come from JIRA, so their JSON names must not change.
	Key       string `json:"key,omitempty"`
	AccountId string `json:"accountId,omitempty"`
	Name      string `json:"name,omitempty"`
}

type UserInfo struct {
	JIRAUserInfo
	IsConnected bool   `json:"is_connected,omitempty"`
	JIRAURL     string `json:"jira_url,omitempty"`
}

func (p *Plugin) handleHTTPUserConnect(w http.ResponseWriter, r *http.Request) (int, error) {
	// TODO Enforce a GET
	mattermostUserID := r.Header.Get("Mattermost-User-ID")
	if mattermostUserID == "" {
		return http.StatusUnauthorized, fmt.Errorf("Not authorized")
	}

	sc, err := p.LoadSecurityContext()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	token, err := p.NewEncodedAuthToken(mattermostUserID)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	v := url.Values{}
	v.Add(argMMToken, token)
	redirectURL := fmt.Sprintf("%v/login?dest-url=%v/plugins/servlet/ac/mattermost-plugin/user-config?%v", sc.BaseURL, sc.BaseURL, v.Encode())
	http.Redirect(w, r, redirectURL, http.StatusFound)
	return http.StatusFound, nil
}

func (p *Plugin) handleHTTPUserDisconnect(w http.ResponseWriter, r *http.Request) (int, error) {
	// TODO Enforce a GET
	mattermostUserID := r.Header.Get("Mattermost-User-ID")
	if mattermostUserID == "" {
		return http.StatusUnauthorized, fmt.Errorf("Not authorized")
	}

	info, err := p.LoadJIRAUserInfo(mattermostUserID)
	if err != nil {
		return http.StatusNotFound, err
	}

	err = p.DeleteUserInfo(mattermostUserID, info)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	html := `
<!DOCTYPE html>
<html>
       <head>
               <script>
                       window.close();
               </script>
       </head>
       <body>
               <p>Disconnected from JIRA. Please close this page.</p>
       </body>
</html>
`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))

	return http.StatusOK, nil
}

func (p *Plugin) handleHTTPUserConfig(w http.ResponseWriter, r *http.Request) (int, error) {
	sc, err := p.LoadSecurityContext()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	_, err = validateJWT(r, sc)
	if err != nil {
		return http.StatusBadRequest, err
	}

	// TODO: Ideally find a way to display a message in the form that includes
	// the MM user ID, not yet sure how to best do it.

	bb := &bytes.Buffer{}
	err = p.userConfigTemplate.ExecuteTemplate(bb, "config",
		struct {
			ArgMMToken            string
			ArgAtlassianAccountID string
			ArgJIRAUserKey        string
			ArgJIRAUserName       string
		}{argMMToken, argAtlassianAccountID, argJIRAUserKey, argJIRAUserName})
	if err != nil {
		return http.StatusInternalServerError, err
	}
	w.Header().Set("Content-Type", "text/html")
	io.Copy(w, bytes.NewReader(bb.Bytes()))
	return http.StatusOK, nil
}

func (p *Plugin) handleHTTPUserConfigSubmit(w http.ResponseWriter, r *http.Request) (int, error) {
	sc, err := p.LoadSecurityContext()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	r.ParseForm()
	mmToken := r.Form.Get(argMMToken)
	uinfo := JIRAUserInfo{
		Key:       r.Form.Get(argJIRAUserKey),
		AccountId: r.Form.Get(argAtlassianAccountID),
		Name:      r.Form.Get(argJIRAUserName),
	}

	mattermostUserID, err := p.ParseAuthToken(mmToken)
	if err != nil {
		return http.StatusBadRequest, err
	}

	err = p.StoreUserInfo(mattermostUserID, uinfo)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	p.API.PublishWebSocketEvent(
		WS_EVENT_CONNECT,
		map[string]interface{}{
			"connected":       true,
			"jira_username":   uinfo.Name,
			"jira_account_id": uinfo.AccountId,
			"jira_url":        sc.BaseURL,
		},
		&model.WebsocketBroadcast{UserId: mattermostUserID},
	)

	// <script src="https://connect-cdn.atl-paas.net/all.js" data-options="base:true" async></script>
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
    <head>
        <link rel="stylesheet" href="https://unpkg.com/@atlaskit/css-reset@2.0.0/dist/bundle.css" media="all">
	<script src="https://connect-cdn.atl-paas.net/all.js" data-options=""></script>
    </head>
    <body>
    granted user ` + mattermostUserID + ` access to JIRA as ` + uinfo.Name + `
    </body>
</html>`))
	return http.StatusOK, nil
}

func (p *Plugin) handleHTTPOAuth2Connect(w http.ResponseWriter, r *http.Request) (int, error) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		return http.StatusUnauthorized, fmt.Errorf("Not authorized")
	}

	// TODO encruypt UserID
	linkURL := p.oauth2Config.AuthCodeURL(
		userID,
		oauth2.SetAuthURLParam("prompt", "consent"),
		oauth2.SetAuthURLParam("audience", "api.atlassian.com"),
	)

	http.Redirect(w, r, linkURL, http.StatusFound)
	return http.StatusFound, nil
}

func (p *Plugin) handleHTTPOAuth2Complete(w http.ResponseWriter, r *http.Request) (int, error) {
	ctx := context.Background()

	err := r.ParseForm()
	if err != nil {
		return http.StatusBadRequest, err
	}
	code := r.Form.Get("code")
	state := r.Form.Get("state")
	// TODO decrypt MM userID
	mattermostUserID := state

	tok, err := p.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	oauthc := p.oauth2Config.Client(ctx, tok)

	jirac, err := jira.NewClient(oauthc, "https://api.atlassian.com/")
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("could not get jira client: %v", err)
	}
	req, _ := jirac.NewRequest("GET", "/oauth/token/accessible-resources", nil)
	resources := []struct {
		Name string `json:"name"`
		Id   string `json:"id"`
	}{}
	_, err = jirac.Do(req, &resources)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("accessible-resources: %v", err)
	}
	if len(resources) != 1 {
		return http.StatusInternalServerError, fmt.Errorf("accessible-resources expoected 1, received %v responses", len(resources))
	}
	// name := resources[0].Name
	cloudId := resources[0].Id

	jirac, err = jira.NewClient(oauthc, "https://api.atlassian.com/ex/jira/"+cloudId)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	info := JIRAUserInfo{}
	req, err = jirac.NewRequest("GET", "rest/api/2/myself", nil)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	_, err = jirac.Do(req, &info)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("could not get user: %v", err)
	}

	sc, err := p.LoadSecurityContext()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	err = p.StoreUserInfo(mattermostUserID, info)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	p.API.PublishWebSocketEvent(
		WS_EVENT_CONNECT,
		map[string]interface{}{
			"connected":       true,
			"jira_username":   info.Name,
			"jira_account_id": info.AccountId,
			"jira_url":        sc.BaseURL,
		},
		&model.WebsocketBroadcast{UserId: mattermostUserID},
	)

	html := `
<!DOCTYPE html>
<html>
       <head>
               <script>
                       window.close();
               </script>
       </head>
       <body>
               <p>Completed connecting to JIRA. Please close this page.</p>
       </body>
</html>
`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
	return http.StatusOK, nil
}

func (p *Plugin) handleHTTPGetUserInfo(w http.ResponseWriter, r *http.Request) (int, error) {
	mattermostUserID := r.Header.Get("Mattermost-User-ID")
	if mattermostUserID == "" {
		return http.StatusUnauthorized, fmt.Errorf("Not authorized")
	}

	sc, err := p.LoadSecurityContext()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	resp := UserInfo{}
	jiraUserInfo, err := p.LoadJIRAUserInfo(mattermostUserID)
	if err == nil {
		resp = UserInfo{
			JIRAUserInfo: jiraUserInfo,
			IsConnected:  true,
			JIRAURL:      sc.BaseURL,
		}
	}

	b, _ := json.Marshal(resp)
	w.Write(b)
	fmt.Println(string(b))
	return http.StatusOK, nil
}
