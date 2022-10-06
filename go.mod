module github.com/mattermost/mattermost-plugin-jira

go 1.13

require (
	github.com/andygrunwald/go-jira v1.10.0
	github.com/dghubble/oauth1 v0.5.0
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/gorilla/mux v1.8.0
	github.com/jarcoal/httpmock v1.0.8
	github.com/mattermost/mattermost-plugin-api v0.0.26-0.20220223141232-cb8b1984774a
	github.com/mattermost/mattermost-plugin-autolink v1.2.2-0.20210709183311-c8fa30db649f
	github.com/mattermost/mattermost-server/v6 v6.3.0
	github.com/mholt/archiver/v3 v3.5.1
	github.com/pkg/errors v0.9.1
	github.com/rbriski/atlassian-jwt v0.0.0-20180307182949-7bb4ae273058
	github.com/stretchr/testify v1.7.0
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8
)

// Until github.com/mattermost/mattermost-server/v6 v6.5.0 is releated,
// this replacement is needed to also import github.com/mattermost/mattermost-plugin-api,
// which uses a different server version.
replace github.com/mattermost/mattermost-server/v6 v6.3.0 => github.com/mattermost/mattermost-server/v6 v6.0.0-20220210052000-0d67995eb491
