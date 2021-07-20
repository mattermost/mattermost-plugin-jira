# Frequently Asked Questions \(FAQ\)

## Why isn't the Jira plugin posting messages to Mattermost?

Try the following troubleshooting steps:

1. Confirm **Site URL** is configured in **System Console > Environment > Web Server**.
   * For older Mattermost versions, this setting is located in **System Console > General > Configuration**.
2. If you specified a JQL query in your Jira webhook page, paste the JQL to Jira issue search and make sure it returns results. If it doesn't, the query may be incorrect. Refer to the [Atlassian documentation](https://confluence.atlassian.com/jirasoftwarecloud/advanced-searching-764478330.html) for help.

If you're using [Legacy Webhooks](https://mattermost.gitbook.io/plugin-jira/administrator-guide/notification-management#legacy-webhooks):

1. Confirm the team URL and channel URL you specified in the Jira webhook URL match up with the path shown in your browser when visiting the channel.
2. For issue updated events, only status changes when the ticket is reopened, or when resolved/closed, and assignee changes are supported for Legacy Webhooks.
3. Use a curl command to make a POST request to the webhook URL. If curl command completes with a `200 OK` response, the plugin is configured correctly. For instance, you can run the following command:

   ```text
   curl -v "https://<your-mattermost-url>/plugins/jira/webhook?secret=<your-secret>&team=<your-team>&channel=<your-channel>&user_id=admin&user_key=admin" --data '{"event":"some_jira_event"}'
   ```

The `<your-mattermost-url>`, `<your-secret>`, `<your-team-url>`, and `<your-channel-url>` fields depend on your setup when configuring the Jira plugin. The curl command won't result in an actual post in your channel.

If you're still having trouble with configuration, please to post in our [Troubleshooting forum](https://forum.mattermost.org/t/how-to-use-the-troubleshooting-forum/150) and we'll be happy to help with issues during setup.

## How do I disable the plugin?

You can disable the Jira plugin at any time from **System Console > Plugins > Management**. Any webhook requests will stop immediately with an error code in **System Console > Logs**. No posts are created until the plugin is re-enabled.

Alternatively, if you only experience problems with Jira-related user interactions in Mattermost such as creating issues, disable these features by setting **Allow users to connect their Mattermost accounts to Jira** to **false** in **System Console > Plugins > Jira**. You will then need to restart  the plugin in **System Console > Plugins > Plugin Management** to reset the plugin state for all users logged in. This setting does not affect Jira webhook notifications.

## Why do I get an error `WebHooks can only use standard http and https ports (80 or 443).`?

Jira only allows webhooks to connect to the standard ports 80 and 443. If you are using a non-standard port, you will need to set up a proxy for the webhook URL, such as:

```text
https://32zanxm6u6.execute-api.us-east-1.amazonaws.com/dev/proxy?url=https%3A%2F%2F<your-mattermost-url>%3A<your-port>%2Fplugins%2Fjira%2Fwebhook%3Fsecret%<your-secret>%26team%3D<your-team-url>%26channel%3D<your-channel-url>
```

The `<your-mattermost-url>`, `<your-port>`, `<your-secret>`, `<your-team-url>`, and `<your-channel-url>` fields depend on your setup from the above steps.

## How do I handle credential rotation for the Jira webhook?

Generate a new secret in **System Console > Plugins > Jira**, then paste the new webhook URL in your Jira webhook configuration.

## What changed in the Jira 2.1 webhook configuration?

In Jira 2.1 there's a modal window for a "Channel Subscription" to Jira issues. This requires a firehose of events to be sent from Jira to Mattermost, and the Jira plugin then "routes" or "drops" the events to particular channels. The Channel Subscription modal \(which you can access by going to a particular channel, then typing `jira /subscribe`\) provides easy access for Mattermost Channel Admins to set up which notifications they want to receive per channel.

Earlier versions of the Jira plugin \(2.0\) used a manual webhook configuration that pointed to specific channels and teams. The webhook configuration was `https://SITEURL/plugins/jira/webhook?secret=WEBHOOKSECRET&team=TEAMURL&channel=CHANNELURL`. This method can still be used to set up notifications from Jira to Mattermost channels if the new Channel Subscription modal doesn't support a particular channel subscription yet.

## How do I manually configure webhooks notifications to be sent to a Mattermost channel?

If your organization's infrastructure is set up in such a way that your Mattermost instance can't connect to your Jira instance, you won't be able to use the Channel Subscriptions feature. Instead, you need to use the [Legacy Webhooks](admininstrator-guide/notification-management.md#legacy-webhooks) feature supported by the Jira plugin.
