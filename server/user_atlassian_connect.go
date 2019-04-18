// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	jira "github.com/andygrunwald/go-jira"
	"github.com/dgrijalva/jwt-go"

	"github.com/mattermost/mattermost-server/model"
)

const (
	argMMToken = "mm_token"
)

func (p *Plugin) handleHTTPUserConnect(w http.ResponseWriter, r *http.Request) (int, error) {
	// TODO Enforce a GET
	mattermostUserId := r.Header.Get("Mattermost-User-Id")
	if mattermostUserId == "" {
		return http.StatusUnauthorized, fmt.Errorf("Not authorized")
	}

	ji, err := p.LoadCurrentJIRAInstance()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	redirectURL, err := ji.GetUserConnectURL(p, mattermostUserId)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	http.Redirect(w, r, redirectURL, http.StatusFound)
	return http.StatusFound, nil
}

func (p *Plugin) handleHTTPUserDisconnect(w http.ResponseWriter, r *http.Request) (int, error) {
	// TODO Enforce a GET
	mattermostUserId := r.Header.Get("Mattermost-User-Id")
	if mattermostUserId == "" {
		return http.StatusUnauthorized, fmt.Errorf("Not authorized")
	}

	ji, err := p.LoadCurrentJIRAInstance()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	err = p.DeleteUserInfo(ji, mattermostUserId)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	p.API.PublishWebSocketEvent(
		WS_EVENT_DISCONNECT,
		map[string]interface{}{
			"is_connected": false,
		},
		&model.WebsocketBroadcast{UserId: mattermostUserId},
	)

	html := `
<!DOCTYPE html>
<html>
       <head>
               <script>
                       // window.close();
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
	ji, err := p.LoadCurrentJIRAInstance()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	jci, ok := ji.(*jiraCloudInstance)
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("Must be a JIRA Cloud instance, is %s", ji.GetType())
	}

	_, tokenString, err := jci.parseHTTPRequestJWT(r)
	if err != nil {
		return http.StatusBadRequest, err
	}

	// TODO: Ideally find a way to display a message in the form that includes
	// the MM user ID, not yet sure how to best do it.

	bb := &bytes.Buffer{}
	err = p.userConfigTemplate.ExecuteTemplate(bb, "config",
		struct {
			JWT        string
			ArgMMToken string
		}{
			JWT:        tokenString,
			ArgMMToken: argMMToken,
		})
	if err != nil {
		return http.StatusInternalServerError, err
	}
	w.Header().Set("Content-Type", "text/html")
	io.Copy(w, bytes.NewReader(bb.Bytes()))
	return http.StatusOK, nil
}

func (p *Plugin) handleHTTPUserConfigSubmit(w http.ResponseWriter, r *http.Request) (int, error) {
	ji, err := p.LoadCurrentJIRAInstance()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	jci, ok := ji.(*jiraCloudInstance)
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("Must be a JIRA Cloud instance, is %s", ji.GetType())
	}

	jwtToken, _, err := jci.parseHTTPRequestJWT(r)
	if err != nil {
		return http.StatusBadRequest, err
	}
	claims, ok := jwtToken.Claims.(jwt.MapClaims)
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("invalid JWT claims")
	}
	context, ok := claims["context"].(map[string]interface{})
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("invalid JWT claim context")
	}
	user, ok := context["user"].(map[string]interface{})
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("invalid JWT: no user data")
	}

	userKey, _ := user["userKey"].(string)
	username, _ := user["username"].(string)
	displayName, _ := user["displayName"].(string)

	mmToken := r.Form.Get(argMMToken)
	uinfo := JIRAUser{
		User: jira.User{
			Key:  userKey,
			Name: username,
		},
	}

	mattermostUserId, err := p.ParseAuthToken(mmToken)
	if err != nil {
		return http.StatusBadRequest, err
	}
	err = p.StoreAndNotifyUserInfo(ji, mattermostUserId, uinfo)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	mmuser, aerr := p.API.GetUser(mattermostUserId)
	if aerr != nil {
		return http.StatusInternalServerError, aerr
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
    <head>
        <link rel="stylesheet" href="https://unpkg.com/@atlaskit/css-reset@2.0.0/dist/bundle.css" media="all">
	<script src="https://connect-cdn.atl-paas.net/all.js" data-options=""></script>
    </head>
    <body>
    granted Mattermost user ` + mmuser.GetDisplayName(model.SHOW_NICKNAME_FULLNAME) + " (" + mmuser.Username + `) access to JIRA as ` + displayName + " (" + username + `)
    </body>
</html>`))
	return http.StatusOK, nil
}
