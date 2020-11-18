# Environment

To contribute to the project see https://www.mattermost.org/contribute-to-mattermost.

Join the [Jira plugin channel](https://community.mattermost.com/core/channels/jira-plugin) on our community server to discuss any questions.

Read our documentation about the [Developer Workflow](https://developers.mattermost.com/extend/plugins/developer-workflow/) and [Developer Setup](https://developers.mattermost.com/extend/plugins/developer-setup/) for more information about developing and extending plugins.

This plugin supports both Jira Server (self-hosted) and Jira Cloud instances. There can be slight differences in behavior between the two systems, so it's best to test with both systems individually when introducing new webhook logic, or adding a new Jira API call.

To test your changes against a local instance of Jira Server, you need [Docker](https://docs.docker.com/install) installed, then you can use the `docker-compose.yml` file in this repository to create a Jira instance. Simply run `docker-compose up` in the directory of the repository, and a new Jira server should start up and be available at http://localhost:8080. It can take a few minutes to start up due to Jira Server's startup processes. If the container fails to start with `exit code 443`, you may need to increase the amount of RAM you are allowing docker to use.

To test your changes against a Jira Cloud instance, we recommend starting a 14-day trial, if you don't have a Jira project to test against. More information can be found here: https://www.atlassian.com/software/jira/try.
