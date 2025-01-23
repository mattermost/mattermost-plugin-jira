// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {JiraUser} from './model';

export type TicketDetails = {
    assigneeName: string;
    assigneeAvatar: string;
    labels: string[];
    description: string;
    summary: string;
    ticketId: string;
    jiraIcon: string;
    versions: string;
    statusKey: string;
    issueIcon: string;
}

export type TicketData = {
    key: string;
    fields: TicketDataFields;
}

export type AvatarUrls = {
    '48x48': string;
}

export type TicketDataFields = {
    assignee: JiraUser | null;
    labels: string[];
    description: string;
    summary: string;
    project: {avatarUrls: AvatarUrls};
    versions: string[];
    status: {name: string};
    issuetype: {iconUrl: string};
}

export type IssueAction = {
    type: string;
    data: TicketData;
}
