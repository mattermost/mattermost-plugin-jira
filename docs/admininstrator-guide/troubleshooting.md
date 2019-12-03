---
description: >-
  If you encounter issues, please try out some of these troubleshooting steps
  and share the results with your support staff or the community to provide
  better diagnoses of problems.
---

# Troubleshooting

**Note**: If you experience problems with Jira-related user interactions in Mattermost such as creating issues, disable these features by setting **Allow users to connect their Mattermost accounts to Jira** to false in **System Console &gt; Plugins &gt; Jira**. This setting does not affect Jira webhook notifications. After changing this setting to false, disable, then re-enable this plugin in **System Console &gt; Plugins &gt; Plugin Management** to reset the plugin state for all users.

Sometimes the plugin may crash unexpectedly, you may notice:

* you will see a response in red text below the chat window displaying `slash command with trigger of  '/(name)' not found,`  - If you check your log file, look for messages that refer to "plugins" and "health check fail", "ExecuteCommand" etc. 

If you encounter these types of issues - you can set `LogSettings.FileLevel` to `DEBUG` in your config.json settings. This will enable debug logging and give more verbose error events in the system log. Then try re-enabling the plugin in the system-console. These log results may be requested by others in the forum or by our support team. **Note** If you have a site with high volumes of activity, this setting can cause Log files to expand substantially and may adversely impact the server performance. Keep an eye on your server logs, or only enable in development environments.

### Jira/Mattermost user connections

Connecting an account between Mattermost and Jira is a key part of the installation process and requires the end-user to authenticate with Jira and allow access to their Jira account. All `create`, `view`, `assign` and `transition` operations are done using the logged in user's Jira access token.

* You must be signed into Mattermost on the same browser you are using to sign into Jira during `connect`
* The domain end-users sign into Mattermost with on that browser must match the SiteURL in config.json

