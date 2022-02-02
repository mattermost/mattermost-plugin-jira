module github.com/mattermost/mattermost-plugin-jira

go 1.13

require (
	github.com/andygrunwald/go-jira v1.10.0
	github.com/circonus-labs/circonusllhist v0.1.3
	github.com/dghubble/oauth1 v0.5.0
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/jarcoal/httpmock v1.0.8
	github.com/mattermost/mattermost-plugin-api v0.0.23-0.20220202173942-88b6cf0af53f
	github.com/mattermost/mattermost-plugin-autolink v1.2.2-0.20210709183311-c8fa30db649f
	github.com/mattermost/mattermost-server/v6 v6.3.0
	github.com/mholt/archiver/v3 v3.5.1
	github.com/pkg/errors v0.9.1
	github.com/rbriski/atlassian-jwt v0.0.0-20180307182949-7bb4ae273058
	github.com/rudderlabs/analytics-go v3.3.1+incompatible
	github.com/stretchr/testify v1.7.0
	github.com/trivago/tgo v1.0.7 // indirect
	golang.org/x/oauth2 v0.0.0-20210805134026-6f1e6394065a
)

// replace github.com/mattermost/mattermost-plugin-api => ../mattermost-plugin-api
