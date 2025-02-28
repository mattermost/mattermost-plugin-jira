// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {IssueAction} from 'types/tooltip';

export const ticketData = (assigneeName: string | null): IssueAction => ({
    data: {
        key: 'ABC-123',
        fields: {
            assignee: {
                displayName: assigneeName || '',
                avatarUrls: assigneeName ? {
                    '48x48': 'https://something.atlassian.net/avatar.png',
                    '16x16': 'https://something.atlassian.net/avatar.png',
                    '24x24': 'https://something.atlassian.net/avatar.png',
                    '36x36': 'https://something.atlassian.net/avatar.png',
                } : '',
            },
            labels: ['label1', 'label2'],
            description: 'This is a test description',
            summary: 'This is a test summary',
            project: {
                avatarUrls: {
                    '48x48': 'https://something.atlassian.net/project.png',
                },
            },
            versions: ['Version 1.0', 'Version 2.0'],
            status: {
                name: 'In Progress',
            },
            issuetype: {
                iconUrl: 'https://something.atlassian.net/issuetype.png',
            },
        },
    },
    type: 'mockType',
});
