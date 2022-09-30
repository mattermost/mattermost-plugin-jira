# JIRA Server

If you want to test the plugin against JIRA Server, you can deploy one locally using `docker-compose`.

Keep in mind this version is a configuration for development purposes only and will be dangerous to deploy in production.

## Recommendations

**It is recommended to test with version 7**. Most, if not all, compatibility issues between Jira Server/Cloud exist in version 7.

**Test with version 8** as well when adding new webhook handling logic, or introducing a new Jira API call.

## Instructions

## Optional configuration

You might want to change the Postgres password before running it [here](https://github.com/mattermost/mattermost-plugin-jira/blob/master/docker-compose-jiraserver.yml#L33).

## Installation

The default version is 7.13.1, you can change it through the `VERSION ENVVAR`. When using version 7, by default it's using `JDK_VERSION=8`.

Keep in mind the current Mattermost JIRA Plugin covers version 7 and 8 of JIRA Server now.

```bash
JDK_VERSION=8 VERSION=7.13.1 docker-compose -f docker-compose-jiraserver.yml up
```

If you want to run JIRA server version 8

```bash
# In case you have used if before to build 7, you need to rebuild it without the cache
# to get the parameters
export JDK_VERSION=11
export VERSION=8.22.0
docker-compose -f docker-compose-jiraserver.yml build --no-cache
docker-compose -f docker-compose-jiraserver.yml up
```
