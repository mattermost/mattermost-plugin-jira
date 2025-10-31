# Mattermost Jira Plugin

[![Build Status](https://img.shields.io/circleci/project/github/mattermost/mattermost-plugin-jira/master)](https://circleci.com/gh/mattermost/mattermost-plugin-jira)
[![Code Coverage](https://img.shields.io/codecov/c/github/mattermost/mattermost-plugin-jira/master)](https://codecov.io/gh/mattermost/mattermost-plugin-jira)
[![Release](https://img.shields.io/github/v/release/mattermost/mattermost-plugin-jira)](https://github.com/mattermost/mattermost-plugin-jira/releases/latest)
[![HW](https://img.shields.io/github/issues/mattermost/mattermost-plugin-jira/Up%20For%20Grabs?color=dark%20green&label=Help%20Wanted)](https://github.com/mattermost/mattermost-plugin-jira/issues?q=is%3Aissue+is%3Aopen+sort%3Aupdated-desc+label%3A%22Up+For+Grabs%22+label%3A%22Help+Wanted%22)

This plugin supports a two-way integration between Mattermost and Jira. Jira Core and Jira Software products, for Server, Data Center, and Cloud platforms are supported. It has been tested with versions 7 and 8.

For versions v3.0 and later of this plugin, support for multiple Jira instances is offered for Mattermost Professional, Enterprise and Enterprise Advanced plans, configured using [Administrator Slash Commands](https://github.com/mattermost/mattermost-plugin-jira#readme).

See the [Mattermost Product Documentation](https://docs.mattermost.com/integrations-guide/jira.html) for details on installing, configuring, enabling, and using this Mattermost integration.

## Getting started locally

Follow these steps to spin up Mattermost and Jira locally so you can iterate on the plugin with real APIs.

### Prerequisites

- Go **1.21.x**
- Node **18.x** and npm (yarn optional)
- Docker Desktop (for Jira Server + optional Postgres)
- The [mattermost](https://github.com/mattermost/mattermost) repo cloned locally alongside this one

### 1. Run Mattermost with the enterprise repo beside it

Clone both repos into the same parent directory so the enterprise code is detected automatically, then start the server:

```bash
git clone https://github.com/mattermost/mattermost.git
git clone https://github.com/mattermost/enterprise.git

cd mattermost/server
make run
```

If `./bin/mmctl` is not present yet, build it once so you can tweak config from the command line:

```bash
go build -o bin/mmctl ./cmd/mmctl
```

Configure the server so Jira can reach it and so plugins can be uploaded. Either edit `config/config.json` directly or run. Note that localhost urls won't be recognized by the plugin:

```bash
./bin/mmctl --local config set ServiceSettings.SiteURL http://host.docker.internal:8065
./bin/mmctl --local config set PluginSettings.Enable true
./bin/mmctl --local config set PluginSettings.EnableUploads true
```

Restart the server if prompted.

### 2. Build and install the Jira plugin

```bash
cd ../mattermost-plugin-jira
npm install
make dist

# optionally let the Makefile upload straight to your dev server
export MM_SERVICESETTINGS_SITEURL=http://host.docker.internal:8065
export MM_ADMIN_USERNAME=admin
export MM_ADMIN_PASSWORD=admin
make deploy
```

If you prefer to upload manually, go to **System Console → Plugins → Upload Plugin**, select `dist/com.github.mattermost.jira-*.tar.gz`, then enable the plugin under **Plugin Management**.

For a tighter edit loop, leave `make watch` running – it rebuilds and redeploys whenever you save.

### 3. Launch a local Jira Server in Docker

```bash
docker pull atlassian/jira-software:9.4.30
docker run --name jira \
  -d -p 8080:8080 \
  atlassian/jira-software:9.4.30
```

Open `http://localhost:8080`, request a **Server** evaluation license from [my.atlassian.com](https://my.atlassian.com/), and finish the setup wizard. Wherever the wizard asks for your Mattermost URL, use `http://host.docker.internal:8065` so the container can call back into your dev server.

### 4. Connect the plugin to Jira

Back in Mattermost, confirm the plugin responds to `/jira help`, then install your local instance:

```bash
/jira instance install server http://localhost:8080
```

The command guides you through creating the Application Link inside Jira, exchanging the consumer key/secret, and configuring webhooks. Once that completes, run `/jira connect` to link your Mattermost user, and use `/jira create` or `/jira subscribe` to test the round trip.

At this point you can edit plugin code, let `make watch` redeploy, and immediately exercise the change against a real Jira API.

### Troubleshooting the local setup

- **Plugin immediately disables itself when enabled** – check `logs/mattermost.log`. If you see `please configure the Mattermost server's SiteURL`, set `ServiceSettings.SiteURL` to a reachable address (e.g. `http://host.docker.internal:8065`) and re-enable the plugin.
- **Upload button missing in System Console** – ensure `PluginSettings.EnableUploads` is `true` in `config/config.json`. You may need to restart the server after changing it.
- **`./bin/mmctl` not found** – build it with `go build -o bin/mmctl ./cmd/mmctl` from the `mattermost/server` directory.
- **Jira setup page refuses license** – generate a **Server** evaluation key for the Server ID shown in the wizard via [my.atlassian.com](https://my.atlassian.com/). Make sure you select the Server deployment type, not Cloud.
- **`XSRF Security Token Missing` during Jira setup** – refresh the page or restart the Docker container and re-run the wizard in a fresh browser session; the token expired.
- **Slash commands complain about unreachable Mattermost URL** – Jira is running in a container, so `localhost` refers to the container itself. Use `http://host.docker.internal:8065` (macOS/Linux with Docker Desktop) or your machine’s LAN IP so Jira can reach Mattermost.

## Feature summary

### Jira to Mattermost notifications

#### Channel subscriptions

Notify your team of the latest updates by sending notifications from your Jira projects to Mattermost channels. You can specify which events trigger a notification - and you can filter out certain types of notifications to keep down the noise.

#### Personal notifications: JiraBot

Each user in Mattermost is connected with their own personal Jira account and notifications for issues where someone is mentioned or assigned an issue is mentioned in your own personal Jira notification bot to help everyone stay on top of their assigned issues.

![A personal JiraBot helps keep you on top of your relevant Jira activities](https://github.com/mattermost/mattermost-plugin-jira/assets/74422101/e15de4fe-1cb3-47d1-9b0d-538ab82ec91d)

### Manage Jira issues in Mattermost

#### Create Jira issues

- Create Jira issues from scratch or based off of a Mattermost message easily.
- Without leaving Mattermost's UI, quickly select the project, issue type and enter other fields to create the issue.

  ![image](https://user-images.githubusercontent.com/13119842/59113188-985a9280-8912-11e9-9def-9a7382b4137e.png)

#### Attach messages to Jira issues

Keep all information in one place by attaching parts of Mattermost conversations in Jira issues as comments.  Then, on the resulting dialog, select the Jira issue you want to attach it to. You may search for issues containing specific text.

![image](./assets/attach-from-post.png)

![image](https://user-images.githubusercontent.com/13119842/59113267-b627f780-8912-11e9-90ec-417d430de7e6.png)

#### Transition Jira issues

Transition issues without the need to switch to your Jira project. To transition an issue, use the `/jira transition <issue-key> <state>` command.

For instance, `/jira transition EXT-20 done` transitions the issue key **EXT-20** to **Done**.

![image](https://user-images.githubusercontent.com/13119842/59113377-dfe11e80-8912-11e9-8971-f869fa123366.png)

#### Assign Jira issues

Assign issues to other Jira users without the need to switch to your Jira project. To assign an issue, use the `/jira assign` command.

For instance, `/jira assign EXT-20 john` transitions the issue key **EXT-20** to **John**.

## License

This repository is licensed under the Apache 2.0 License, except for the [server/enterprise](server/enterprise) directory which is licensed under the [Mattermost Source Available License](LICENSE.enterprise). See [Mattermost Source Available License](https://docs.mattermost.com/overview/faq.html#mattermost-source-available-license) to learn more.

## Development

This plugin contains both a server and web app portion. Read our documentation about the [Developer Workflow](https://developers.mattermost.com/integrate/plugins/developer-workflow/) and [Developer Setup](https://developers.mattermost.com/integrate/plugins/developer-setup/) for more information about developing and extending plugins.

### Releasing new versions

The version of a plugin is determined at compile time, automatically populating a `version` field in the [plugin manifest](plugin.json):
* If the current commit matches a tag, the version will match after stripping any leading `v`, e.g. `1.3.1`.
* Otherwise, the version will combine the nearest tag with `git rev-parse --short HEAD`, e.g. `1.3.1+d06e53e1`.
* If there is no version tag, an empty version will be combined with the short hash, e.g. `0.0.0+76081421`.

To disable this behaviour, manually populate and maintain the `version` field.

## How to Release

To trigger a release, follow these steps:

1. **For Patch Release:** Run the following command:
    ```
    make patch
    ```
   This will release a patch change.

2. **For Minor Release:** Run the following command:
    ```
    make minor
    ```
   This will release a minor change.

3. **For Major Release:** Run the following command:
    ```
    make major
    ```
   This will release a major change.

4. **For Patch Release Candidate (RC):** Run the following command:
    ```
    make patch-rc
    ```
   This will release a patch release candidate.

5. **For Minor Release Candidate (RC):** Run the following command:
    ```
    make minor-rc
    ```
   This will release a minor release candidate.

6. **For Major Release Candidate (RC):** Run the following command:
    ```
    make major-rc
    ```
   This will release a major release candidate.

### Environment

Join the [Jira plugin channel](https://community.mattermost.com/core/channels/jira-plugin) on our community server to discuss any questions.

Read our documentation about the [Developer Workflow](https://developers.mattermost.com/extend/plugins/developer-workflow/) and [Developer Setup](https://developers.mattermost.com/extend/plugins/developer-setup/) for more information about developing and extending plugins.

This plugin supports both Jira Server (self-hosted) and Jira Cloud instances. There can be slight differences in behavior between the two systems, so it's best to test with both systems individually when introducing new webhook logic, or adding a new Jira API call.

To test your changes against a local instance of Jira Server, you need [Docker](https://docs.docker.com/install) installed, then you can use the `docker-compose.yml` file in this repository to create a Jira instance. Simply run `docker-compose up` in the directory of the repository, and a new Jira server should start up and be available at http://localhost:8080. It can take a few minutes to start up due to Jira Server's startup processes. If the container fails to start with `exit code 137`, you may need to increase the amount of RAM you are allowing docker to use.

To test your changes against a Jira Cloud instance, we recommend starting a 14-day trial, if you don't have a Jira project to test against. More information can be found here: https://www.atlassian.com/software/jira/try.

### Help wanted!

If you're interested in joining our community of developers who contribute to Mattermost - check out the current set of issues [that are being requested](https://github.com/mattermost/mattermost-plugin-jira/issues?q=is%3Aissue+is%3Aopen+label%3AEnhancement).

You can also find issues labeled ["Help Wanted"](https://github.com/mattermost/mattermost-plugin-jira/issues?q=is%3Aissue+is%3Aopen+label%3A%22Help+Wanted%22) in the Jira Repository that we have laid out the primary requirements for and could use some coding help from the community.

### Help and support

- For Mattermost customers - Please open a support case.
- For questions, suggestions, and help, visit the [Jira Plugin channel](https://community.mattermost.com/core/channels/jira-plugin) on our Community server.
- To report a bug, please [open an issue](https://github.com/mattermost/mattermost-plugin-jira/issues).
