# Administrator Slash Commands

Administrator slash commands are used to perform system-level functions that require administrator access.

## Install Jira instances

* `/jira instance install cloud [jiraURL]` - Connect Mattermost to a Jira Cloud instance located at `<jiraURL>`
* `/jira instance install server [jiraURL]` - Connect Mattermost to a Jira Server or Data Center instance located at `<jiraURL>`

## Uninstall Jira instances

* `/jira instance uninstall cloud [jiraURL]` - Disconnect Mattermost from a Jira Cloud instance located at `<jiraURL>`
* `/jira instance uninstall server [jiraURL]` - Disconnect Mattermost from a Jira Server or Data Center instance located at `<jiraURL>`

## Manage Channel Subscriptions

* `/jira subscribe` - Configure the Jira notifications sent to this channel. See the [Notification Management](notification-management#who-can-set-up-notification-subscriptions-for-a-channel) page to see how to configure which users have access to the `subscribe` command.
* `/jira subscribe list` - Display all the the subscription rules setup across all the channels and teams on your Mattermost instance. This command is only available to Mattermost system administrators.

## Other

* `/jira instance alias [URL] [alias-name]` - Assign an alias to an instance
* `/jira instance unalias [alias-name]` - Remove an alias from an instance
* `/jira instance list` - List installed Jira instances
* `/jira instance v2 <jiraURL>` - Set the Jira instance to process \"v2\" webhooks and subscriptions (not prefixed with the instance ID)
* `/jira stats` - Display usage statistics
* `/jira webhook [--instance=<jiraURL>]` -  Show the Mattermost webhook to receive JQL queries
* `/jira v2revert` - Revert to V2 jira plugin data model
