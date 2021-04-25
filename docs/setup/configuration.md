# Configuration

### Step 1: Configure the plugin in Mattermost

1. Go to **Plugins Marketplace > Jira**.
   1. Click **Configure**.
   2. Generate a **Secret** for `Webhook Secret` and `Stats API Secret`.
   3. Optionally change settings for **Notifications permissions** and **Issue Creation** capabilities.
   4. Click **Save**.
2. At the top of the page set **Enable Plugin** to **True**.
3. Choose **Save** to enable the Jira plugin.

### Step 2: Configure webhooks on the Jira server

As of Jira 2.1, you need to configure a single webhook for all possible event triggers that you would like to be pushed into Mattermost. This is called a firehose; the plugin gets sent a stream of events from the Jira server via the webhook configured below. The plugin's Channel Subscription feature processes the firehose of data and then routes the events to channels based on your subscriptions.

Previously configured webhooks that point to specific channels are still supported and will continue to work.

To control Mattermost channel subscriptions, use the command `/jira subscribe` in the channel in which you want to receive subscriptions. Then select the project and event triggers that will post to the channel. To manage all channel subscriptions as an administrator see [Notification Management](../administrator-guide/notification-management.md).

1. As a Jira System Administrator, go to **Jira Settings > System > WebHooks**.
   * For older versions of Jira, click the gear icon in bottom left corner, then go to **Advanced > WebHooks**.
2. Click **Create a WebHook** to create a new webhook. 
3. Enter a **Name** for the webhook and add the Jira webhook URL [https://SITEURL/plugins/jira/api/v2/webhook?secret=WEBHOOKSECRET](https://SITEURL/plugins/jira/api/v2/webhook?secret=WEBHOOKSECRET) as the **URL**.

   * Replace `SITEURL` with the site URL of your Mattermost instance, and `WEBHOOKSECRET` with the secret generated in Mattermost via **System Console > Plugins > Jira**. For instance, if the site URL is `https://community.mattermost.com`, and the generated webhook secret is `5JlVk56KPxX629ujeU3MOuxaiwsPzLwh`, then the final webhook URL would be:

   ```text
   https://community.mattermost.com/plugins/jira/api/v2/webhook?secret=5JlVk56KPxX629ujeU3MOuxaiwsPzLwh
   ```

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

### Step 3: Install the plugin as an application in Jira

To allow users to [create and manage Jira issues across Mattermost channels](../end-user-guide/using-jira-commands.md), install the plugin as an application in your Jira instance. For Jira Server or Data Center instances, post `/jira install server <your-jira-url>` to a Mattermost channel as a Mattermost System Admin, and follow the steps posted to the channel. For Jira Cloud, post `/jira install cloud <your-jira-url>`.

If you encounter any issues during the installation process, see our [Frequently Asked Questions](../administrator-guide/frequently-asked-questions-faq.md) for troubleshooting help, or open an issue in the [Mattermost Forum](http://forum.mattermost.org).
