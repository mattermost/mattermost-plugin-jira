#!/usr/bin/env bash

set -euf -o pipefail

# If we were debugging, we have to unattach the delve process or else we can't disable the plugin.
# NOTE: we are assuming the dlv was listening on port 2346, as in the debug-plugin.sh script.
DELVE_PID=$(ps aux | grep "dlv attach.*2346" | grep -v "grep" | awk -F " " '{print $2}')
if [[ -n ${DELVE_PID} ]]
then
	echo "Located existing delve process running with PID: ${DELVE_PID}. Killing."
	kill -9 ${DELVE_PID}
fi

PLUGIN_ID=$(build/bin/manifest id)

if [[ -z ${PLUGIN_ID} ]]
then
    echo "Could not find plugin id. Exiting."
    exit 1
fi

PLUGIN_PID=$(ps aux | grep "plugins/${PLUGIN_ID}" | grep -v "grep" | awk -F " " '{print $2}')

echo "Located Plugin running with PID: ${PLUGIN_PID}. Killing."
kill -9 ${PLUGIN_PID}
