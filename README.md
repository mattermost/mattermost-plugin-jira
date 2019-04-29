# Mattermost Jira Plugin ![CircleCI branch](https://img.shields.io/circleci/project/github/mattermost/mattermost-plugin-jira/master.svg) ![Codecov branch](https://img.shields.io/codecov/c/github/mattermost/mattermost-plugin-jira/master.svg)

This plugin supports a two-way integration between Mattermost and Jira. It is currently in development and not yet considered stable for production. For a stable production release, please download the latest version [in the Releases tab](https://github.com/mattermost/mattermost-plugin-jira/releases) and follow [this documentation](https://docs.mattermost.com/integrations/jira.html) for install and configuration.

We are looking for help from our community to complete the development of v2.0 of the Mattermost Jira plugin. If you're interested, review [open Help Wanted issues](https://github.com/mattermost/mattermost-plugin-jira/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22) to get started.

## 1. Features

Below is a list of features currently supported. Each is considered Beta and may be removed in subsequent updates. We are also aware of a few known issues and are tracking development [in Jira](https://mattermost.atlassian.net/issues/?jql=status%20in%20(%22At%20Risk%22%2C%20Backlog%2C%20%22Future%20Consideration%22%2C%20%22In%20Progress%22%2C%20Open%2C%20Pending%2C%20%22Planned%3A%20Backlog%22%2C%20%22Planned%3A%20Scheduled%22%2C%20Reopened%2C%20Resolved%2C%20Reverted%2C%20%22Selected%20for%20Development%22%2C%20Submitted%2C%20%22To%20Do%22)%20AND%20%22Epic%20Link%22%20%3D%20MM-12474%20ORDER%20BY%20cf%5B10011%5D%20ASC%2C%20created%20DESC).

- Notifications for 10+ issue events from Jira to Mattermost with full JQL support, using webhooks. This includes notifications for the following events: issue created; issue transitioned to “Reopened”, “In Progress”, "Submitted" or “Resolved”; issue deleted or closed; comments created, updated or deleted; assignee, title, description, priority, sprint or rank updated; attachments or labels added; attachments or labels removed
- Create Jira issues via Mattermost UI (Desktop App and browser only)
- Scan posts for Jira issue references, and automatically add them as comments in Jira. (Note: This functionality will likely be removed when v2.0 is released)
- Send direct messages for Jira at-mentions and issue assignments
- Slash commands for
  - `/jira connect` - Connect to a Jira project and subscribe to events.
  - `/jira disconnect` - Disconnect from a Jira project and subscribe from events.
  - `/jira instance` - Manage connected Jira instances. Must have System Admin role in Mattermost.

The above are supported for Jira Core and Jira Software, both for Cloud and Server platforms.

## 2. Jira Plugin v2.0 Roadmap

Below is a full list of features planned for Jira plugin v2.0.

We are looking for help from our community to complete the development of v2.0 of the Mattermost Jira plugin. If you're interested, review [open Help Wanted issues](https://github.com/mattermost/mattermost-plugin-jira/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22) to get started.

- Notifications for 10+ issue events from Jira to Mattermost with full JQL support, using webhooks. This includes notifications for the following events: issue created; issue transitioned to “Reopened”, “In Progress”, "Submitted" or “Resolved”; issue deleted or closed; comments created, updated or deleted; assignee, title, description, priority, sprint or rank updated; attachments or labels added; attachments or labels removed
- Create Jira issues via Mattermost UI (Desktop App and browser only)
- Attach Mattermost messages to Jira issues via Mattermost UI (Desktop App and browser only)
- Preview Jira issues in Mattermost when a ticket is referenced
- Send direct messages for Jira at-mentions and issue assignments
- Slash commands for
  - `/jira connect` - Connect to a Jira project and subscribe to events.
  - `/jira disconnect` - Disconnect from a Jira project and subscribe from events.
  - `/jira subscribe` - Subscribe a Mattermost channel to receive notifications for above issue updates in a Jira project.
  - `/jira instance` - Manage connected Jira instances. Must have System Admin role in Mattermost.
  - `/jira create [description]` - Create a Jira ticket.
  - `/jira assign <issue-key> <assignee>` - Assign a Jira issue specified by `issue-key`. `assignee` must be a member of the Jira project.
  - `/jira transition <issue-key> <state>` - Transition a Jira issue specified by `issue-key`. `state` must be a valid Jira state such as "Done".
  - `/jira settings preview [on/off]` - Set whether previews of Jira issues are shown.
  - `/jira settings notifications [on/off]` - Set whether Direct Message notifications are sent for assignments and comments in assigned issues.
  - `/jira view <issue-key>` - View a Jira issue specified by `issue-key`.  

## 3. Installation

### 3.1 Jira Cloud (Core + Software)

#### 3.1.1 Mattermost

1. Download this binary: // XXX
2. Go to **System Console > Plugins (Beta) > Management** and upload the plugin from step 1. If you don't have the ability to upload plugins, uploads may be disabled on your server. To enable them, set **PluginSettings > EnableUploads** to `true` in your `config.json` file.
3. Once uploaded, the Jira plugin will appear in a list of installed plugins. Click **Enable** to enable it.
4. Go to **System Console > Plugins (Beta) > Management > Jira**, select the username that this plugin is attached to, generate a **Secret** and hit **Save**.

#### 3.1.2 Jira

As a Jira administrator, you have two steps to configure the plugin:

#### 3.1.3 Jira - Deploy via Jira AppConnect

1. As a Jira System Administrator, go to **Jira Settings > Apps > Manage Apps**. 
  - For older versions of Jira, go to **Administration > Applications > Add-ons > Manage add-ons**

2. Click **Settings** at bottom of page and enable development mode, and apply this change.
  - Enabling development mode allows you to install apps that are not from the Atlassian Marketplace.
  - Mattermost has opted not to publish to Atlassian Marketplace, as we don’t have an efficient way to provide a callback URL for the app in the marketplace, to enable user-specific interactions between the Mattermost server and Jira cloud instance.

3. Click **Upload add-on**, then enter the Atlassian Connect app descriptor in the form https://<SITEURL>/plugins/jira/atlassian-connect.json where `SITEURL` is your [Mattermost Site URL](https://docs.mattermost.com/administration/config-settings.html#site-url). Select **Upload**.

4. Wait for the app to install.

You're all set. Users can now connect their Mattermost account with Jira using `/jira connect`.

#### 3.1.4 Jira - Configure Webhooks

Only the Jira System Admin has permissions to link a Jira project to a Mattermost channel via webhooks. We are planning to allow anyone with access to subscribe a Jira project to a Mattermost channel via `/jira subscribe`.

1. As a Jira System Administrator, go to **Jira Settings > System > WebHooks**.
  - For older versions of Jira, click the gear icon in bottom left corner, then go to **Advanced > WebHooks**.

2. Click **Create a WebHook** to create a new webhook. Choose a unique name and add the JIRA webhook URL https://SITEURL/plugins/jira/webhook?secret=WEBHOOKSECRET&team=TEAMURL&channel=CHANNELURL as the URL.
  - Make sure to replace `TEAMURL` and `CHANNELURL` with the Mattermost team URL and channel URL you want the JIRA events to post to. The values should be in lower case.
  - Moreover, replace `SITEURL` with the site URL of your Mattermost instance, and `WEBHOOKSECRET` with the secret generated in Mattermost via **System Console > Plugins (Beta) > Jira**

For instance, if the team URL is `contributors`, channel URL is `town-square` and site URL is `https://community.mattermost.com`, and the generated webhook secrete is `5JlVk56KPxX629ujeU3MOuxaiwsPzLwh`, then the final webhook URL would be

```
https://ci-linux-postgres.mattermost.com/plugins/jira/webhook?secret=5JlVk56KPxX629ujeU3MOuxaiwsPzLwh&team=contributors&channel=town-square
```

3. (Optional) Set a description and a custom JQL query to determine which types of tickets trigger events. For more information on JQL queries, refer to the [Atlassian help documentation](https://confluence.atlassian.com/jirasoftwarecloud/advanced-searching-764478330.html).

4. Finally, set which issue events send messages to Mattermost channels. The following are supported:

 - Issue: Created, Updated, Deleted
 - Comment: Created, Updated, Deleted

### 3.2 Jira Server (Core + Software)

Documentation to be added.

## 4. Developing

This plugin contains both a server and web app portion.

Use `make dist` to build distributions of the plugin that you can upload to a Mattermost server.
Use `make check-style` to check the style.
Use `make deploy` to deploy the plugin to your local server.

For additional information on developing plugins, refer to [our plugin developer documentation](https://developers.mattermost.com/extend/plugins/).

To test your changes against Jira locally, we recommend starting a 14-day trial for Jira Software Cloud, if you don't have a Jira project to test against. More information can be found here: https://www.atlassian.com/software/jira/try
