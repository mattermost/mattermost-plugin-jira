# Configuration

### Step 1: Configure the plugin in Mattermost

1. Go to **Plugins Marketplace > Jira**.
   1. Click **Configure**.
   2. Generate a **Secret** for `Webhook Secret` and `Stats API Secret`.
   3. Optionally change settings for **Notifications permissions** and **Issue Creation** capabilities.
   4. Click **Save**.
2. At the top of the page set **Enable Plugin** to **True**.
3. Choose **Save** to enable the Jira plugin.

### Step 2: Install the plugin as an application in Jira

To allow users to [create and manage Jira issues across Mattermost channels](../end-user-guide/using-jira-commands.md), install the plugin as an application in your Jira instance. For Jira Server or Data Center instances, post `/jira instance install server <your-jira-url>` to a Mattermost channel as a Mattermost System Admin, and follow the steps posted to the channel. For Jira Cloud, post `/jira instance install cloud <your-jira-url>`.

### Step 3: Configure webhooks on the Jira server

As of Jira 2.1, you need to configure a single webhook for all possible event triggers that you would like to be pushed into Mattermost. This is called a firehose; the plugin gets sent a stream of events from the Jira server via the webhook configured below. The plugin's Channel Subscription feature processes the firehose of data and then routes the events to channels based on your subscriptions.

1. To get the appropriate webhook URL, post `/jira webhhok <your-jira-url>` to a Mattermost channel as a Mattermost System Admin.
1. As a Jira System Administrator, go to **Jira Settings > System > WebHooks**.
   * For older versions of Jira, click the gear icon in bottom left corner, then go to **Advanced > WebHooks**.
2. Click **Create a WebHook** to create a new webhook. 
3. Enter a **Name** for the webhook and add the Jira webhook URL retrieved above as the **URL**.
3. Finally, set which issue events send messages to Mattermost channels and select all of the following:
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
4. Choose **Save**.

Previously configured webhooks that point to specific channels are still supported and will continue to work.

To control Mattermost channel subscriptions, use the command `/jira subscribe` in the channel in which you want to receive subscriptions. Then select the project and event triggers that will post to the channel. To manage all channel subscriptions as an administrator see [Notification Management](../admininstrator-guide/notification-management.md).

If you encounter any issues during the installation process, see our [Frequently Asked Questions](../administrator-guide/frequently-asked-questions-faq.md) for troubleshooting help, or open an issue in the [Mattermost Forum](http://forum.mattermost.org).
