// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {RHSIssue} from 'types/model';
import {formatRHSRelativeTime} from 'utils/rhs_time';

import RHSIssueTypeIcon from './rhs_issue_type_icon';

export function rhsStatusClass(categoryKey: string): string {
    switch (categoryKey) {
    case 'new':
    case 'indeterminate':
    case 'done':
    case 'undefined':
        return 'jira-rhs-status jira-rhs-status--' + categoryKey;
    default:
        return 'jira-rhs-status jira-rhs-status--default';
    }
}

export type Props = {
    issue: RHSIssue;
    nowMs: number;
};

export default function RHSIssueRow(props: Props): JSX.Element {
    const issue = props.issue;
    return (
        <article
            className='jira-rhs-issue'
            data-testid={'rhs-issue-' + issue.key}
        >
            <a
                className='jira-rhs-issue-link'
                href={issue.browseUrl}
                target='_blank'
                rel='noopener noreferrer'
            >
                <div className='jira-rhs-issue-line1'>
                    <RHSIssueTypeIcon
                        issueType={issue.issueType}
                        iconUrl={issue.issueTypeIconUrl}
                    />
                    <span className='jira-rhs-issue-key'>{issue.key}</span>
                    <span
                        className='jira-rhs-issue-summary'
                        title={issue.summary}
                    >
                        {issue.summary}
                    </span>
                </div>
                <div className='jira-rhs-issue-line2'>
                    <span className={rhsStatusClass(issue.status.categoryKey)}>
                        {issue.status.name}
                    </span>
                    <span className='jira-rhs-issue-dot'>{'·'}</span>
                    <span className='jira-rhs-issue-project'>{issue.project}</span>
                    <span className='jira-rhs-issue-dot'>{'·'}</span>
                    <span className='jira-rhs-issue-time'>{formatRHSRelativeTime(issue.updated, props.nowMs)}</span>
                </div>
            </a>
        </article>
    );
}
