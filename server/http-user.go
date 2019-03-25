// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	jira "github.com/andygrunwald/go-jira"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost-server/model"
)

const (
	WS_EVENT_CONNECT = "connect"
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
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		return http.StatusUnauthorized, fmt.Errorf("Not authorized")
	}

	sc, err := p.LoadSecurityContext()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	// TODO: Add an encrypted token that contains MM user ID so that UserConfig
	redirectURL := fmt.Sprintf("%v/login?dest-url=%v/plugins/servlet/ac/mattermost-plugin/config?qqqqq=wwwww", sc.BaseURL, sc.BaseURL)
	http.Redirect(w, r, redirectURL, http.StatusFound)
	return http.StatusFound, nil
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

	// <script src="https://connect-cdn.atl-paas.net/all.js" data-options="base:true" async></script>

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
    <head>
        <link rel="stylesheet" href="https://unpkg.com/@atlaskit/css-reset@2.0.0/dist/bundle.css" media="all">
	<script src="https://connect-cdn.atl-paas.net/all.js"></script>

	<script>
		function getParameterByName(name, url) {
		    	var regex = new RegExp('[?&]' + name + '(=([^&#]*)|&|#|$)'),
				results = regex.exec(url);
		    	if (!results) return null;
		    	if (!results[2]) return '';
		    	return decodeURIComponent(results[2].replace(/\+/g, ' '));
		}

		AP.getLocation(function (location) {
			document.getElementById("mm-token").value = getParameterByName("mm-token", location);
		});
	</script>
    </head>
    <body>
      <form action="/plugins/jira/config-2">
        <input type="hidden" id="mm-token" name="mm-token" value="none"/>
        <input type="submit" value="Submit"/>
      </form>
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
