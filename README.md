# Mattermost Jira Plugin ![CircleCI branch](https://img.shields.io/circleci/project/github/mattermost/mattermost-plugin-jira/master.svg) ![Codecov branch](https://img.shields.io/codecov/c/github/mattermost/mattermost-plugin-jira/master.svg)

This plugin supports a two-way integration between Mattermost and Jira. For a stable production release, please download the latest version [in the Releases tab](https://github.com/mattermost/mattermost-plugin-jira/releases) and follow [these instructions](#2-installation) for install and configuration.

This plugin supports Jira Core and Jira Software products, for Cloud, Server and Data Center platforms.

Support for multiple Jira instances is considered, but not yet supported.

## Table of Contents

 - [1. Features](#1-features)
 - [2. Installation](#2-installation)
 - [3. Jira v2 Roadmap](#3-jira-v2-roadmap)
 - [4. Development](#4-development)
 - [5. Frequently Asked Questions (FAQ)](#5-frequently-asked-questions-faq)

## 1. Features

### 1.1 Send notifications from Jira to Mattermost

Notify your team of latest updated by sending notifications from your Jira projects to Mattermost channels.

// TODO: Add 1 or 2 screenshots

Notifications are configured with webhooks and offer full JQL support. Configuration is restricted to Jira System Admins only. See [these instructions](#2-installation) for install and configuration.

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

// TODO: Add screenshot

Then, on the resulting issue creation dialog, select the project, issue type and enter other fields to create the issue.

// TODO: Add screenshot

Click **Create** and the Jira issue is now created.

// TODO: Add screenshot

**NOTE**: This plugin does not support all Jira fields. If the project you tried to create an issue for has **required fields** not yet supported, you will be prompted to manually create an issue. Clicking the provided link brings the user to an issue creation screen in Jira, with the fields they entered previously pre-filled.

// TODO: Add 1 or 2 screenshots

The supported Jira fields are:

  - **Project Picker**: Custom fields and the built-in **Project** field.
  - **Single-Line Text**: Custom fields, and built-in fields such as **Summary**, **Environment**.
  - **Multi-Line Text**: Custom fields, and built-in fields such as **Description**.
  - **Single-Choice Issue**: Custom fields, and built-in fields such as **Issue Type** and **Priority**. 

#### 1.2.2 Attach Messages to Jira Issues

Keep all information in one place by attaching parts of Mattermost conversations in Jira issues as comments. To attach a message, click the **More Actions** (...) option of any message in the channel (available when you hover over a message), then select **Attach to Jira Issue**.

// TODO: Add screenshot

Then, on the resulting issue creation dialog, select the project and the issue you want to attach it to. You may search for issues containing specific text.

// TODO: Add screenshot

Click **Attach** and the message is now attached to the Jira issue.

// TODO: Add screenshot from Jira UI

#### 1.2.3 Transition Jira issues

Transition issues without the need to switch to your Jira project. To transition an issue, use the `/jira transition <issue-key> <state>` command.

For instance, `/jira transition MM-1234 done` transitions the issue key **MM-1234** to **Done**.

Note that states and issue transitions are based on your Jira project workflow configuration.

## 2. Installation

#### Step 1: Configure plugin in Mattermost

1. Go to **System Console > Integrations > Jira**, select the username that this plugin is attached to, generate a **Secret** and hit **Save**.
   - You may optionally create a new user account for your Jira plugin, which acts as a bot account posting Jira updates to Mattermost channels.
2. Go to **System Console > Integrations > Plugins** and click **Enable** to enable the Jira plugin.
   - For older Mattermost versions, this setting is located in **System Console > Plugins > Management**.

#### Step 2: Configure webhooks in Jira

If you want to [send notifications from Jira to Mattermost](#11-send-notifications-from-Jira-to-Mattermost), link a Jira project to a Mattermost channel via webhooks.

1. As a Jira System Administrator, go to **Jira Settings > System > WebHooks**.
  - For older versions of Jira, click the gear icon in bottom left corner, then go to **Advanced > WebHooks**.

2. Click **Create a WebHook** to create a new webhook. Choose a unique name and add the JIRA webhook URL https://SITEURL/plugins/jira/webhook?secret=WEBHOOKSECRET&team=TEAMURL&channel=CHANNELURL as the URL.
  - Make sure to replace `TEAMURL` and `CHANNELURL` with the Mattermost team URL and channel URL you want the JIRA events to post to. The values should be in lower case.
  - Moreover, replace `SITEURL` with the site URL of your Mattermost instance, and `WEBHOOKSECRET` with the secret generated in Mattermost via **System Console > Integrations > Jira**

For instance, if the team URL is `contributors`, channel URL is `town-square` and site URL is `https://community.mattermost.com`, and the generated webhook secret is `5JlVk56KPxX629ujeU3MOuxaiwsPzLwh`, then the final webhook URL would be

```
https://community.mattermost.com/plugins/jira/webhook?secret=5JlVk56KPxX629ujeU3MOuxaiwsPzLwh&team=contributors&channel=town-square
```

3. (Optional) Set a description and a custom JQL query to determine which tickets trigger events. For more information on JQL queries, refer to the [Atlassian help documentation](https://confluence.atlassian.com/jirasoftwarecloud/advanced-searching-764478330.html).

4. Finally, set which issue events send messages to Mattermost channels. The following are supported:

// TODO: Add a screenshot

#### Step 3: Link the plugin as an application in Jira

See separate instructions for [Jira Cloud](#step-3-jira-cloud) and for [Jira Server or Data Center](#step-3-jira-server-or-data-center) to complete this step.

##### Step 3: Jira Cloud

As a Mattermost System Admin, post `/jira install cloud <your-jira-url>`, and follow the steps posted to the channel. They are also outlined below:

1. As a Jira System Administrator, go to **Jira Settings > Apps > Manage Apps**. 
  - For older versions of Jira, go to **Administration > Applications > Add-ons > Manage add-ons**
2. Click **Settings** at bottom of page, enable development mode, and apply this change.
  - Enabling development mode allows you to install apps that are not from the Atlassian Marketplace.
3. Click **Upload app**.
4. In the **From this URL field**, enter: %s%s
5. Wait for the app to install. Once completed, you should see an "Installed and ready to go!" message.
6. Use the "/jira connect" command to connect your Mattermost account with your Jira account.
7. Click the "More Actions" (...) option of any message in the channel (available when you hover over a message).

If you see an option to create a Jira issue, you're all set! If not, see our [Frequently Asked Questions](#5-frequently-asked-questions-faq) for troubleshooting help.

##### Step 3: Jira Server or Data Center

As a Mattermost System Admin, post `/jira install server <your-jira-url>`, and follow the steps posted to the channel. They are also outlined below:

1. As a Jira System Administrator, go to **Jira Settings > Applications > Application Links**.
2. Enter your Mattermost URL as the application link, then click **Create new link**.
3. In **Configure Application URL** screen, confirm your Mattermost URL is entered as the "New URL". Ignore any displayed errors and click **Continue**.
4. In **Link Applications** screen, set the following values:
  - **Application Name**: Mattermost
  - **Application Type**: Generic Application
5. Check the **Create incoming link** value, then click **Continue**.
6. In the following **Link Applications** screen, set the following values:
  - **Consumer Key**: Copy the value generated in step 1 for this field.
  - **Consumer Name**: Mattermost
  - **Public Key**: Copy the value generated in step 1 for this field.
7. Click **Continue**.
8. Use the "/jira connect" command to connect your Mattermost account with your Jira account.
9. Click the "More Actions" (...) option of any message in the channel (available when you hover over a message).

If you see an option to create a Jira issue, you're all set! If not, see our [Frequently Asked Questions](#5-frequently-asked-questions-faq) for troubleshooting help.

## 3. Jira v2 Roadmap

### Timeline

The ship target dates are included below. These are subject to change:
  - May 29th: Jira 2.0 Release Candidate cut
       - Deployed to community.mattermost.com for wider testing
       - Shared with customers for early feedback
  - June 16th: Jira 2.0 released as part of Mattermost Server v5.12
  - June 19th: Jira 2.1 Release Candidate cut
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

If you're interested add improvements or bug fixes, review [open Help Wanted issues](https://github.com/mattermost/mattermost-plugin-jira/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22) to get started.

## 4. Development

This plugin contains both a server and web app portion.

Use `make dist` to build distributions of the plugin that you can upload to a Mattermost server.
Use `make check-style` to check the style.
Use `make deploy` to deploy the plugin to your local server.

For additional information on developing plugins, refer to [our plugin developer documentation](https://developers.mattermost.com/extend/plugins/).

To test your changes against Jira locally, we recommend starting a 14-day trial for Jira Software Cloud, if you don't have a Jira project to test against. More information can be found here: https://www.atlassian.com/software/jira/try

## 5. Frequently Asked Questions (FAQ)

### Why doesn't my Jira plugin post any messages to Mattermost?

Try the following troubleshooting steps:

1. Confirm **Site URL** is configured in **System Console > Environment > Web Server**.
   - For older Mattermost versions, this setting is located in **System Console > General > Configuration**.

2. Confirm **User** field is set in **System Console > Integrations > Jira**. The plugin needs to be attached to a user account for the webhook to post messages.
3. Confirm the team URL and channel URL you specified in the Jira webhook URL is in lower case.
4. For issue updated events, only status changes when the ticket is reopened, or when resolved/closed, and assignee changes are supported. If you'd like to see support for additional events, `let us know <https://mattermost.uservoice.com/forums/306457-general>`__.
5. If you specified a JQL query in your Jira webhook page, paste the JQL to Jira issue search and make sure it returns results. If it doesn't, the query may be incorrect. Refer to the `Atlassian documentation <https://confluence.atlassian.com/jirasoftwarecloud/advanced-searching-764478330.html>`__ for help.
6. Use a curl command to make a POST request to the webhook URL. If curl command completes with a ``200 OK`` response, the plugin is configured correctly. For instance, you can run the following command

   ``` 
   curl -v --insecure "https://<your-mattermost-url>/plugins/jira/webhook?secret=<your-secret>&team=<your-team>&channel=<your-channel>&user_id=admin&user_key=admin" --data '{"event":"some_jira_event"}'
   ```

   where `<your-mattermost-url>`, `<your-secret>`, `<your-team-url>` and `<your-channel-url>` depend on your setup when configuring the Jira plugin.
   
   Note that the curl command won't result in an actual post in your channel.

If you are still having trouble with configuration, please to post in our [Troubleshooting forum](https://forum.mattermost.org/t/how-to-use-the-troubleshooting-forum/150) and we'll be happy to help with issues during setup.

### How do I disable the plugin quickly in an emergency?

Disable the Jira plugin any time from **System Console > Integrations > Plugins**. Requests will stop immediately with an error code in **System Console > Logs**. No posts are created until the plugin is re-enabled.

### Why do I get an error ``WebHooks can only use standard http and https ports (80 or 443).``?

Jira only allows webhooks to connect to the standard ports 80 and 443. If you are using a non-standard port, you will need to set up a proxy for the webhook URL, such as

```
https://32zanxm6u6.execute-api.us-east-1.amazonaws.com/dev/proxy?url=https%3A%2F%2F<your-mattermost-url>%3A<your-port>%2Fplugins%2Fjira%2Fwebhook%3Fsecret%<your-secret>%26team%3D<your-team-url>%26channel%3D<your-channel-url>
```
    
where `<your-mattermost-url>`, `<your-port>`, `<your-secret>`, `<your-team-url>` and `<your-channel-url>` depend on your setup from the above steps.

### How do I handle credential rotation for the Jira webhook?

You can generate a new secret in **System Console > Integrations > Jira**, and paste the new webhook URL in your JIRA webhook configuration. 

This might result in downtime of the JIRA plugin, but it should only be a few minutes at most.

### Why does Jira issue creation fail?

// TODO: E.g. https://mattermost.atlassian.net/browse/MM-15828 ?
