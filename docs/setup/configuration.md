# Configuration

### Step 1: Configure the plugin in Mattermost

1. Go to **Plugins Marketplace > Jira**.
   1. Click **Configure**.
   2. Generate a **Secret** for `Webhook Secret`.
   3. Optionally change settings for **Notifications permissions** and **Issue Creation** capabilities.
   4. Click **Save**.
2. At the top of the page set **Enable Plugin** to **True**.
3. Choose **Save** to enable the Jira plugin.

Once you have the plugin configured, you may continue the process by typing `/jira setup` on any channel. This will prompt a direct message from jira bot, which will guide you through the next setup steps as described below.

### Step 2: Install the plugin as an application in Jira

To allow users to [create and manage Jira issues across Mattermost channels](../end-user-guide/using-jira-commands.md), install the plugin as an application in your Jira instance. For Jira Server or Data Center instances, post `/jira instance install server <your-jira-url>` to a Mattermost channel as a Mattermost System Admin, and follow the steps posted to the channel. For Jira Cloud, post `/jira instance install cloud <your-jira-url>`.

### Step 3: Configure webhooks on the Jira server

As of Jira 2.1, you need to configure a single webhook for all possible event triggers that you would like to be pushed into Mattermost. This is called a firehose; the plugin gets sent a stream of events from the Jira server via the webhook configured below. The plugin's Channel Subscription feature processes the firehose of data and then routes the events to channels based on your subscriptions.

Use the `/jira webhook` command to get your webhook URL to copy into Jira.

To control Mattermost channel subscriptions, use the `/jira subscribe` command in the channel in which you want to receive subscriptions. Then select the project and event triggers that will post to the channel. To manage all channel subscriptions as an administrator see [Notification Management](../administrator-guide/notification-management.md).


1. To get the appropriate webhook URL, post `/jira webhook <your-jira-url>` to a Mattermost channel as a Mattermost System Admin.
2. As a Jira System Administrator, go to **Jira Settings > System > WebHooks**.
   * For older versions of Jira, click the gear icon in bottom left corner, then go to **Advanced > WebHooks**.
3. Click **Create a WebHook** to create a new webhook. 
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

For granular control of how issues are sent to certain channels in your Mattermost instance, you can use the Jira plugin's Channel Subscriptions feature in the Mattermost interface. The webhook JQL used on Jira's side should be made broad enough to make full use out of the feature.

You can create multiple webhooks in Jira to point to the same endpoint on your Mattermost instance. This is useful when you only want issues from certain projects, or issues that fit a specific JQL criteria to be sent to your Mattermost instance. Larger organizations may want to filter the webhooks by project to minimize load on their Jira server, if they only need specific projects used for the webhook feature.

### Step 4: Install the plugin as an application in Jira

To control Mattermost channel subscriptions, use the command `/jira subscribe` in the channel in which you want to receive subscriptions. Then select the project and event triggers that will post to the channel. To manage all channel subscriptions as an administrator see [Notification Management](../admininstrator-guide/notification-management.md).

If you encounter any issues during the installation process, see our [Frequently Asked Questions](../administrator-guide/frequently-asked-questions-faq.md) for troubleshooting help, or open an issue in the [Mattermost Forum](http://forum.mattermost.org).
