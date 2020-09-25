---
description: A general note about Jira and Mattermost permissions
---

# Permissions

## Can I restrict users from creating or attaching Mattermost messages to Jira issues?

Yes, there is a plugin setting to disable that functionality.

## How does Mattermost know which issues a user can see?

Mattermost only displays static messages in the channel and does not enforce Jira permissions on viewers in a channel. 

Any messages in a channel can be seen by all users of that channel. Subscriptions to Jira issues should be made carefully to avoid unwittingly exposing sensitive Jira issues in a public channel for example. Exposure is limited to the information posted to the channel. To transition an issue, or re-assign it the user needs to have the appropriate permissions in Jira.

## Why does each user need to authenticate with Jira?

The authentication with Jira lets the JiraBot provide personal notifications for each Mattermost/Jira user whenever they are mentioned on an issue, comment on an issue, or have an issue assigned to them. Additionally, the plugin uses their authentication information to perform actions on their behalf. Tasks such as searching, viewing, creating, assigning, and transitioning issues all abide by the permissions granted to the user within Jira.
