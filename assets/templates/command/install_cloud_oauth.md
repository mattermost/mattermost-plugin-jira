1. Navigate to the [Atlassian Developer Console Page](https://developer.atlassian.com/console/myapps).
2. From the **Create** dropdown, select the option **OAuth 2.0 Integration** to create a new OAuth 2.0 App
3. Name your app according to its purpose, accept the terms and click on **Create** button.
4. Select **Authorization** in the left menu.
5. Next to OAuth 2.0 (3LO), click on the **Add** button to Configure.
6. Enter the Callback URL as `{{ .PluginURL }}/oauth/complete` 
7. Click **Save Changes**. 
8. Select Permissions in the left menu.
9. Next to the Jira API, select Add.

If you see an option to create a Jira issue, you're all set! If not, refer to our [documentation](https://mattermost.gitbook.io/plugin-jira) for troubleshooting help.