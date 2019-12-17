# Frequently Asked Questions \(FAQ\)

### Why doesn't my Jira plugin post any messages to Mattermost?

Try the following troubleshooting steps:

1. Confirm **Site URL** is configured in **System Console &gt; Environment &gt; Web Server**.
   * For older Mattermost versions, this setting is located in **System Console &gt; General &gt; Configuration**.
2. Confirm **User** field is set in **System Console &gt; Plugins &gt; Jira**. The plugin needs to be attached to a user account for the webhook to post messages.
3. Confirm the team URL and channel URL you specified in the Jira webhook URL is in lower case.
4. For issue updated events, only status changes when the ticket is reopened, or when resolved/closed, and assignee changes are supported. If you'd like to see support for additional events, [let us know](https://mattermost.uservoice.com/forums/306457-general).
5. If you specified a JQL query in your Jira webhook page, paste the JQL to Jira issue search and make sure it returns results. If it doesn't, the query may be incorrect. Refer to the [Atlassian documentation](https://confluence.atlassian.com/jirasoftwarecloud/advanced-searching-764478330.html) for help.
6. Use a curl command to make a POST request to the webhook URL. If curl command completes with a `200 OK` response, the plugin is configured correctly. For instance, you can run the following command

   ```text
   curl -v --insecure "https://<your-mattermost-url>/plugins/jira/webhook?secret=<your-secret>&team=<your-team>&channel=<your-channel>&user_id=admin&user_key=admin" --data '{"event":"some_jira_event"}'
   ```

   where `<your-mattermost-url>`, `<your-secret>`, `<your-team-url>` and `<your-channel-url>` depend on your setup when configuring the Jira plugin.

   Note that the curl command won't result in an actual post in your channel.

If you are still having trouble with configuration, please to post in our [Troubleshooting forum](https://forum.mattermost.org/t/how-to-use-the-troubleshooting-forum/150) and we'll be happy to help with issues during setup.

### How do I disable the plugin quickly in an emergency?

Disable the Jira plugin any time from **System Console &gt; Plugins &gt; Management**. Requests will stop immediately with an error code in **System Console &gt; Logs**. No posts are created until the plugin is re-enabled.

Alternatively, if you only experience problems with Jira-related user interactions in Mattermost such as creating issues, disable these features by setting **Allow users to connect their Mattermost accounts to Jira** to false in **System Console &gt; Plugins &gt; Jira**. This setting does not affect Jira webhook notifications. After changing this setting to false, disable, then re-enable this plugin in **System Console &gt; Plugins &gt; Plugin Management** to reset the plugin state for all users.

### Why do I get an error `WebHooks can only use standard http and https ports (80 or 443).`?

Jira only allows webhooks to connect to the standard ports 80 and 443. If you are using a non-standard port, you will need to set up a proxy for the webhook URL, such as

```text
https://32zanxm6u6.execute-api.us-east-1.amazonaws.com/dev/proxy?url=https%3A%2F%2F<your-mattermost-url>%3A<your-port>%2Fplugins%2Fjira%2Fwebhook%3Fsecret%<your-secret>%26team%3D<your-team-url>%26channel%3D<your-channel-url>
```

where `<your-mattermost-url>`, `<your-port>`, `<your-secret>`, `<your-team-url>` and `<your-channel-url>` depend on your setup from the above steps.

### How do I handle credential rotation for the Jira webhook?

You can generate a new secret in **System Console &gt; Plugins &gt; Jira**, and paste the new webhook URL in your JIRA webhook configuration.

This might result in downtime of the JIRA plugin, but it should only be a few minutes at most.

### What changed in the Jira 2.1 Webhook configuration?

In Jira 2.1 there is a modal window for a "Channel Subscription" to Jira issues. This requires a firehose of events to be sent from Jira to Mattermost, and the Jira plugin then "routes" or "drops" the events to particular channels. The Channel Subscription modal \(which you can access by going to a particular channel, then typing `jira /subscribe`\) provides easy access for Mattermost Channel Admins to setup which notifications they want to receive per channel.

Earlier versions of the Jira plugin \(2.0\) used a manual webhook configuration that pointed to specific channels and teams. The webhook configuration was `https://SITEURL/plugins/jira/webhook?secret=WEBHOOKSECRET&team=TEAMURL&channel=CHANNELURL`. This method can still be used to setup notifications from Jira to Mattermost channels if the new Channel Subscription modal can't support a particular channel subscription yet.

#### How do I manually Configure webhooks notifications to be sent to a Mattermost Channel?

If you want to send notifications from Jira to Mattermost, link a Jira project to a Mattermost channel via webhooks.

1. As a Jira System Administrator, go to **Jira Settings &gt; System &gt; WebHooks**.
   * For older versions of Jira, click the gear icon in bottom left corner, then go to **Advanced &gt; WebHooks**.
2. Click **Create a WebHook** to create a new webhook. Enter a **Name** for the webhook and add the JIRA webhook URL [https://SITEURL/plugins/jira/webhook?secret=WEBHOOKSECRET&team=TEAMURL&channel=CHANNELURL](https://SITEURL/plugins/jira/webhook?secret=WEBHOOKSECRET&team=TEAMURL&channel=CHANNELURL) \(for Jira 2.1\) as the **URL**.

   * Replace `TEAMURL` and `CHANNELURL` with the Mattermost team URL and channel URL you want the JIRA events to post to. The values should be in lower case.
   * Replace `SITEURL` with the site URL of your Mattermost instance, and `WEBHOOKSECRET` with the secret generated in Mattermost via **System Console &gt; Plugins &gt; Jira**.

   For instance, if the team URL is `contributors`, channel URL is `town-square`, site URL is `https://community.mattermost.com`, and the generated webhook secret is `5JlVk56KPxX629ujeU3MOuxaiwsPzLwh`, then the final webhook URL would be

   ```text
   https://community.mattermost.com/plugins/jira/webhook?secret=5JlVk56KPxX629ujeU3MOuxaiwsPzLwh&team=contributors&channel=town-square
   ```

3. \(Optional\) Set a description and a custom JQL query to determine which tickets trigger events. For more information on JQL queries, refer to the [Atlassian help documentation](https://confluence.atlassian.com/jirasoftwarecloud/advanced-searching-764478330.html).
4. Finally, set which issue events send messages to Mattermost channels, then hit **Save**. The following issue events are supported:
   * Issue Created; Issue Deleted
   * Issue Updated, including when an issue is reopened or resolved, or when the assignee is changed. Optionally send notifications for comments, see below.

**Note**: You can send notifications for comments by selecting **Issue Updated**, then adding `&updated_comments=1` to the end of the webhook URL, such as

```text
https://community.mattermost.com/plugins/jira/webhook?secret=5JlVk56KPxX629ujeU3MOuxaiwsPzLwh&team=contributors&channel=town-square&updated_comments=1
```

This sends all comment notifications to a Mattermost channel, including public and private comments, so be cautious of which channel you send these notifications to.

