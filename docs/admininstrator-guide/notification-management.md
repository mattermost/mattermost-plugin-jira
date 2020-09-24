---
description: Centrally view all notification subscriptions in your system
---

# Notification Management

## What are Notifications?

Jira notifications are messages sent to a Mattermost channel when a particular event occurs in Jira. They can be subscribed to from a channel via `/jira subscribe` \(managed within Mattermost\). A webhook can be manually set up from Jira to send a message to a particular channel in Mattermost \(managed via Jira\).

Notifications and webhooks can be used together or you can opt for one of them.

![This is a channel notification of a new bug that was created in Jira](../.gitbook/assets/image%20%281%29.png)

When any webhook event is received from Jira the plugin reviews all the notification subscriptions. If it matches a rule it will post a notification to the channel. If there are no subscription matches, the webhook event is discarded.

The notifications and metadata shown in a channel are not protected by Jira permissions. Anyone in the channel can see what's posted to the channel. However if they do not have the appropriate permission they won't be able to see further details of the issue if they click through to it.

## What is a notification subscription?

Mattermost users can set up rules that define when a particular event with certain criteria are met in Jira that trigger a notification is sent to a particular channel. These subscription rules can specify the `Jira Project`, `Event Type`, `Issue Type`, and can filter out issues with certain values. 

When a user is setting up a notification subscription they will only see the projects and issue types they have access to within Jira. If they can't see a project in Jira it won't be displayed as an option for that particular user when they are trying to setup a subscription in Mattermost.

An approximate JQL query is output as well. If you're using custom fields or values with spaces in them you'll need to add " 's around the values.

## Who can set up Notification Subscriptions for a channel?

You can define who can set up a notification subscription in the plugin configuration. First, set which **Mattermost** user roles are allowed to access the subscription functionality at all:

![](../.gitbook/assets/image%20%282%29.png)

Then, you can specify which Jira groups they also need to be a member of, in order to access the subscription editor:

![](../.gitbook/assets/image%20%283%29.png)

## How can I see all the notification subscriptions that are set up in Mattermost? 

While logged in as a System Admin type `/jira list` in a Mattermost channel.

## How can I set up Mattermost notifications directly within Jira?

Notifications are configured with webhooks and offer full JQL support.

## Which notification events are supported?

The following Jira event notifications are supported:

* Issue created
* Issue updated, including when an issue is reopened or resolved, or when the assignee is changed
* Issue deleted when not yet resolved
* Comments created, updated, or deleted

If you’d like to see support for additional events, [let us know](https://mattermost.uservoice.com/forums/306457-general).
