module github.com/mattermost/mattermost-plugin-jira

go 1.12

require (
	github.com/andygrunwald/go-jira v1.10.0
	github.com/circonus-labs/circonusllhist v0.1.3
	github.com/dghubble/oauth1 v0.5.0
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/fatih/structs v1.1.0 // indirect
	github.com/google/uuid v1.1.1
	github.com/mattermost/mattermost-plugin-workflow v0.0.0-00010101000000-000000000000
	github.com/mattermost/mattermost-server/v5 v5.18.0
	github.com/pkg/errors v0.8.1
	github.com/rbriski/atlassian-jwt v0.0.0-20180307182949-7bb4ae273058
	github.com/stretchr/testify v1.4.0
	github.com/trivago/tgo v1.0.7 // indirect
	golang.org/x/oauth2 v0.0.0-20191202225959-858c2ad4c8b6
)

replace willnorris.com/go/imageproxy => willnorris.com/go/imageproxy v0.8.1-0.20190422234945-d4246a08fdec

replace github.com/mattermost/mattermost-plugin-workflow => ../mattermost-plugin-workflow
