# Include custom targets and environment variables here

.DEFAULT_GOAL := all

ifndef MM_RUDDER_WRITE_KEY
	MM_RUDDER_WRITE_KEY = 1d5bMvdrfWClLxgK1FvV3s4U1tg
endif
LDFLAGS += -X "github.com/mattermost/mattermost-plugin-jira/server/utils/telemetry.rudderWriteKey=$(MM_RUDDER_WRITE_KEY)"

# Build info
BUILD_DATE = $(shell date -u)
BUILD_HASH = $(shell git rev-parse HEAD)
BUILD_HASH_SHORT = $(shell git rev-parse --short HEAD)
LDFLAGS += -X "main.BuildDate=$(BUILD_DATE)"
LDFLAGS += -X "main.BuildHash=$(BUILD_HASH)"
LDFLAGS += -X "main.BuildHashShort=$(BUILD_HASH_SHORT)"

GO_BUILD_FLAGS = -ldflags '$(LDFLAGS)'

.PHONY: jira
jira:
	docker-compose up
