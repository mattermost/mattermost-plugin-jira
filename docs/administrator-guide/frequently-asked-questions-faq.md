# Frequently Asked Questions \(FAQ\)

## Why isn't the Jira plugin posting messages to Mattermost?

Try the following troubleshooting steps:

1. Confirm **Site URL** is set in your Mattermost configuration, and that the webhook created in Jira is pointing to this address. The **Site URL** setting can be found at **System Console > Environment > Web Server**. To ensure the URL is correct, run `/jira webhook`, then copy the output and paste it into Jira's webhook setup page.

2. If you specified a JQL query in your Jira webhook setup, paste the JQL to Jira issue search and make sure it returns results. If it doesn't, the query may be incorrect. Refer to the [Atlassian documentation](https://confluence.atlassian.com/jirasoftwarecloud/advanced-searching-764478330.html) for help. Note that you don't need to include a JQL query when setting up the webhook.

If you're using [Legacy Webhooks](https://mattermost.gitbook.io/plugin-jira/administrator-guide/notification-management#legacy-webhooks):

1. Confirm the team URL and channel URL you specified in the Jira webhook URL match up with the path shown in your browser when visiting the channel.

2. Only events described in the Legacy Webhook [docs](https://mattermost.gitbook.io/plugin-jira/administrator-guide/notification-management#legacy-webhooks) are supported

3. Use a curl command to make a POST request to the webhook URL. If curl command completes with a `200 OK` response, the plugin is configured correctly. For instance, you can run the following command:

   ```text
   curl -X POST -v "https://<your-mattermost-url>/plugins/jira/webhook?secret=<your-secret>&team=<your-team>&channel=<your-channel>&user_id=admin&user_key=admin" --data '{"event":"some_jira_event"}'
   ```

The `<your-mattermost-url>`, `<your-secret>`, `<your-team-url>`, and `<your-channel-url>` fields depend on your setup when configuring the Jira plugin. The curl command won't result in an actual post in your channel.

If you're still having trouble with configuration, please to post in our [Troubleshooting forum](https://forum.mattermost.org/t/how-to-use-the-troubleshooting-forum/150) and we'll be happy to help with issues during setup.

## How do I disable the plugin?

You can disable the Jira plugin at any time from Mattermost via **System Console > Plugins > Management**. After disabling the plugin, any webhook requests coming from Jira will be ignored. Also, users will not be able to create Jira issues from Mattermost.

If wish to only disable Jira-related user interactions coming from Mattermost such as creating issues, you can disable these features by setting **Allow users to connect their Mattermost accounts to Jira** to **false** in **System Console > Plugins > Jira**. You will then need to restart the plugin in **System Console > Plugins > Plugin Management** to update the UI for users currently logged in to Mattermost, or they can refresh to see the changes. This setting does not affect Jira webhook notifications.

## Why do I get an error `WebHooks can only use standard http and https ports (80 or 443).`?

Jira only allows webhooks to connect to the standard `ports 80 and 443`. If you are using a non-standard port, you will need to set up a proxy between Jira and your Mattermost instance to let Jira communicate over `port 443`.

## How do I handle credential rotation for the Jira webhook?

Generate a new secret in **System Console > Plugins > Jira**, then paste the new webhook URL in your Jira webhook configuration.

## What changed in the Jira 2.1 webhook configuration?

In Jira 2.1 there's a modal window for a "Channel Subscription" to Jira issues. This requires a firehose of events to be sent from Jira to Mattermost, and the Jira plugin then "routes" or "drops" the events to particular channels. The Channel Subscription modal \(which you can access by going to a particular channel, then typing `jira /subscribe`\) provides easy access for Mattermost Channel Admins to set up which notifications they want to receive per channel.

If your organization's infrastructure is set up in such a way that your Mattermost instance can't connect to your Jira instance, the Channel Subscriptions feature won't be accessible. Instead, you will need to use the [Legacy Webhooks](admininstrator-guide/notification-management.md#legacy-webhooks) feature supported by the Jira plugin, which allows a Jira webhook to post to a specific channel.
