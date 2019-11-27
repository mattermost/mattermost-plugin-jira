module github.com/mattermost/mattermost-plugin-jira

go 1.12

require (
	github.com/andygrunwald/go-jira v1.10.0
	github.com/circonus-labs/circonusllhist v0.1.3
	github.com/dghubble/oauth1 v0.5.0
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/fatih/structs v1.1.0 // indirect
	github.com/go-ldap/ldap v3.0.3+incompatible // indirect
	github.com/google/uuid v1.1.1
	github.com/hashicorp/go-hclog v0.9.2 // indirect
	github.com/hashicorp/go-plugin v1.0.1 // indirect
	github.com/lib/pq v1.1.1 // indirect
	github.com/mattermost/go-i18n v1.11.0 // indirect
	github.com/mattermost/mattermost-server v1.4.1-0.20191019085930-e13dad9fd4b6
	github.com/pelletier/go-toml v1.4.0 // indirect
	github.com/pkg/errors v0.8.1
	github.com/rbriski/atlassian-jwt v0.0.0-20180307182949-7bb4ae273058
	github.com/stretchr/objx v0.2.0 // indirect
	github.com/stretchr/testify v1.3.0
	github.com/trivago/tgo v1.0.7 // indirect
	go.uber.org/atomic v1.4.0 // indirect
	go.uber.org/zap v1.10.0 // indirect
	golang.org/x/crypto v0.0.0-20190621222207-cc06ce4a13d4 // indirect
	golang.org/x/net v0.0.0-20190620200207-3b0461eec859 // indirect
	golang.org/x/oauth2 v0.0.0-20190319182350-c85d3e98c914
	golang.org/x/sys v0.0.0-20190626221950-04f50cda93cb // indirect
	google.golang.org/appengine v1.6.1 // indirect
	google.golang.org/genproto v0.0.0-20190626174449-989357319d63 // indirect
	google.golang.org/grpc v1.21.1 // indirect
)

// Workaround for Jira GDPR change. Remove this and update golang.org/x/oauth2 when it's fixed.
// See: https://github.com/golang/oauth2/issues/312
replace golang.org/x/oauth2 => github.com/cpoile/oauth2 v0.0.0-20190717234427-10945bd219c7

replace willnorris.com/go/imageproxy => willnorris.com/go/imageproxy v0.8.1-0.20190422234945-d4246a08fdec
