// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import JiraIcon from 'components/icon';

interface Props {
    actionText: string;
}

export default function AttachCommentToIssuePostMenuAction({actionText}: Props): JSX.Element {
    return (
        <li
            className='MenuItem'
            role='menuitem'
        >
            <button className='style--none'>
                <span className='MenuItem__icon'>
                    <JiraIcon type='menu'/>
                </span>
                {actionText}
            </button>
        </li>
    );
}