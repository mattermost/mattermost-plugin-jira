# Include custom targets and environment variables here
ifndef MM_RUDDER_WRITE_KEY
	MM_RUDDER_WRITE_KEY = 1d5bMvdrfWClLxgK1FvV3s4U1tg
endif
LDFLAGS += -X "github.com/mattermost/mattermost-plugin-jira/server/utils/telemetry.rudderWriteKey=$(MM_RUDDER_WRITE_KEY)"
