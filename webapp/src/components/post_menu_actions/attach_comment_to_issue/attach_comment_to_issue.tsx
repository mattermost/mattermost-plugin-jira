// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import JiraIcon from 'components/icon';

interface Props {
    actionText: string;
}

export default function AttachCommentToIssuePostMenuAction({actionText}: Props): JSX.Element {
    return (
        <>
            <JiraIcon type='menu'/>
            {actionText}
        </>
    );
}