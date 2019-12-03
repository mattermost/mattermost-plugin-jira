# Feature Summary

## Jira to Mattermost Notifications

### Channel Subscriptions

Notify your team of the latest updates by sending notifications from your Jira projects to Mattermost channels. You can specify which events trigger a notification - and you can filter out certain types of notifications to keep down the noise.

![image](https://user-images.githubusercontent.com/13119842/59113100-6cd7a800-8912-11e9-9e23-3639c0eb9c4d.png)



### Personal Notifications: JiraBot

Each user in Mattermost is connected with their own personal Jira account and notifications for issues where someone is mentioned or assigned an issue is mentioned in your own personal jira notification bot to help everyone stay on top of their assigned issues.

![A personal JiraBot helps keep you on top of your relevant Jira activities](.gitbook/assets/image%20%287%29.png)

## Manage Jira issues in Mattermost

### Create Jira issues

Create Jira issues from scratch or based off of a Mattermost message easily

Without leaving Mattermost's UI, quickly select the project, issue type and enter other fields to create the issue.

![image](https://user-images.githubusercontent.com/13119842/59113188-985a9280-8912-11e9-9def-9a7382b4137e.png)



### Attach Messages to Jira Issues

Keep all information in one place by attaching parts of Mattermost conversations in Jira issues as comments.  Then, on the resulting dialog, select the Jira issue you want to attach it to. You may search for issues containing specific text.

![image](https://user-images.githubusercontent.com/13119842/59113267-b627f780-8912-11e9-90ec-417d430de7e6.png)

### Transition Jira issues

Transition issues without the need to switch to your Jira project. To transition an issue, use the `/jira transition <issue-key> <state>` command.

For instance, `/jira transition EXT-20 done` transitions the issue key **EXT-20** to **Done**.

![image](https://user-images.githubusercontent.com/13119842/59113377-dfe11e80-8912-11e9-8971-f869fa123366.png)

### Assign Jira issues

Assign issues to other Jira users without the need to switch to your Jira project. To assign an issue, use the /jira assign .

For instance, `/jira assign EXT-20 john` transitions the issue key **EXT-20** to **John**.

