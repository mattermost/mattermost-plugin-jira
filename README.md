# Mattermost Jira Plugin

[![Build Status](https://img.shields.io/circleci/project/github/mattermost/mattermost-plugin-jira/master.svg)](https://circleci.com/gh/mattermost/mattermost-plugin-jira)
[![Code Coverage](https://img.shields.io/codecov/c/github/mattermost/mattermost-plugin-jira/master.svg)](https://codecov.io/gh/mattermost/mattermost-plugin-jira)

This plugin supports a two-way integration between Mattermost and Jira. For a stable production release, please download the latest version [in the Releases tab](https://github.com/mattermost/mattermost-plugin-jira/releases) and follow [these instructions](#2-configuration) for install and configuration.

This plugin supports Jira Core and Jira Software products, for Server, Data Center and Cloud platforms.  It has been tested with versions 7 and 8.

Support for multiple Jira instances is considered, but not yet supported.

## Table of Contents

 - [1. Features](#1-features)
 - [2. Configuration](#2-configuration)
 - [3. Jira v2 Roadmap](#3-jira-v2-roadmap)
 - [4. Development](#4-development)
 - [5. Frequently Asked Questions (FAQ)](#5-frequently-asked-questions-faq)

## 1. Features

### 1.1 Send notifications from Jira to Mattermost

Notify your team of the latest updates by sending notifications from your Jira projects to Mattermost channels.

![image](https://user-images.githubusercontent.com/13119842/59113100-6cd7a800-8912-11e9-9e23-3639c0eb9c4d.png)

![image](https://user-images.githubusercontent.com/13119842/59113138-7f51e180-8912-11e9-9fc5-3077ba90a8a8.png)

Notifications are configured with webhooks and offer full JQL support. Configuration is restricted to Jira System Admins only. See [these instructions](#2-configuration) for install and configuration.

The following Jira event notifications are supported:

  - Issue created
  - Issue updated, including when an issue is reopened or resolved, or when the assignee is changed
  - Issue deleted when not yet resolved
  - Comments created, updated or deleted

If you’d like to see support for additional events, [let us know](https://mattermost.uservoice.com/forums/306457-general).

### 1.2 Create and manage Jira issues in Mattermost

Connect your Mattermost account to Jira via `/jira connect` command, then create and manage issues across Mattermost channels. You can disconnect your account anytime via `/jira disconnect`.

#### 1.2.1 Create Jira issues

Create Jira issues from a Mattermost message by clicking the **More Actions** (...) option of any message in the channel (available when you hover over a message), then selecting **Create Jira Issue**.

Then, on the resulting issue creation dialog, select the project, issue type and enter other fields to create the issue.

![image](https://user-images.githubusercontent.com/13119842/59113188-985a9280-8912-11e9-9def-9a7382b4137e.png)

Click **Create** and the Jira issue is now created, including any file attachments part of the Mattermost message.

![image](https://user-images.githubusercontent.com/13119842/59113219-a4deeb00-8912-11e9-9741-5ddc8a4b51fa.png)

**NOTE**: This plugin does not support all Jira fields. If the project you tried to create an issue for has **required fields** not yet supported, you will be prompted to manually create an issue. Clicking the provided link brings you to an issue creation screen in Jira, with the fields entered previously pre-filled.

The supported Jira fields are:

  - **Project Picker**: Custom fields and the built-in **Project** field.
  - **Single-Line Text**: Custom fields, and built-in fields such as **Summary**, **Environment**.
  - **Multi-Line Text**: Custom fields, and built-in fields such as **Description**.
  - **Single-Choice Issue**: Custom fields, and built-in fields such as **Issue Type** and **Priority**. 

#### 1.2.2 Attach Messages to Jira Issues

Keep all information in one place by attaching parts of Mattermost conversations in Jira issues as comments. To attach a message, click the **More Actions** (...) option of any message in the channel (available when you hover over a message), then select **Attach to Jira Issue**.

Then, on the resulting dialog, select the issue you want to attach it to. You may search for issues containing specific text.

![image](https://user-images.githubusercontent.com/13119842/59113267-b627f780-8912-11e9-90ec-417d430de7e6.png)

Click **Attach** and the message is attached to the selected Jira issue as a comment.

#### 1.2.3 Transition Jira issues

Transition issues without the need to switch to your Jira project. To transition an issue, use the `/jira transition <issue-key> <state>` command.

For instance, `/jira transition EXT-20 done` transitions the issue key **EXT-20** to **Done**.

![image](https://user-images.githubusercontent.com/13119842/59113377-dfe11e80-8912-11e9-8971-f869fa123366.png)

Note that states and issue transitions are based on your Jira project workflow configuration. If an invalid state is entered, an ephemeral message is returned mentioning that the state couldn't be found.

## 2. Configuration

#### Step 1: Configure plugin in Mattermost

1. Go to **System Console > Plugins > Jira**, select the username that this plugin is attached to, generate a **Secret** and hit **Save**.
   - You may optionally create a new user account for your Jira plugin, which acts as a bot account posting Jira updates to Mattermost channels.
2. Go to **System Console > Plugins > Management** and click **Enable** to enable the Jira plugin.

#### Step 2: Configure webhooks in Jira

If you want to [send notifications from Jira to Mattermost](#11-send-notifications-from-Jira-to-Mattermost), link a Jira project to a Mattermost channel via webhooks.

1. As a Jira System Administrator, go to **Jira Settings > System > WebHooks**.
  - For older versions of Jira, click the gear icon in bottom left corner, then go to **Advanced > WebHooks**.

2. Click **Create a WebHook** to create a new webhook. Enter a **Name** for the webhook and add the JIRA webhook URL https://SITEURL/plugins/jira/webhook?secret=WEBHOOKSECRET&team=TEAMURL&channel=CHANNELURL as the **URL**.
  - Replace `TEAMURL` and `CHANNELURL` with the Mattermost team URL and channel URL you want the JIRA events to post to. The values should be in lower case.
  - Replace `SITEURL` with the site URL of your Mattermost instance, and `WEBHOOKSECRET` with the secret generated in Mattermost via **System Console > Plugins > Jira**.

  For instance, if the team URL is `contributors`, channel URL is `town-square`, site URL is `https://community.mattermost.com`, and the generated webhook secret is `5JlVk56KPxX629ujeU3MOuxaiwsPzLwh`, then the final webhook URL would be

  ```
  https://community.mattermost.com/plugins/jira/webhook?secret=5JlVk56KPxX629ujeU3MOuxaiwsPzLwh&team=contributors&channel=town-square
  ```

3. (Optional) Set a description and a custom JQL query to determine which tickets trigger events. For more information on JQL queries, refer to the [Atlassian help documentation](https://confluence.atlassian.com/jirasoftwarecloud/advanced-searching-764478330.html).

4. Finally, set which issue events send messages to Mattermost channels, then hit **Save**. The following issue events are supported:
     - Issue Created; Issue Deleted
     - Issue Updated, including when an issue is reopened or resolved, or when the assignee is changed. Optionally send notifications for comments, see below.

**Note**: You can send notifications for comments by selecting **Issue Updated**, then adding `&updated_comments=1` to the end of the webhook URL, such as 

```
https://community.mattermost.com/plugins/jira/webhook?secret=5JlVk56KPxX629ujeU3MOuxaiwsPzLwh&team=contributors&channel=town-square&updated_comments=1
```

This sends all comment notifications to a Mattermost channel, including public and private comments, so be cautious of which channel you send these notifications to.

#### Step 3: Install the plugin as an application in Jira

If you want to allow users to [create and manage Jira issues across Mattermost channels](#11-create-and-manage-jira-issues-in-mattermost), install the plugin as an application in your Jira instance. For Jira Server or Data Center instances, post `/jira install server <your-jira-url>` to a Mattermost channel as a Mattermost System Admin, and follow the steps posted to the channel. For Jira Cloud, post `/jira install cloud <your-jira-url>`.

If you face issues installing the plugin, see our [Frequently Asked Questions](#5-frequently-asked-questions-faq) for troubleshooting help, or open an issue in the [Mattermost Forum](http://forum.mattermost.org).

**Note**: If you experience problems with Jira-related user interactions in Mattermost such as creating issues, disable these features by setting **Allow users to connect their Mattermost accounts to Jira** to false in **System Console > Plugins > Jira**. This setting does not affect Jira webhook notifications. After changing this setting to false, disable, then re-enable this plugin in **System Console > Plugins > Plugin Management** to reset the plugin state for all users.

## 3. Jira v2 Roadmap

### Timeline

The ship target dates are included below. These are subject to change:
  - June 16th: Jira 2.0 released as part of Mattermost Server v5.12
  - June 25th: Jira 2.1 Release Candidate cut
       - Deployed to community.mattermost.com for wider testing
       - Shared with customers for early feedback
  - August 16th: Jira 2.1 released as part of Mattermost Server v5.14

### Jira 2.0 Features

Below is a full list of features scheduled for v2.0.

- Send notifications for issue events from Jira to Mattermost with full JQL support, using webhooks. Restricted to Jira System Admins only.
   - This includes notifications for the following events: issue created; issue transitioned to “Reopened”, “In Progress”, "Submitted" or “Resolved”; issue deleted or closed; comments created, updated or deleted; assignee updated
- Create Jira issues via Mattermost UI (Desktop App and browser only)
- Attach Mattermost messages to Jira issues via Mattermost UI (Desktop App and browser only)
- Slash commands for
  - `/jira connect` - Connect your Mattermost account to Jira. Enables you to create issues, attach messages to Jira and transition issues in Mattermost.
  - `/jira disconnect` - Disconnect your Mattermost account from Jira.
  - `/jira transition <issue-key> <state>` - Transition a Jira issue specified by `issue-key`. `state` must be a valid Jira state such as "Done".

### Jira 2.1 Features

Below is a full list of features scheduled for v2.1.

- Subscribe Jira projects to Mattermost channels through the Mattermost user interface. Available to any users with appropriate permissions.
   - Subscribed notifications include the following events: issue created; issue transitioned to “Reopened”, “In Progress”, "Submitted" or “Resolved”; issue deleted or closed; comments created, updated or deleted; assignee, title, description, priority, sprint or rank updated; attachments or labels added; attachments or labels removed
- Send notifications for issue events from Jira to Mattermost with full JQL support, using webhooks. Restricted to Jira System Admins only.
   - This includes notifications for the following events: issue created; issue transitioned to “Reopened”, “In Progress”, "Submitted" or “Resolved”; issue deleted or closed; comments created, updated or deleted; assignee updated
- Create Jira issues via Mattermost UI (Desktop App and browser only)
- Attach Mattermost messages to Jira issues via Mattermost UI (Desktop App and browser only)
- Receive direct messages for Jira at-mentions and issue assignments
- Slash commands for
  - `/jira connect` - Connect your Mattermost account to Jira. Enables you to create issues, attach messages to Jira and take other quick actions in Mattermost.
  - `/jira disconnect` - Disconnect your Mattermost account from Jira.
  - `/jira assign <issue-key> <assignee>` - Assign a Jira issue specified by `issue-key`. `assignee` must be a member of the Jira project.
  - `/jira create [description]` - Create a Jira ticket.
  - `/jira settings notifications [on/off]` - Set whether Direct Message notifications are sent for assignments and comments in assigned issues.
  - `/jira transition <issue-key> <state>` - Transition a Jira issue specified by `issue-key`. `state` must be a valid Jira state such as "Done".
  - `/jira view <issue-key>` - View a Jira issue specified by `issue-key`.  

If you're interested to add improvements or bug fixes, review [open Help Wanted issues](https://github.com/mattermost/mattermost-plugin-jira/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22) to get started.

## 4. Development

This plugin contains both a server and web app portion.

Use `make dist` to build distributions of the plugin that you can upload to a Mattermost server.
Use `make check-style` to check the style.
Use `make deploy` to deploy the plugin to your local server.

For additional information on developing plugins, refer to [our plugin developer documentation](https://developers.mattermost.com/extend/plugins/).

To test your changes against Jira locally, we recommend starting a 14-day trial for Jira Software Cloud, if you don't have a Jira project to test against. More information can be found here: https://www.atlassian.com/software/jira/try.

## 5. Frequently Asked Questions (FAQ)

### Why doesn't my Jira plugin post any messages to Mattermost?

Try the following troubleshooting steps:

1. Confirm **Site URL** is configured in **System Console > Environment > Web Server**.
   - For older Mattermost versions, this setting is located in **System Console > General > Configuration**.

2. Confirm **User** field is set in **System Console > Plugins > Jira**. The plugin needs to be attached to a user account for the webhook to post messages.
3. Confirm the team URL and channel URL you specified in the Jira webhook URL is in lower case.
4. For issue updated events, only status changes when the ticket is reopened, or when resolved/closed, and assignee changes are supported. If you'd like to see support for additional events, [let us know](https://mattermost.uservoice.com/forums/306457-general).
5. If you specified a JQL query in your Jira webhook page, paste the JQL to Jira issue search and make sure it returns results. If it doesn't, the query may be incorrect. Refer to the [Atlassian documentation](https://confluence.atlassian.com/jirasoftwarecloud/advanced-searching-764478330.html) for help.
6. Use a curl command to make a POST request to the webhook URL. If curl command completes with a ``200 OK`` response, the plugin is configured correctly. For instance, you can run the following command

   ``` 
   curl -v --insecure "https://<your-mattermost-url>/plugins/jira/webhook?secret=<your-secret>&team=<your-team>&channel=<your-channel>&user_id=admin&user_key=admin" --data '{"event":"some_jira_event"}'
   ```

   where `<your-mattermost-url>`, `<your-secret>`, `<your-team-url>` and `<your-channel-url>` depend on your setup when configuring the Jira plugin.
   
   Note that the curl command won't result in an actual post in your channel.

If you are still having trouble with configuration, please to post in our [Troubleshooting forum](https://forum.mattermost.org/t/how-to-use-the-troubleshooting-forum/150) and we'll be happy to help with issues during setup.

### How do I disable the plugin quickly in an emergency?

Disable the Jira plugin any time from **System Console > Plugins > Management**. Requests will stop immediately with an error code in **System Console > Logs**. No posts are created until the plugin is re-enabled.

Alternatively, if you only experience problems with Jira-related user interactions in Mattermost such as creating issues, disable these features by setting **Allow users to connect their Mattermost accounts to Jira** to false in **System Console > Plugins > Jira**. This setting does not affect Jira webhook notifications. After changing this setting to false, disable, then re-enable this plugin in **System Console > Plugins > Plugin Management** to reset the plugin state for all users.

### Why do I get an error ``WebHooks can only use standard http and https ports (80 or 443).``?

Jira only allows webhooks to connect to the standard ports 80 and 443. If you are using a non-standard port, you will need to set up a proxy for the webhook URL, such as

```
https://32zanxm6u6.execute-api.us-east-1.amazonaws.com/dev/proxy?url=https%3A%2F%2F<your-mattermost-url>%3A<your-port>%2Fplugins%2Fjira%2Fwebhook%3Fsecret%<your-secret>%26team%3D<your-team-url>%26channel%3D<your-channel-url>
```
    
where `<your-mattermost-url>`, `<your-port>`, `<your-secret>`, `<your-team-url>` and `<your-channel-url>` depend on your setup from the above steps.

### How do I handle credential rotation for the Jira webhook?

You can generate a new secret in **System Console > Plugins > Jira**, and paste the new webhook URL in your JIRA webhook configuration. 

This might result in downtime of the JIRA plugin, but it should only be a few minutes at most.
