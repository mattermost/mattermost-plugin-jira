# Mattermost/Jira Plugin

[![Build Status](https://img.shields.io/circleci/project/github/mattermost/mattermost-plugin-jira/master)](https://circleci.com/gh/mattermost/mattermost-plugin-jira)
[![Code Coverage](https://img.shields.io/codecov/c/github/mattermost/mattermost-plugin-jira/master)](https://codecov.io/gh/mattermost/mattermost-plugin-jira)
[![Release](https://img.shields.io/github/v/release/mattermost/mattermost-plugin-jira)](https://github.com/mattermost/mattermost-plugin-jira/releases/latest)
[![HW](https://img.shields.io/github/issues/mattermost/mattermost-plugin-jira/Up%20For%20Grabs?color=dark%20green&label=Help%20Wanted)](https://github.com/mattermost/mattermost-plugin-jira/issues?q=is%3Aissue+is%3Aopen+sort%3Aupdated-desc+label%3A%22Up+For+Grabs%22+label%3A%22Help+Wanted%22)

This plugin supports a two-way integration between Mattermost and Jira. Jira Core and Jira Software products, for Server, Data Center, and Cloud platforms are supported. It has been tested with versions 7 and 8.

For versions v3.0 and later of this plugin, support for multiple Jira instances is offered for Mattermost E20, Professionsal, and Enterprise Edition configured using [Administrator Slash Commands](https://mattermost.gitbook.io/plugin-jira/administrator-guide/administrator-slash-commands). Note that for versions v3.0.0 and v3.0.1 of this plugin, an E20 license is required to set up multiple Jira instances. 

**Maintainer:** [@mickmister](https://github.com/mickmister)

**Co-Maintainer:** [@jfrerich](https://github.com/jfrerich)

## Feature summary

### Jira to Mattermost notifications

#### Channel subscriptions

Notify your team of the latest updates by sending notifications from your Jira projects to Mattermost channels. You can specify which events trigger a notification - and you can filter out certain types of notifications to keep down the noise.

#### Personal notifications: JiraBot

Each user in Mattermost is connected with their own personal Jira account and notifications for issues where someone is mentioned or assigned an issue is mentioned in your own personal Jira notification bot to help everyone stay on top of their assigned issues.

![A personal JiraBot helps keep you on top of your relevant Jira activities](https://github.com/mattermost/mattermost-plugin-jira/assets/74422101/e15de4fe-1cb3-47d1-9b0d-538ab82ec91d)

### Manage Jira issues in Mattermost

#### Create Jira issues

- Create Jira issues from scratch or based off of a Mattermost message easily.
- Without leaving Mattermost's UI, quickly select the project, issue type and enter other fields to create the issue.

  ![image](https://user-images.githubusercontent.com/13119842/59113188-985a9280-8912-11e9-9def-9a7382b4137e.png)

#### Attach messages to Jira issues

Keep all information in one place by attaching parts of Mattermost conversations in Jira issues as comments.  Then, on the resulting dialog, select the Jira issue you want to attach it to. You may search for issues containing specific text.

![image](https://user-images.githubusercontent.com/13119842/59113267-b627f780-8912-11e9-90ec-417d430de7e6.png)

#### Transition Jira issues

Transition issues without the need to switch to your Jira project. To transition an issue, use the `/jira transition <issue-key> <state>` command.

For instance, `/jira transition EXT-20 done` transitions the issue key **EXT-20** to **Done**.

![image](https://user-images.githubusercontent.com/13119842/59113377-dfe11e80-8912-11e9-8971-f869fa123366.png)

#### Assign Jira issues

Assign issues to other Jira users without the need to switch to your Jira project. To assign an issue, use the `/jira assign` command.

For instance, `/jira assign EXT-20 john` transitions the issue key **EXT-20** to **John**.

## Admin guide

### Prerequisites

* For Jira 2.1 Mattermost Server v5.14+ is required \(certain plugin APIs became available\).
* For Jira 2.0 Mattermost Server v5.12+ is required.

### Installation

#### Marketplace installation

1. Go to **Main Menu > Plugin Marketplace** in Mattermost.
2. Search for "Jira" or manually find the plugin from the list and select **Install**.
3. After the plugin has downloaded and been installed, select the **Configure** button.

#### Manual installation

If your server doesn't have access to the internet, you can download the latest [plugin binary release](https://github.com/mattermost/mattermost-plugin-jira/releases) and upload it to your server via **System Console > Plugin Management**. The releases on this page are the same used by the Marketplace.

### Configuration

#### Step 1: Configure the plugin in Mattermost

1. Go to **Plugins Marketplace > Jira**.
   1. Select **Configure**.
   2. Generate a **Secret** for `Webhook Secret` and `Stats API Secret`.
   3. Optionally change settings for **Notifications permissions** and **Issue Creation** capabilities.
   4. Select **Save**.
2. At the top of the page set **Enable Plugin** to **True**.
3. Select **Save** to enable the Jira plugin.
4. Run `/jira setup` to start configuring the plugin.

#### Step 2: Install the plugin as an application in Jira

To allow users to [create and manage Jira issues across Mattermost channels](../end-user-guide/using-jira-commands.md), install the plugin as an application in your Jira instance. For Jira Server or Data Center instances, post `/jira instance install server <your-jira-url>` to a Mattermost channel as a Mattermost system admin, and follow the steps posted to the channel. For Jira Cloud, post `/jira instance install cloud <your-jira-url>`.

#### Step 3: Configure webhooks on the Jira server

As of Jira 2.1, you need to configure a single webhook for all possible event triggers that you would like to be pushed into Mattermost. This is called a firehose; the plugin gets sent a stream of events from the Jira server via the webhook configured below. The plugin's Channel Subscription feature processes the firehose of data and then routes the events to channels based on your subscriptions.

Use the `/jira webhook` command to get your webhook URL to copy into Jira.

To control Mattermost channel subscriptions, use the `/jira subscribe` command in the channel in which you want to receive subscriptions. Then select the project and event triggers that will post to the channel. To manage all channel subscriptions as an administrator see [Notification Management](../administrator-guide/notification-management.md).

1. To get the appropriate webhook URL, post `/jira webhook <your-jira-url>` to a Mattermost channel as a Mattermost system admin.
2. As a Jira system administrator, go to **Jira Settings > System > WebHooks**.
   * For older versions of Jira, select the gear icon in bottom left corner, then go to **Advanced > WebHooks**.
3. Select **Create a WebHook** to create a new webhook. 
4. Enter a **Name** for the webhook and add the Jira webhook URL retrieved above as the **URL**.
5. Finally, set which issue events send messages to Mattermost channels and select all of the following:
   * Worklog
      * created
      * updated
      * deleted
   * Comment
      * created
      * updated
      * deleted
   * Issue
      * created
      * updated
      * deleted
   * Issue link
      * created
      * deleted
   * Attachment
      * created
      * deleted

6. Choose **Save**.

Previously configured webhooks that point to specific channels are still supported and will continue to work.

### Update the plugin

When a new version of the plugin is released to the **Plugin Marketplace**, the system prompts you to update your current version of the Jira plugin to the newest one. There may be a warning shown if there is a major version change that **may** affect the installation. Generally, updates are seamless and don't interrupt the user experience in Mattermost.

### Administrator slash commands

Administrator slash commands are used to perform system-level functions that require administrator access.

#### Install Jira instances

* `/jira instance install cloud [jiraURL]` - Connect Mattermost to a Jira Cloud instance located at `<jiraURL>`
* `/jira instance install server [jiraURL]` - Connect Mattermost to a Jira Server or Data Center instance located at `<jiraURL>`

#### Uninstall Jira instances

* `/jira instance uninstall cloud [jiraURL]` - Disconnect Mattermost from a Jira Cloud instance located at `<jiraURL>`
* `/jira instance uninstall server [jiraURL]` - Disconnect Mattermost from a Jira Server or Data Center instance located at `<jiraURL>`

#### Manage channel subscriptions

* `/jira subscribe` - Configure the Jira notifications sent to this channel. See the [Notification Management](notification-management#who-can-set-up-notification-subscriptions-for-a-channel) page to see how to configure which users have access to the `subscribe` command.
* `/jira subscribe list` - Display all the the subscription rules set up across all the channels and teams on your Mattermost instance. This command is only available to Mattermost system admins.

#### Other

* `/jira instance alias [URL] [alias-name]` - Assign an alias to an instance
* `/jira instance unalias [alias-name]` - Remove an alias from an instance
* `/jira instance list` - List installed Jira instances
* `/jira instance v2 <jiraURL>` - Set the Jira instance to process \"v2\" webhooks and subscriptions (not prefixed with the instance ID)
* `/jira stats` - Display usage statistics
* `/jira webhook [--instance=<jiraURL>]` -  Show the Mattermost webhook to receive JQL queries
* `/jira v2revert` - Revert to V2 jira plugin data model

### Notification management

#### What are notifications?

Jira notifications are messages sent to a Mattermost channel when a particular event occurs in Jira. They can be subscribed to from a channel via `/jira subscribe` \(managed within Mattermost\). A webhook can be manually set up from Jira to send a message to a particular channel in Mattermost \(managed via Jira\).

Notifications and webhooks can be used together or you can opt for one of them.

![This is a channel notification of a new bug that was created in Jira](https://github.com/mattermost/mattermost-plugin-jira/assets/74422101/e7020c3e-48f6-4825-8193-6a189f6c96eb)

When any webhook event is received from Jira the plugin reviews all the notification subscriptions. If it matches a rule it will post a notification to the channel. If there are no subscription matches, the webhook event is discarded.

The notifications and metadata shown in a channel are not protected by Jira permissions. Anyone in the channel can see what's posted to the channel. However if they do not have the appropriate permission they won't be able to see further details of the issue if they click through to it.

#### What is a notification subscription?

Mattermost users can set up rules that define when a particular event with certain criteria are met in Jira that trigger a notification is sent to a particular channel. These subscription rules can specify the `Jira Project`, `Event Type`, `Issue Type`, and can filter out issues with certain values.

When a user is setting up a notification subscription they'll only see the projects and issue types they have access to within Jira. If they can't see a project in Jira it won't be displayed as an option for that particular user when they are trying to set up a subscription in Mattermost.

An approximate JQL query is output as well. This is not guaranteed to be valid JQL and is only shown as a reference to what the query may look like if converted to JQL.

#### Who can set up notification subscriptions for a channel?

You can specify who can set up a notification subscription in the plugin configuration. First, set which **Mattermost** user roles are allowed to access the subscription functionality:

![image](https://github.com/mattermost/mattermost-plugin-jira/assets/74422101/900695fe-3eca-408f-9fda-afeac14a6798)

You can also specify a comma-separated list of Jira groups the user needs to be a member of to be able to create/edit subscriptions. The user editing a subscription only needs to be a member of one of the listed groups. If this is left blank there will be no restriction on Jira groups.

![image](https://github.com/mattermost/mattermost-plugin-jira/assets/74422101/05a8b36e-4616-4211-b406-ce387a3e0bd5)

A user must meet the criteria of both the Mattermost user settings and Jira group settings in order to edit subscriptions.

#### How can I see all the notification subscriptions that are set up in Mattermost?

While logged in as a system admin, type `/jira subscribe list` in a Mattermost channel.

#### Which notification events are supported?

The following Jira event notifications are supported:

* An issue is created
* Certain fields of an issue issue are updated, configurable per subscription
* An issue is reopened or resolved
* An issue is deleted, when not yet resolved
* Comments created, updated, or deleted

If you’d like to see support for additional events, [let us know](https://mattermost.uservoice.com/forums/306457-general).

![This is the Channel Subscription modal](https://github.com/mattermost/mattermost-plugin-jira/assets/74422101/4dab17fa-5d49-48eb-91b1-cb9596780787)

#### Setting up the webhook in Jira

In order to have Jira post events to your Mattermost instance, you'll need to set up a webhook inside of Jira. Please see the instructions at [configure webhooks on the Jira server](https://mattermost.gitbook.io/plugin-jira/setting-up/configuration#step-2-configure-webhooks-on-the-jira-server).

#### Legacy webhooks

If your organization's infrastructure is set up in such a way that your Mattermost instance can't connect to your Jira instance, you won't be able to use the Channel Subscriptions feature. Instead, you'll need to use the Legacy Webhooks feature (the first iteration of the webhooks feature supported by the Jira plugin).

To generate the webhook URL for a specific channel, run `/jira webhook` and use the URL output in the "Legacy Webhooks" section of the output.

1. As a Jira system administrator, go to **Jira Settings > System > WebHooks**.
   * For older versions of Jira, select the gear icon in bottom left corner, then go to **Advanced > WebHooks**.
2. Select **Create a WebHook** to create a new webhook. Enter a **Name** for the webhook and add the Jira webhook URL [https://SITEURL/plugins/jira/webhook?secret=WEBHOOKSECRET&team=TEAMURL&channel=CHANNELURL](https://SITEURL/plugins/jira/webhook?secret=WEBHOOKSECRET&team=TEAMURL&channel=CHANNELURL) \(for Jira 2.1\) as the **URL**.

   * Replace `TEAMURL` and `CHANNELURL` with the Mattermost team URL and channel URL you want the Jira events to post to. The values should be in lower case.
   * Replace `SITEURL` with the site URL of your Mattermost instance, and `WEBHOOKSECRET` with the secret generated in Mattermost via **System Console > Plugins > Jira**.

   For instance, if the team URL is `contributors`, channel URL is `town-square`, site URL is `https://community.mattermost.com`, and the generated webhook secret is `MYSECRET`, then the final webhook URL would be:

   ```text
   https://community.mattermost.com/plugins/jira/webhook?secret=MYSECRET&team=contributors&channel=town-square
   ```
3. \(Optional\) Set a description and a custom JQL query to determine which tickets trigger events. For more information on JQL queries, refer to the [Atlassian help documentation](https://confluence.atlassian.com/jirasoftwarecloud/advanced-searching-764478330.html).
4. Finally, set which issue events send messages to Mattermost channels, then select **Save**. The following issue events are supported:
   * Issue Created
   * Issue Deleted
   * Issue Updated, including when an issue is reopened or resolved, or when the assignee is changed. Optionally send notifications for comments, see below.

By default, the legacy webhook integration publishes notifications for issue create, resolve, unresolve, reopen, and assign events. To post more events, use the following extra `&`-separated parameters:

- `updated_all=1`: all events
- `updated_comments=1`: all comment events
- `updated_attachment=1`: updated issue attachments
- `updated_description=1`: updated issue description
- `updated_labels=1`: updated issue labels
- `updated_prioity=1`: updated issue priority
- `updated_rank=1`: ranked issue higher or lower
- `updated_sprint=1`: assigned issue to a different sprint
- `updated_status=1`: transitioned issed to a different status, like Done, In Progress
- `updated_summary=1`: renamed issue

Here's an example of a webhook configured to create a post for comment events:

```text
https://community.mattermost.com/plugins/jira/webhook?secret=MYSECRET&team=contributors&channel=town-square&updated_comments=1
```
### Permissions

#### Can I restrict users from creating or attaching Mattermost messages to Jira issues?

Yes, there is a plugin setting to disable that functionality.

#### How does Mattermost know which issues a user can see?

Mattermost only displays static messages in the channel and does not enforce Jira permissions on viewers in a channel. 

Any messages in a channel can be seen by all users of that channel. Subscriptions to Jira issues should be made carefully to avoid unwittingly exposing sensitive Jira issues in a public channel for example. Exposure is limited to the information posted to the channel. To transition an issue, or re-assign it the user needs to have the appropriate permissions in Jira.

#### Why does each user need to authenticate with Jira?

The authentication with Jira lets the JiraBot provide personal notifications for each Mattermost/Jira user whenever they are mentioned on an issue, comment on an issue, or have an issue assigned to them. Additionally, the plugin uses their authentication information to perform actions on their behalf. Tasks such as searching, viewing, creating, assigning, and transitioning issues all abide by the permissions granted to the user within Jira.

### Troubleshooting

If you experience problems with Jira-related user interactions in Mattermost such as creating issues, disable these features by setting **Allow users to connect their Mattermost accounts to Jira** to **false** in **System Console > Plugins > Jira**. This setting does not affect Jira webhook notifications. Then re-enable this plugin in **System Console > Plugins > Plugin Management** to reset the plugin state for all users.

Sometimes the plugin may crash unexpectedly and you may notice a response in red text below the chat window displaying `slash command with trigger of  '/(name)' not found,`. If you check your log file, look for messages that refer to `plugins` and `health check fail`, `ExecuteCommand` etc. 

If you encounter these types of issues you can set `LogSettings.FileLevel` to `DEBUG` in your `config.json` settings. This will enable debug logging and give more verbose error events in the system log. Then try re-enabling the plugin in the system-console. These log results may be requested by others in the forum or by our support team. 

**Note:** If you have a site with high volumes of activity, this setting can cause Log files to expand substantially and may adversely impact the server performance. Keep an eye on your server logs, or only enable it in development environments.

#### Jira/Mattermost user connections

Connecting an account between Mattermost and Jira is a key part of the installation process and requires the end-user to authenticate with Jira and allow access to their Jira account. All `create`, `view`, `assign`, and `transition` operations are done using the logged-in user's Jira access token.

* You must be signed into Mattermost on the same browser you are using to sign into Jira during `connect`.
* The domain end users sign into Mattermost with on that browser must match the SiteURL in `config.json`.

## User guide

### Getting started

To get started with the Jira/Mattermost connector is easy. You'll first need to connect your Jira account with your Mattermost account so the system can perform actions such as searching, viewing and creating Jira issues on your behalf.

1. Go into any channel within Mattermost, and type `/jira connect`.
2. Follow the link that gets presented to you - it will bring you to your Jira server.
3. Select **Allow**.

You may notice that when you type `/` a menu pops up - these are called **slash commands** and bring the functionality of Jira \(and other integrations\) to your fingertips.

![image](https://github.com/mattermost/mattermost-plugin-jira/assets/74422101/aeeaa352-b9f4-4be6-89a7-7f5a413b6d2d)

#### Authentication issues with Jira Cloud

If connecting to a Jira cloud instance, you will need to temporarily enable third-party cookies in your browser during the Jira authentication process.
If you are using Google Chrome, this can be done by going to the browser's cookie settings and selecting "Allow all cookies". You can paste `chrome://settings/cookies` into your address bar to access these settings. After your Jira account is connected, feel free to disable the third-party cookies setting in your browser.

### Use `/jira` commands

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

#### Authenticate with Jira

Use the `/jira connect` and `/jira disconnect` commands to manage the connection between your Mattermost account and Jira account.

#### Create a Jira issue

Use the `/jira issue create` command to create a Jira issue within Mattermost. A form will show that will allow you to fill out the issue. You can prepopulate the issue's summary using the command:

`/jira issue create This is my issue's summary`

#### Transition Jira issues

Transition issues without the need to switch to your Jira project. To transition an issue, use the `/jira transition <issue-key> <state>` command.

For instance, `/jira transition EXT-20 done` transitions the issue key **EXT-20** to **Done**.

![image](https://user-images.githubusercontent.com/13119842/59113377-dfe11e80-8912-11e9-8971-f869fa123366.png)

**Note:**

* States and issue transitions are based on your Jira project workflow configuration. If an invalid state is entered, an ephemeral message is returned mentioning that the state couldn't be found.
* Partial matches work. For example, typing `/jira transition EXT-20 in` will transition to `In Progress`.  However, if there are states of `In Review`, `In Progress`, the plugin bot will ask you to be more specific and display the partial matches.

#### Assign Jira issues

Assign issues to other Jira users without the need to switch to your Jira project. To assign an issue, use the `/jira assign` command. For instance, `/jira assign EXT-20 john` transitions the issue key **EXT-20** to **John**.

**Note**: Partial Matches work with Usernames and Firstname/Lastname.

### Frequently asked auestions \(FAQ\)

#### Why isn't the Jira plugin posting messages to Mattermost?

Try the following troubleshooting steps:

1. Confirm **Site URL** is set in your Mattermost configuration, and that the webhook created in Jira is pointing to this address. The **Site URL** setting can be found at **System Console > Environment > Web Server**. To ensure the URL is correct, run `/jira webhook`, then copy the output and paste it into Jira's webhook setup page.

2. If you specified a JQL query in your Jira webhook setup, paste the JQL to Jira issue search and make sure it returns results. If it doesn't, the query may be incorrect. Refer to the [Atlassian documentation](https://confluence.atlassian.com/jirasoftwarecloud/advanced-searching-764478330.html) for help. Note that you don't need to include a JQL query when setting up the webhook.

If you're using [Legacy Webhooks](https://mattermost.gitbook.io/plugin-jira/administrator-guide/notification-management#legacy-webhooks):

1. Confirm the team URL and channel URL you specified in the Jira webhook URL match up with the path shown in your browser when visiting the channel.

2. Only events described in the Legacy Webhook [docs](https://mattermost.gitbook.io/plugin-jira/administrator-guide/notification-management#legacy-webhooks) are supported.

3. Use a curl command to make a POST request to the webhook URL. If curl command completes with a `200 OK` response, the plugin is configured correctly. For instance, you can run the following command:

   ```text
   curl -X POST -v "https://<your-mattermost-url>/plugins/jira/webhook?secret=<your-secret>&team=<your-team>&channel=<your-channel>&user_id=admin&user_key=admin" --data '{"event":"some_jira_event"}'
   ```

The `<your-mattermost-url>`, `<your-secret>`, `<your-team-url>`, and `<your-channel-url>` fields depend on your setup when configuring the Jira plugin. The curl command won't result in an actual post in your channel.

If you're still having trouble with configuration, please to post in our [Troubleshooting forum](https://forum.mattermost.org/t/how-to-use-the-troubleshooting-forum/150) and we'll be happy to help with issues during setup.

#### How do I disable the plugin?

You can disable the Jira plugin at any time from Mattermost via **System Console > Plugins > Management**. After disabling the plugin, any webhook requests coming from Jira will be ignored. Also, users will not be able to create Jira issues from Mattermost.

If wish to only disable Jira-related user interactions coming from Mattermost such as creating issues, you can disable these features by setting **Allow users to connect their Mattermost accounts to Jira** to **false** in **System Console > Plugins > Jira**. You will then need to restart the plugin in **System Console > Plugins > Plugin Management** to update the UI for users currently logged in to Mattermost, or they can refresh to see the changes. This setting does not affect Jira webhook notifications.

#### Why do I get an error `WebHooks can only use standard http and https ports (80 or 443).`?

Jira only allows webhooks to connect to the standard `ports 80 and 443`. If you are using a non-standard port, you will need to set up a proxy between Jira and your Mattermost instance to let Jira communicate over `port 443`.

#### How do I handle credential rotation for the Jira webhook?

Generate a new secret in **System Console > Plugins > Jira**, then paste the new webhook URL in your Jira webhook configuration.

#### What changed in the Jira 2.1 webhook configuration?

In Jira 2.1 there's a modal window for a "Channel Subscription" to Jira issues. This requires a firehose of events to be sent from Jira to Mattermost, and the Jira plugin then "routes" or "drops" the events to particular channels. The Channel Subscription modal \(which you can access by going to a particular channel, then typing `jira /subscribe`\) provides easy access for Mattermost Channel Admins to set up which notifications they want to receive per channel.

If your organization's infrastructure is set up in such a way that your Mattermost instance can't connect to your Jira instance, the Channel Subscriptions feature won't be accessible. Instead, you will need to use the [Legacy Webhooks](admininstrator-guide/notification-management.md#legacy-webhooks) feature supported by the Jira plugin, which allows a Jira webhook to post to a specific channel.

## License

This repository is licensed under the Apache 2.0 License, except for the [server/enterprise](server/enterprise) directory which is licensed under the [Mattermost Source Available License](LICENSE.enterprise). See [Mattermost Source Available License](https://docs.mattermost.com/overview/faq.html#mattermost-source-available-license) to learn more.

## Development

Read our [development docs](https://mattermost.gitbook.io/plugin-jira/development/environment) for this project, as well as the [Developer Workflow](https://developers.mattermost.com/extend/plugins/developer-workflow/) and [Developer Setup](https://developers.mattermost.com/extend/plugins/developer-setup/) documentation for more information about developing and extending plugins.

### Environment

Join the [Jira plugin channel](https://community.mattermost.com/core/channels/jira-plugin) on our community server to discuss any questions.

Read our documentation about the [Developer Workflow](https://developers.mattermost.com/extend/plugins/developer-workflow/) and [Developer Setup](https://developers.mattermost.com/extend/plugins/developer-setup/) for more information about developing and extending plugins.

This plugin supports both Jira Server (self-hosted) and Jira Cloud instances. There can be slight differences in behavior between the two systems, so it's best to test with both systems individually when introducing new webhook logic, or adding a new Jira API call.

To test your changes against a local instance of Jira Server, you need [Docker](https://docs.docker.com/install) installed, then you can use the `docker-compose.yml` file in this repository to create a Jira instance. Simply run `docker-compose up` in the directory of the repository, and a new Jira server should start up and be available at http://localhost:8080. It can take a few minutes to start up due to Jira Server's startup processes. If the container fails to start with `exit code 137`, you may need to increase the amount of RAM you are allowing docker to use.

To test your changes against a Jira Cloud instance, we recommend starting a 14-day trial, if you don't have a Jira project to test against. More information can be found here: https://www.atlassian.com/software/jira/try.

### Help wanted!

If you're interested in joining our community of developers who contribute to Mattermost - check out the current set of issues [that are being requested](https://github.com/mattermost/mattermost-plugin-jira/issues?q=is%3Aissue+is%3Aopen+label%3AEnhancement).

You can also find issues labeled ["Help Wanted"](https://github.com/mattermost/mattermost-plugin-jira/issues?q=is%3Aissue+is%3Aopen+label%3A%22Help+Wanted%22) in the Jira Repository that we have laid out the primary requirements for and could use some coding help from the community.

### Help and support

- For Mattermost customers - Please open a support case.
- For questions, suggestions, and help, visit the [Jira Plugin channel](https://community.mattermost.com/core/channels/jira-plugin) on our Community server.
- To report a bug, please [open an issue](https://github.com/mattermost/mattermost-plugin-jira/issues).
