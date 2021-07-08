---
description: Use slash commands to interact with Jira issues
---

# Using `/jira` commands

The available commands are listed below.

* `/jira help` - Launch the Jira plugin command line help syntax
* `/jira info` - Display information about the current user and the Jira plugin
* `/jira connect [jiraURL]` - Connect your Mattermost account to your Jira account
* `/jira disconnect [jiraURL]` - Disconnect your Mattermost account from your Jira account
* `/jira issue assign [issue-key] [assignee]` - Change the assignee of a Jira issue
* `/jira issue create [text]` - Create a new Issue with 'text' inserted into the description field
* `/jira issue transition [issue-key] [state]` - Change the state of a Jira issue
* `/jira issue unassign [issue-key]` - Unassign the Jira issue
* `/jira issue view [issue-key]` - View the details of a specific Jira issue
* `/jira instance settings` - View your user settings
* `/jira instance settings [setting] [value]` - Update your user settings

**Note:** For the `/jira instance settings` command, [setting] can be `notifications` and [value] can be `on` or `off`

### Authenticating with Jira

Use the `/jira connect` and `/jira disconnect` commands to manage the connection between your Mattermost account and Jira account.

### Creating a Jira issue

Use the `/jira issue create` command to create a Jira issue within Mattermost. A form will show that will allow you to fill out the issue. You can prepopulate the issue's summary using the command:

`/jira issue create This is my issue's summary`

### Transition Jira issues

Transition issues without the need to switch to your Jira project. To transition an issue, use the `/jira transition <issue-key> <state>` command.

For instance, `/jira transition EXT-20 done` transitions the issue key **EXT-20** to **Done**.

![image](https://user-images.githubusercontent.com/13119842/59113377-dfe11e80-8912-11e9-8971-f869fa123366.png)

**Note:**

* States and issue transitions are based on your Jira project workflow configuration. If an invalid state is entered, an ephemeral message is returned mentioning that the state couldn't be found.
* Partial matches work. For example, typing `/jira transition EXT-20 in` will transition to `In Progress`.  However, if there are states of `In Review`, `In Progress`, the plugin bot will ask you to be more specific and display the partial matches.

### Assign Jira issues

Assign issues to other Jira users without the need to switch to your Jira project. To assign an issue, use the `/jira assign` command. For instance, `/jira assign EXT-20 john` transitions the issue key **EXT-20** to **John**.

**Note:**

* Partial Matches work with Usernames and Firstname/Lastname
