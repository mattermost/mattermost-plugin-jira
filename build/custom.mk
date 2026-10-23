# Include custom targets and environment variables here

# If there's no MM_RUDDER_PLUGINS_PROD, add DEV data
RUDDER_WRITE_KEY = 1d5bMvdrfWClLxgK1FvV3s4U1tg
ifdef MM_RUDDER_PLUGINS_PROD
  RUDDER_WRITE_KEY = $(MM_RUDDER_PLUGINS_PROD)
endif

LDFLAGS += -X "github.com/mattermost/mattermost-plugin-jira/server/telemetry.rudderWriteKey=$(RUDDER_WRITE_KEY)"

.PHONY: jira
jira:
	docker-compose up

E2E_DIR = e2e/playwright
E2E_FAKE_ATLASSIAN_URL ?= http://fake-jira:8090

# Command-line GO_BUILD_LDFLAGS is required: the Makefile assigns it after this file is included.
.PHONY: e2e-dist
e2e-dist: ## Builds dist/ with Jira Cloud OAuth endpoints pointed at the E2E fake. Never ship this bundle.
	$(MAKE) dist GO_BUILD_LDFLAGS='-ldflags="-s -w -X main.atlassianAuthBaseURL=$(E2E_FAKE_ATLASSIAN_URL) -X main.atlassianAPIBaseURL=$(E2E_FAKE_ATLASSIAN_URL)"'

.PHONY: e2e-real
e2e-real: dist $(E2E_DIR)/node_modules ## Runs the real Jira Cloud journeys (needs e2e/playwright/.env and `npm run real:login`).
	cd $(E2E_DIR) && npm run test:real-jira

$(E2E_DIR)/node_modules: $(E2E_DIR)/package-lock.json
	cd $(E2E_DIR) && $(NPM) ci && npx playwright install chromium
	touch $@

.PHONY: e2e-env-up
e2e-env-up: ## Starts the Mattermost stack used by the Playwright suite
	cd $(E2E_DIR) && docker compose up -d --wait

.PHONY: e2e-env-down
e2e-env-down: ## Stops the Playwright Mattermost stack and drops its data
	cd $(E2E_DIR) && docker compose down -v

.PHONY: e2e
e2e: e2e-dist $(E2E_DIR)/node_modules ## Runs the Playwright suite against the stack from e2e-env-up
	cd $(E2E_DIR) && npx playwright test
