// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState} from 'react';

import SVGWrapper from 'components/svgWrapper';

export type RHSIssueTypeIconName = 'bug' | 'story' | 'epic' | 'subtask' | 'incident' | 'task';

export function rhsIssueTypeIconName(issueType: string): RHSIssueTypeIconName {
    const name = issueType.trim().toLowerCase();
    switch (name) {
    case 'bug':
    case 'fault':
        return 'bug';
    case 'story':
        return 'story';
    case 'epic':
        return 'epic';
    case 'sub-task':
    case 'subtask':
        return 'subtask';
    case 'incident':
        return 'incident';
    case 'task':
        return 'task';
    default:
        if (name.indexOf('sub-task') !== -1 || name.indexOf('subtask') !== -1) {
            return 'subtask';
        }
        return 'task';
    }
}

function iconGlyph(name: RHSIssueTypeIconName): React.ReactNode {
    switch (name) {
    case 'bug':
        return (
            <path d='M8 1a3 3 0 0 1 3 3v1h1.5a.5.5 0 0 1 0 1H11v1h1.5a.5.5 0 0 1 0 1H11v1a3 3 0 0 1-6 0V8H3.5a.5.5 0 0 1 0-1H5V6H3.5a.5.5 0 0 1 0-1H5V4a3 3 0 0 1 3-3z'/>
        );
    case 'story':
        return (
            <path d='M4 2h8a1 1 0 0 1 1 1v11l-5-3-5 3V3a1 1 0 0 1 1-1z'/>
        );
    case 'epic':
        return (
            <path d='M9 1L3 9h4l-1 6 7-9H9l1-5z'/>
        );
    case 'incident':
        return (
            <React.Fragment>
                <path d='M8 2l6 11H2L8 2z'/>
                <path d='M8 5.5v4'/>
                <path d='M8 11.5h.01'/>
            </React.Fragment>
        );
    case 'subtask':
        return (
            <g transform='translate(3 3) scale(0.7)'>
                <path
                    d='M3 3h10v10H3V3z'
                    fill='none'
                    stroke='currentColor'
                />
                <path
                    d='M5 8.2l2 2 4-4'
                    fill='none'
                    stroke='currentColor'
                />
            </g>
        );
    case 'task':
        return (
            <React.Fragment>
                <path
                    d='M3 3h10v10H3V3z'
                    fill='none'
                    stroke='currentColor'
                />
                <path
                    d='M5 8.2l2 2 4-4'
                    fill='none'
                    stroke='currentColor'
                />
            </React.Fragment>
        );
    default: {
        const exhaustive: never = name;
        return exhaustive;
    }
    }
}

export type Props = {
    issueType: string;
    iconUrl?: string;
};

export default function RHSIssueTypeIcon(props: Props): JSX.Element {
    const [failed, setFailed] = useState(false);
    const name = rhsIssueTypeIconName(props.issueType);
    const iconUrl = props.iconUrl;

    if (iconUrl && !failed) {
        return (
            <span className='jira-rhs-type-icon-wrap'>
                <img
                    className={'jira-rhs-type-icon jira-rhs-type-icon--' + name}
                    src={iconUrl}
                    alt={props.issueType}
                    width={14}
                    height={14}
                    onError={() => setFailed(true)}
                />
            </span>
        );
    }

    return (
        <span
            className='jira-rhs-type-icon-wrap'
            title={props.issueType}
        >
            <SVGWrapper
                width={14}
                height={14}
                viewBox='0 0 16 16'
                fill='currentColor'
                className={'jira-rhs-type-icon jira-rhs-type-icon--' + name}
            >
                {iconGlyph(name)}
            </SVGWrapper>
        </span>
    );
}
