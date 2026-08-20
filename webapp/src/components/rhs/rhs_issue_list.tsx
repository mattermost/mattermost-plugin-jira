// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {RHSErrorCode, RHSIssue} from 'types/model';

import RHSIssueRow from './rhs_issue_row';
import {RHS_STRINGS} from './rhs_strings';

export type Props = {
    issues: RHSIssue[];
    loading: boolean;
    isLast: boolean;
    error: RHSErrorCode | null;
    nowMs: number;
    onLoadMore: () => void;
    onRetry: () => void;
};

export default function RHSIssueList(props: Props): JSX.Element {
    return (
        <div
            className='jira-rhs-list'
            data-testid='rhs-issue-list'
        >
            {props.issues.map((issue) => (
                <RHSIssueRow
                    key={issue.key}
                    issue={issue}
                    nowMs={props.nowMs}
                />
            ))}
            {props.loading && (
                <div
                    className='jira-rhs-list-loading'
                    data-testid='rhs-list-loading'
                >
                    <span className='fa fa-spin fa-spinner'/>
                    <span>{RHS_STRINGS.loadingMore}</span>
                </div>
            )}
            {props.error !== null && !props.loading && (
                <div
                    className='jira-rhs-list-error'
                    data-testid='rhs-list-error'
                >
                    <span>{props.error === 'rate_limited' ? RHS_STRINGS.rateLimited : RHS_STRINGS.error}</span>
                    <button
                        type='button'
                        onClick={props.onRetry}
                    >
                        {RHS_STRINGS.retry}
                    </button>
                </div>
            )}
            {!props.isLast && !props.loading && (
                <button
                    type='button'
                    className='jira-rhs-load-more'
                    data-testid='rhs-load-more'
                    onClick={props.onLoadMore}
                >
                    {RHS_STRINGS.loadMore}
                </button>
            )}
        </div>
    );
}
