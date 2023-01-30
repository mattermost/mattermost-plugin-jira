{{ .JiraURL }} has been successfully added. To finish the configuration, add an Application Link in your Jira instance following these steps:

1. Navigate to [**Settings > Applications > Application
   Links**]({{ .JiraURL }}/plugins/servlet/applinks/listApplicationLinks)
2. Enter {{ .PluginURL }} as the application link, then click **Create new
   link**.
3. In **Configure Application URL** screen, confirm {{ .PluginURL }} as both
   "Entered URL" and "New URL". Ignore any displayed errors and click
   **Continue**.
   - **Note**: For Jira 9.x - Ensure "Atlassian product" is selected as the Application type.
4. In **Link Applications** screen, set the following values:
  - **Application Name**: Mattermost
  - **Application Type**: Generic Application
  - Check the **Create incoming link** value.
      **(Important)**
  - Click **Continue**
5. In the following **Link Applications** screen, set the following values:
  - **Consumer Key**: `{{ .MattermostKey }}`
  - **Consumer Name**: `Mattermost`
  - **Public Key**:
	```
	{{ .PublicKey }}
	```
  - **Consumer Callback URL**: _leave blank_
  - **Allow 2-legged OAuth**: _leave unchecked_
  - Click **Continue**
6. Use the "/jira connect" command to connect your Mattermost account with your
   Jira account.
7. Click the "More Actions" (...) option of any message in the channel
   (available when you hover over a message).

If you see an option to create a Jira issue, you're all set! If not, refer to our [documentation](https://mattermost.gitbook.io/plugin-jira) for troubleshooting help.
