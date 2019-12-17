---
description: A general note about Jira and Mattermost permissions
---

# Permissions

## Can I restrict users from creating or attaching Mattermost messages to Jira issues?

Yes, there is a plugin setting to disable that functionality

## How does Mattermost know which issues a user can see?

Mattermost only displays static messages in the channel and does not enforce Jira permissions on viewers in a channel.  Any messages in a channel can be seen by all users of that channel. Subscriptions to jira issues should be made carefully to avoid unwittingly exposing sensitive Jira issues in a public channel for example. Exposure is limited to the information posted to the channel - to transition an issue, or re-assign it - the user would need to have the appropriate permissions in Jira to perform that action.

## Why does each user need to authenticate with Jira?

The authentication with Jira lets the JiraBot provide personal notifications for each Mattermost/Jira user whenever they are mentioned on an issue or comment or have an issue assigned to them.  Additionally, the plugin uses their authentication information to perform actions on their behalf - searching, viewing, creating, assigning and transitioning issues - all abide by the user's permissions granted within Jira.

