# Environment

To contribute to the project see https://www.mattermost.org/contribute-to-mattermost.

Join the [Jira plugin channel](https://community.mattermost.com/core/channels/jira-plugin) on our community server to discuss any questions.

Read our documentation about the [Developer Workflow](https://developers.mattermost.com/extend/plugins/developer-workflow/) and [Developer Setup](https://developers.mattermost.com/extend/plugins/developer-setup/) for more information about developing and extending plugins.

This plugin supports both Jira Server (self-hosted) and Jira Cloud instances. There can be slight differences in behavior between the two systems, so it's best to test with both systems individually when introducing new webhook logic, or adding a new Jira API call.

To test your changes against a Jira Cloud instance, we recommend starting a 14-day trial, if you don't have a Jira project to test against. More information can be found here: https://www.atlassian.com/software/jira/try.

If you are contributing to a feature that requires multiple Jira instances to be installed, please enable [ServiceSettings.EnableDeveloper](https://docs.mattermost.com/configure/configuration-settings.html#enable-developer-mode) in your server's config in order to circumvent the Enterprise license requirement. 

### Run a local instance of Jira server

To test your changes against a local instance of Jira Server, you need [Docker](https://docs.docker.com/install) installed. 

**Pre-requisite**
As per the [sizing recommendations from Jira](https://confluence.atlassian.com/jirakb/jira-server-sizing-guide-975033809.html), it requires atleast a minimum memory of 8GB. Hence it is advised to increase the amount of resources allocated for your Docker to use. Here are the steps on how to do this using Docker Desktop:
- Click on the Settings icon on the Docker Desktop
- Navigate to Resources section
- Ensure that the Memory is set to atleast to 8GB or more
- Ensure that the CPUs is set to atleast 4 or more
- Click on Apply and Restart

**Setup your local Jira server**
- Run the command `make jira` in the root of the repository to spin up the Jira server
Note: It can take a few minutes to start up due to Jira Server's startup processes. If the container fails to start with `exit code 137`, you may need to increase the amount of RAM you are allowing docker to use. 

- Once the above command completes, visit the URL http://localhost:8080 to start setting up the Jira Server
- Select the option "Set it up for me" and click Continue to MyAtlassian
- Select the option - "Jira Software (Data Center)" from the list of License Types
- Enter any Organization Name and click on Generate License
- Click on the "Yes" button on the Confirmation dialog `Please confirm that you wish to install the license key on the following server: localhost`
- You will then be redirected to setup Administrator account. Enter all the details and click Next
- Now sit back while the set up completes. It might take a few minutes to complete

**Note:** You cannot use `localhost` to connect the Jira plugin to your server, you can use a proxy if needed.