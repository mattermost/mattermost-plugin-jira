NPM ?= $(shell command -v npm 2> /dev/null)
CURL ?= $(shell command -v curl 2> /dev/null)
MANIFEST_FILE ?= plugin.json
MM_UTILITIES_DIR ?= ../mattermost-utilities

MINIMUM_SUPPORTED_GO_MAJOR_VERSION = 1
MINIMUM_SUPPORTED_GO_MINOR_VERSION = 12
export GO111MODULE=on
GO ?= $(shell command -v go 2> /dev/null)
GOFLAGS ?= $(GOFLAGS:)
BUILD_DATE = $(shell date -u)
BUILD_HASH = $(shell git rev-parse HEAD)
BUILD_HASH_SHORT = $(shell git rev-parse --short HEAD)
LDFLAGS += -X "main.BuildDate=$(BUILD_DATE)"
LDFLAGS += -X "main.BuildHash=$(BUILD_HASH)"
LDFLAGS += -X "main.BuildHashShort=$(BUILD_HASH_SHORT)"
GO_MAJOR_VERSION = $(shell $(GO) version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f1)
GO_MINOR_VERSION = $(shell $(GO) version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f2)
GO_VERSION_VALIDATION_ERR_MSG = Golang version is not supported, please update to at least $(MINIMUM_SUPPORTED_GO_MAJOR_VERSION).$(MINIMUM_SUPPORTED_GO_MINOR_VERSION)

GOVET = $(GO) vet $(GOFLAGS)
GOTOOL = $(GO) tool $(GOFLAGS)
GOBUILD = $(GO) build $(GOFLAGS) -ldflags '$(LDFLAGS)'
GOTEST = $(GO) test $(GOFLAGS) -ldflags '$(LDFLAGS)'

# You can include assets this directory into the bundle. This can be e.g. used to include profile pictures.
ASSETS_DIR ?= assets
PWD := $(shell pwd)
OS := $(shell uname 2> /dev/null)

# Verify environment, and define PLUGIN_ID, PLUGIN_VERSION, HAS_SERVER and HAS_WEBAPP as needed.
include build/setup.mk

BUNDLE_NAME ?= $(PLUGIN_ID)-$(PLUGIN_VERSION).tar.gz

## Checks the code style, tests, builds and bundles the plugin.
all: check-style test dist

## Propagates plugin manifest information into the server/ and webapp/ folders as required.
.PHONY: apply
apply:
	./build/bin/manifest apply

## Runs govet and gofmt against all packages.
.PHONY: check-style
check-style: .npminstall gofmt govet
	@echo Checking for style guide compliance

ifneq ($(HAS_WEBAPP),)
	cd webapp && npm run lint
endif

## Runs gofmt against all packages.
.PHONY: gofmt
gofmt:
ifneq ($(HAS_SERVER),)
	@echo Running gofmt
	@for package in $$(go list ./server/...); do \
		echo "Checking "$$package; \
		files=$$(go list -f '{{range .GoFiles}}{{$$.Dir}}/{{.}} {{end}}' $$package); \
		if [ "$$files" ]; then \
			gofmt_output=$$(gofmt -d -s $$files 2>&1); \
			if [ "$$gofmt_output" ]; then \
				echo "$$gofmt_output"; \
				echo "Gofmt failure"; \
				exit 1; \
			fi; \
		fi; \
	done
	@echo Gofmt success
endif

## Runs govet against all packages.
.PHONY: govet
govet: validate-go-version
ifneq ($(HAS_SERVER),)
	@echo Running govet
	@# Workaroung because you can't install binaries without adding them to go.mod
	env GO111MODULE=off $(GO) get golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow
	$(GOVET) ./server/...
	$(GO) vet -vettool=$(GOPATH)/bin/shadow $(GOFLAGS) ./server/...
	@echo Govet success
endif

## Builds the server, if it exists, including support for multiple architectures.
.PHONY: server
server: validate-go-version
ifneq ($(HAS_SERVER),)
	rm -rf server/dist;
	mkdir -p server/dist;
	cd server && env GOOS=linux GOARCH=amd64 $(GOBUILD) -o dist/plugin-linux-amd64;
	cd server && env GOOS=darwin GOARCH=amd64 $(GOBUILD) -o dist/plugin-darwin-amd64;
	cd server && env GOOS=windows GOARCH=amd64 $(GOBUILD) -o dist/plugin-windows-amd64.exe;
endif

## Ensures NPM dependencies are installed without having to run this all the time.
.npminstall:
ifneq ($(HAS_WEBAPP),)
	cd webapp && $(NPM) install
	touch $@
endif

## Builds the webapp, if it exists.
.PHONY: webapp
webapp: .npminstall
ifneq ($(HAS_WEBAPP),)
	cd webapp && $(NPM) run build;
endif

## Generates a tar bundle of the plugin for install.
.PHONY: bundle
bundle:
	rm -rf dist/
	mkdir -p dist/$(PLUGIN_ID)
	cp $(MANIFEST_FILE) dist/$(PLUGIN_ID)/
ifneq ($(wildcard $(ASSETS_DIR)/.),)
	cp -r $(ASSETS_DIR) dist/$(PLUGIN_ID)/
endif
ifneq ($(HAS_PUBLIC),)
	cp -r public/ dist/$(PLUGIN_ID)/
endif
ifneq ($(HAS_SERVER),)
	mkdir -p dist/$(PLUGIN_ID)/server/dist;
	cp -r server/dist/* dist/$(PLUGIN_ID)/server/dist/;
endif
ifneq ($(HAS_WEBAPP),)
	mkdir -p dist/$(PLUGIN_ID)/webapp/dist;
	cp -r webapp/dist/* dist/$(PLUGIN_ID)/webapp/dist/;
endif
	cd dist && tar -cvzf $(BUNDLE_NAME) $(PLUGIN_ID)

	@echo plugin built at: dist/$(BUNDLE_NAME)

## Builds and bundles the plugin.
.PHONY: dist
dist:	apply server webapp bundle

## Installs the plugin to a (development) server.
.PHONY: deploy
deploy: dist
## It uses the API if appropriate environment variables are defined,
## or copying the files directly to a sibling mattermost-server directory.
ifneq ($(and $(MM_SERVICESETTINGS_SITEURL),$(MM_ADMIN_USERNAME),$(MM_ADMIN_PASSWORD),$(CURL)),)
	@echo "Installing plugin via API"
	$(eval TOKEN := $(shell curl -i -X POST $(MM_SERVICESETTINGS_SITEURL)/api/v4/users/login -d '{"login_id": "$(MM_ADMIN_USERNAME)", "password": "$(MM_ADMIN_PASSWORD)"}' | grep -io "Token: [0-9a-z]*" | cut -f2 -d' ' 2> /dev/null))
	@curl -s -H "Authorization: Bearer $(TOKEN)" -X POST $(MM_SERVICESETTINGS_SITEURL)/api/v4/plugins -F "plugin=@dist/$(BUNDLE_NAME)" -F "force=true" > /dev/null && \
		curl -s -H "Authorization: Bearer $(TOKEN)" -X POST $(MM_SERVICESETTINGS_SITEURL)/api/v4/plugins/$(PLUGIN_ID)/enable > /dev/null && \
		echo "OK." || echo "Sorry, something went wrong."
else ifneq ($(wildcard ../mattermost-server/.*),)
	@echo "Installing plugin via filesystem. Server restart and manual plugin enabling required"
	mkdir -p ../mattermost-server/plugins
	tar -C ../mattermost-server/plugins -zxvf dist/$(BUNDLE_NAME)
else
	@echo "No supported deployment method available. Install plugin manually."
endif

## Runs any lints and unit tests defined for the server and webapp, if they exist.
.PHONY: test
test: .npminstall validate-go-version
ifneq ($(HAS_SERVER),)
	$(GOTEST) -race -v ./server/...
endif
ifneq ($(HAS_WEBAPP),)
	cd webapp && $(NPM) run fix && $(NPM) run test;
endif

## Creates a coverage report for the server code.
.PHONY: coverage
coverage: .npminstall validate-go-version
ifneq ($(HAS_SERVER),)
	$(GOTEST) -race -coverprofile=server/coverage.txt ./server/...
	$(GOTOOL) cover -html=server/coverage.txt
endif
ifneq ($(HAS_WEBAPP),)
	cd webapp && $(NPM) run fix && $(NPM) run test-ci;
endif

## Extract strings for translation from the source code.
.PHONY: i18n-extract
i18n-extract:
ifneq ($(HAS_WEBAPP),)
	@[[ -d $(MM_UTILITIES_DIR) ]] || echo "You must clone github.com/mattermost/mattermost-utilities repo in .. to use this command"
	@[[ -d $(MM_UTILITIES_DIR) ]] && cd $(MM_UTILITIES_DIR) && npm install && npm run babel && node mmjstool/build/index.js i18n extract-webapp --webapp-dir ../mattermost-plugin-demo/webapp
endif

## Clean removes all build artifacts.
.PHONY: clean
clean:
	rm -fr dist/
ifneq ($(HAS_SERVER),)
	rm -fr server/dist
endif
ifneq ($(HAS_WEBAPP),)
	rm -fr .npminstall
	rm -fr webapp/dist
	rm -fr webapp/node_modules
endif
	rm -fr build/bin/

# Help documentatin à la https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help:
	@cat Makefile | grep -v '\.PHONY' |  grep -v '\help:' | grep -B1 -E '^[a-zA-Z0-9_.-]+:.*' | sed -e "s/:.*//" | sed -e "s/^## //" |  grep -v '\-\-' | sed '1!G;h;$$!d' | awk 'NR%2{printf "\033[36m%-30s\033[0m",$$0;next;}1' | sort

# sd is an easier-to-type alias for server-debug
.PHONY: sd
sd: server-debug

# server-debug builds and deploys a debug version of the plugin for your architecture.
# Then resets the plugin to pick up the changes.
.PHONY: server-debug
server-debug: server-debug-deploy reset

.PHONY: server-debug-deploy
server-debug-deploy: validate-go-version
	./build/bin/manifest apply
	mkdir -p server/dist

ifeq ($(OS),Darwin)
	cd server && env GOOS=darwin GOARCH=amd64 $(GOBUILD) -gcflags "all=-N -l" -o dist/plugin-darwin-amd64;
else ifeq ($(OS),Linux)
	cd server && env GOOS=linux GOARCH=amd64 $(GOBUILD) -gcflags "all=-N -l" -o dist/plugin-linux-amd64;
else ifeq ($(OS),Windows_NT)
	cd server && env GOOS=windows GOARCH=amd64 $(GOBUILD) -gcflags "all=-N -l" -o dist/plugin-windows-amd64.exe;
else
	$(error make debug depends on uname to return your OS. If it does not return 'Darwin' (meaning OSX), 'Linux', or 'Windows_NT' (all recent versions of Windows), you will need to edit the Makefile for your own OS.)
endif

	rm -rf dist/
	mkdir -p dist/$(PLUGIN_ID)/server/dist
	cp $(MANIFEST_FILE) dist/$(PLUGIN_ID)/
	cp -r server/dist/* dist/$(PLUGIN_ID)/server/dist/
	mkdir -p ../mattermost-server/plugins
	cp -r dist/* ../mattermost-server/plugins/

# wd is an easier-to-type alias for webapp-debug
.PHONY: wd
wd: webapp-debug

# webapp-debug builds and deploys the plugin's webapp in watch mode with source-maps.
# Webpack will run make-reset after detecting and compiling changes.
.PHONY: webapp-debug
webapp-debug:
ifneq ($(HAS_WEBAPP),)
# link the webapp directory
	rm -rf ../mattermost-server/plugins/$(PLUGIN_ID)/webapp
	mkdir -p ../mattermost-server/plugins/$(PLUGIN_ID)/webapp
	ln -nfs $(PWD)/webapp/dist ../mattermost-server/plugins/$(PLUGIN_ID)/webapp/dist
# start an npm watch
	cd webapp && $(NPM) run run
endif

# Reset the plugin
.PHONY: reset
reset:
ifeq ($(and $(MM_SERVICESETTINGS_SITEURL),$(MM_ADMIN_USERNAME),$(MM_ADMIN_PASSWORD),$(CURL)),)
	$(error In order to use make reset, the following environment variables need to be defined: MM_SERVICESETTINGS_SITEURL, MM_ADMIN_USERNAME, MM_ADMIN_PASSWORD, and you need to have curl installed.)
endif

# If we were debugging, we have to unattach the delve process or else we can't disable the plugin.
# NOTE: we are assuming the dlv was listening on port 2346, as in the debug-plugin.sh script.
	@DELVE_PID=$(shell ps aux | grep "dlv attach.*2346" | grep -v "grep" | awk -F " " '{print $$2}') && \
	if [ "$$DELVE_PID" -gt 0 ] > /dev/null 2>&1 ; then \
		echo "Located existing delve process running with PID: $$DELVE_PID. Killing." ; \
		kill -9 $$DELVE_PID ; \
	fi

	@echo "\nRestarting plugin via API"
	$(eval TOKEN := $(shell curl -i -X POST $(MM_SERVICESETTINGS_SITEURL)/api/v4/users/login -d '{"login_id": "$(MM_ADMIN_USERNAME)", "password": "$(MM_ADMIN_PASSWORD)"}' | grep -o "Token: [0-9a-z]*" | cut -f2 -d' ' 2> /dev/null))
	@curl -s -H "Authorization: Bearer $(TOKEN)" -X POST $(MM_SERVICESETTINGS_SITEURL)/api/v4/plugins/$(PLUGIN_ID)/disable > /dev/null && \
		curl -s -H "Authorization: Bearer $(TOKEN)" -X POST $(MM_SERVICESETTINGS_SITEURL)/api/v4/plugins/$(PLUGIN_ID)/enable > /dev/null && \
		echo "OK." || echo "Sorry, something went wrong. Check that MM_ADMIN_USERNAME and MM_ADMIN_PASSWORD env variables are set correctly."


.PHONY: debug-plugin
debug-plugin:
	$(eval PLUGIN_PID := $(shell ps aux | grep "plugins/${PLUGIN_ID}" | grep -v "grep" | awk -F " " '{print $$2}'))
	$(eval NUM_PID := $(shell echo -n ${PLUGIN_PID} | wc -w))

	@if [ ${NUM_PID} -gt 2 ]; then \
		echo "** There is more than 1 plugin process running. Run 'make kill-plugin' to get rid of them."; \
		echo "   Then run 'make reset' to start the plugin process again, and 'make debug-plugin' attach the dlv process."; \
		exit 1; \
	fi

	@if [ -z ${PLUGIN_PID} ]; then \
		echo "Could not find plugin PID; the plugin is not running. Exiting."; \
		exit 1; \
	fi

	@echo "Located Plugin running with PID: ${PLUGIN_PID}"
	dlv attach ${PLUGIN_PID} --listen :2346 --headless=true --api-version=2 --accept-multiclient &

.PHONY: kill-plugin
kill-plugin:
# If we were debugging, we have to unattach the delve process or else we can't disable the plugin.
# NOTE: we are assuming the dlv was listening on port 2346, as in the debug-plugin.sh script.
	$(eval DELVE_PID := $(shell ps aux | grep "dlv attach.*2346" | grep -v "grep" | awk -F " " '{print $$2}'))

	@if [ -n "${DELVE_PID}" ]; then \
		echo "Located existing delve process running with PID: ${DELVE_PID}. Killing."; \
		kill -9 ${DELVE_PID}; \
	fi

	$(eval PLUGIN_PID := $(shell ps aux | grep "plugins/${PLUGIN_ID}" | grep -v "grep" | awk -F " " '{print $$2}'))

	@for PID in ${PLUGIN_PID}; do \
		echo "Killing plugin pid $$PID"; \
		kill -9 $$PID; \
	done; \

.PHONY: validate-go-version
validate-go-version: ## Validates the installed version of go against Mattermost's minimum requirement.
	@if [ $(GO_MAJOR_VERSION) -gt $(MINIMUM_SUPPORTED_GO_MAJOR_VERSION) ]; then \
		exit 0 ;\
	elif [ $(GO_MAJOR_VERSION) -lt $(MINIMUM_SUPPORTED_GO_MAJOR_VERSION) ]; then \
		echo '$(GO_VERSION_VALIDATION_ERR_MSG)';\
		exit 1; \
	elif [ $(GO_MINOR_VERSION) -lt $(MINIMUM_SUPPORTED_GO_MINOR_VERSION) ] ; then \
		echo '$(GO_VERSION_VALIDATION_ERR_MSG)';\
		exit 1; \
	fi
