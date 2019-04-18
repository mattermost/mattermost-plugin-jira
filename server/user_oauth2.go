// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"context"
	"fmt"
	"net/http"

	jira "github.com/andygrunwald/go-jira"
	"golang.org/x/oauth2"
)

func (p *Plugin) handleHTTPOAuth2Connect(w http.ResponseWriter, r *http.Request) (int, error) {
	userId := r.Header.Get("Mattermost-User-Id")
	if userId == "" {
		return http.StatusUnauthorized, fmt.Errorf("Not authorized")
	}

	ji, err := p.LoadCurrentJIRAInstance()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	jic, ok := ji.(*jiraCloudInstance)
	if !ok {
		return http.StatusInternalServerError, fmt.Errorf("Must be a JIRA Cloud instance")
	}

	// TODO encruypt UserID
	linkURL := jic.oauth2Config.AuthCodeURL(
		userId,
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
	mattermostUserId := state

	ji, err := p.LoadCurrentJIRAInstance()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	jic, ok := ji.(*jiraCloudInstance)
	if !ok {
		return http.StatusInternalServerError, fmt.Errorf("Must be a JIRA Cloud instance")
	}

	tok, err := jic.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	oauthc := jic.oauth2Config.Client(ctx, tok)

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

	jiraUser := JIRAUser{}
	req, err = jirac.NewRequest("GET", "rest/api/2/myself", nil)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	_, err = jirac.Do(req, &jiraUser)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("could not get user: %v", err)
	}

	err = p.StoreAndNotifyUserInfo(ji, mattermostUserId, jiraUser)
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
               <p>Completed connecting to JIRA. Please close this page.</p>
       </body>
</html>
`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
	return http.StatusOK, nil
}
