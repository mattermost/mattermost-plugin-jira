GO ?= $(shell command -v go 2> /dev/null)
DEP ?= $(shell command -v dep 2> /dev/null)
NPM ?= $(shell command -v npm 2> /dev/null)
HTTP ?= $(shell command -v http 2> /dev/null)
CURL ?= $(shell command -v curl 2> /dev/null)
MANIFEST_FILE ?= plugin.json
PWD := $(shell pwd)

# Verify environment, and define PLUGIN_ID, PLUGIN_VERSION, HAS_SERVER and HAS_WEBAPP as needed.
include build/setup.mk

BUNDLE_NAME ?= $(PLUGIN_ID)-$(PLUGIN_VERSION).tar.gz

# all, the default target, tests, builds and bundles the plugin.
all: check-style test dist

# apply propagates the plugin id into the server/ and webapp/ folders as required.
.PHONY: apply
apply:
	./build/bin/manifest apply

.PHONY: check-style
check-style: server/.depensure webapp/.npminstall gofmt govet
	@echo Checking for style guide compliance

ifneq ($(HAS_WEBAPP),)
	cd webapp && npm run lint
endif

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

.PHONY: govet
govet:
ifneq ($(HAS_SERVER),)
	@echo Running govet
	$(GO) get golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow
	$(GO) vet $$(go list ./server/...)
	$(GO) vet -vettool=$(GOPATH)/bin/shadow $$(go list ./server/...)
	@echo Govet success
endif

# server/.depensure ensures the server dependencies are installed
server/.depensure:
ifneq ($(HAS_SERVER),)
	cd server && $(DEP) ensure
	touch $@
endif

# server builds the server, if it exists, including support for multiple architectures
.PHONY: server
server: server/.depensure
ifneq ($(HAS_SERVER),)
	mkdir -p server/dist;
	cd server && env GOOS=linux GOARCH=amd64 $(GO) build -o dist/plugin-linux-amd64;
	cd server && env GOOS=darwin GOARCH=amd64 $(GO) build -o dist/plugin-darwin-amd64;
	cd server && env GOOS=windows GOARCH=amd64 $(GO) build -o dist/plugin-windows-amd64.exe;
	cd server && cp -r templates dist/templates
endif

# webapp/.npminstall ensures NPM dependencies are installed without having to run this all the time
webapp/.npminstall:
ifneq ($(HAS_WEBAPP),)
	cd webapp && $(NPM) install
	touch $@
endif

# webapp builds the webapp, if it exists
.PHONY: webapp
webapp: webapp/.npminstall
ifneq ($(HAS_WEBAPP),)
	cd webapp && $(NPM) run build;
endif

# bundle generates a tar bundle of the plugin for install
.PHONY: bundle
bundle:
	rm -rf dist/
	mkdir -p dist/$(PLUGIN_ID)
	cp $(MANIFEST_FILE) dist/$(PLUGIN_ID)/
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

# dist builds and bundles the plugin
.PHONY: dist
dist: apply \
      server \
      webapp \
      bundle

# deploy installs the plugin to a (development) server, using the API if appropriate environment
# variables are defined, or copying the files directly to a sibling mattermost-server directory
.PHONY: deploy
deploy: dist
ifneq ($(and $(MM_SERVICESETTINGS_SITEURL),$(MM_ADMIN_USERNAME),$(MM_ADMIN_PASSWORD),$(HTTP)),)
	@echo "Installing plugin via API"
		(TOKEN=`http --print h POST $(MM_SERVICESETTINGS_SITEURL)/api/v4/users/login login_id=$(MM_ADMIN_USERNAME) password=$(MM_ADMIN_PASSWORD) X-Requested-With:"XMLHttpRequest" | grep Token | cut -f2 -d' '` && \
		  http --print b GET $(MM_SERVICESETTINGS_SITEURL)/api/v4/users/me Authorization:"Bearer $$TOKEN" && \
			http --print b DELETE $(MM_SERVICESETTINGS_SITEURL)/api/v4/plugins/$(PLUGIN_ID) Authorization:"Bearer $$TOKEN" && \
			http --print b --check-status --form POST $(MM_SERVICESETTINGS_SITEURL)/api/v4/plugins plugin@dist/$(BUNDLE_NAME) Authorization:"Bearer $$TOKEN" && \
		  http --print b POST $(MM_SERVICESETTINGS_SITEURL)/api/v4/plugins/$(PLUGIN_ID)/enable Authorization:"Bearer $$TOKEN" && \
		  http --print b POST $(MM_SERVICESETTINGS_SITEURL)/api/v4/users/logout Authorization:"Bearer $$TOKEN" \
	  )
else ifneq ($(and $(MM_SERVICESETTINGS_SITEURL),$(MM_ADMIN_USERNAME),$(MM_ADMIN_PASSWORD),$(CURL)),)
	@echo "Installing plugin via API"
	$(eval TOKEN := $(shell curl -i -X POST $(MM_SERVICESETTINGS_SITEURL)/api/v4/users/login -d '{"login_id": "$(MM_ADMIN_USERNAME)", "password": "$(MM_ADMIN_PASSWORD)"}' | grep Token | cut -f2 -d' ' 2> /dev/null))
	@curl -s -H "Authorization: Bearer $(TOKEN)" -X DELETE $(MM_SERVICESETTINGS_SITEURL)/api/v4/plugins/$(PLUGIN_ID) > /dev/null
	@curl -s -H "Authorization: Bearer $(TOKEN)" -X POST $(MM_SERVICESETTINGS_SITEURL)/api/v4/plugins -F "plugin=@dist/$(BUNDLE_NAME)" > /dev/null && \
		curl -s -H "Authorization: Bearer $(TOKEN)" -X POST $(MM_SERVICESETTINGS_SITEURL)/api/v4/plugins/$(PLUGIN_ID)/enable > /dev/null && \
		echo "OK." || echo "Sorry, something went wrong."
else ifneq ($(wildcard ../mattermost-server/.*),)
	@echo "Installing plugin via filesystem. Server restart and manual plugin enabling required"
	mkdir -p ../mattermost-server/plugins
	tar -C ../mattermost-server/plugins -zxvf dist/$(BUNDLE_NAME)
else
	@echo "No supported deployment method available. Install plugin manually."
endif

# test runs any lints and unit tests defined for the server and webapp, if they exist
.PHONY: test
test: server/.depensure webapp/.npminstall
ifneq ($(HAS_SERVER),)
	cd server && $(GO) test -race -v -coverprofile=coverage.txt ./...
endif
ifneq ($(HAS_WEBAPP),)
	cd webapp && $(NPM) run fix;
endif

# clean removes all build artifacts
.PHONY: clean
clean:
	rm -fr dist/
ifneq ($(HAS_SERVER),)
	rm -fr server/dist
	rm -fr server/.depensure
endif
ifneq ($(HAS_WEBAPP),)
	rm -fr webapp/.npminstall
	rm -fr webapp/dist
	rm -fr webapp/node_modules
endif
	rm -fr build/bin/

# debug builds and deploys a debug version of the webapp, for only my architecture.
.PHONY: debug
debug:
	./build/bin/manifest apply
ifneq ($(HAS_SERVER),)
	mkdir -p server/dist
	cd server && env GOOS=darwin GOARCH=amd64 $(GO) build -gcflags "all=-N -l" -o dist/plugin-darwin-amd64

	# uncomment for your own architecture. This speeds up the cycle.
	#cd server && env GOOS=linux GOARCH=amd64 $(GO) build -gcflags "all=-N -l" -o dist/plugin-linux-amd64;
	#cd server && env GOOS=windows GOARCH=amd64 $(GO) build -gcflags "all=-N -l" -o dist/plugin-windows-amd64.exe;
	cd server && cp -r templates dist/templates
endif
	rm -rf dist/
	mkdir -p dist/$(PLUGIN_ID)
	cp $(MANIFEST_FILE) dist/$(PLUGIN_ID)/
ifneq ($(HAS_SERVER),)
	mkdir -p dist/$(PLUGIN_ID)/server/dist
	cp -r server/dist/* dist/$(PLUGIN_ID)/server/dist/
endif

	mkdir -p ../mattermost-server/plugins
	cp -r dist/* ../mattermost-server/plugins/

	# if there is a current plugin running, kill it
	$(eval PLUGIN_PID := $(shell ps aux | grep "$(PLUGIN_ID)" | grep -v "grep" | awk -F " " '{print $$2}'))
ifneq ($(PLUGIN_PID),)
	@echo "\n\n*** Located existing Plugin running with PID: $(PLUGIN_PID). Killing. \n*** Restart plugin from system conssole and run debug-plugin.sh\n\n"
	kill -9 $(PLUGIN_PID)
endif

	# link the webapp directory for many benefits
ifneq ($(HAS_WEBAPP),)
	mkdir -p ../mattermost-server/plugins/$(PLUGIN_ID)/webapp
	mkdir -p ./webapp
	ln -nfs $(PWD)/webapp/dist ../mattermost-server/plugins/$(PLUGIN_ID)/webapp/dist
	# start an npm watch
	cd webapp && $(NPM) run run &
endif

	@echo "\n\n*** After the frontend is compiled, run 'make reset' to reset the plugin. Run reset to make the server pickup chages in your plugin.\n\n"

# Reset the plugin
reset:
ifneq ($(and $(MM_SERVICESETTINGS_SITEURL),$(MM_ADMIN_USERNAME),$(MM_ADMIN_PASSWORD),$(CURL)),)
	@echo "\n\nRestarting plugin via API"
	$(eval TOKEN := $(shell curl -i -X POST $(MM_SERVICESETTINGS_SITEURL)/api/v4/users/login -d '{"login_id": "$(MM_ADMIN_USERNAME)", "password": "$(MM_ADMIN_PASSWORD)"}' | grep Token | cut -f2 -d' '))
	@curl -s -H "Authorization: Bearer $(TOKEN)" -X POST $(MM_SERVICESETTINGS_SITEURL)/api/v4/plugins/$(PLUGIN_ID)/disable && \
		curl -s -H "Authorization: Bearer $(TOKEN)" -X POST $(MM_SERVICESETTINGS_SITEURL)/api/v4/plugins/$(PLUGIN_ID)/enable > /dev/null
endif

stop: ## Stops webpack
	@echo Stopping changes watching

ifeq ($(OS),Windows_NT)
	wmic process where "Caption='node.exe' and CommandLine like '%webpack%'" call terminate
else
	@for PROCID in $$(ps -ef | grep "[n]ode.*[w]ebpack" | awk '{ print $$2 }'); do \
		echo stopping webpack watch $$PROCID; \
		kill $$PROCID; \
	done
endif

