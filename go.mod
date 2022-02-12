module github.com/mattermost/mattermost-plugin-jira

go 1.13

require (
	github.com/andygrunwald/go-jira v1.10.0
	github.com/dghubble/oauth1 v0.5.0
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/gorilla/mux v1.8.0
	github.com/jarcoal/httpmock v1.0.8
	github.com/mattermost/mattermost-plugin-api v0.0.25-0.20220211191519-03e63a42dc92
	github.com/mattermost/mattermost-plugin-autolink v1.2.2-0.20210709183311-c8fa30db649f
	github.com/mattermost/mattermost-server/v6 v6.3.0
	github.com/mholt/archiver/v3 v3.5.1
	github.com/pkg/errors v0.9.1
	github.com/rbriski/atlassian-jwt v0.0.0-20180307182949-7bb4ae273058
	github.com/stretchr/testify v1.7.0
	golang.org/x/oauth2 v0.0.0-20210805134026-6f1e6394065a
)

// replace github.com/mattermost/mattermost-plugin-api => ../mattermost-plugin-api
