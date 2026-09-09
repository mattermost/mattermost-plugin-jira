// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import Loading from 'components/loading';

import {RHS_STRINGS} from './rhs_strings';

export function RHSLoadingState(): JSX.Element {
    return (
        <div
            className='jira-rhs-state'
            data-testid='rhs-state-loading'
        >
            <Loading/>
        </div>
    );
}

export function RHSEmptyState(): JSX.Element {
    return (
        <div
            className='jira-rhs-state'
            data-testid='rhs-state-empty'
        >
            <p>{RHS_STRINGS.empty}</p>
        </div>
    );
}

function RHSRetryState(props: {testId: string; message: string; onRetry: () => void}): JSX.Element {
    return (
        <div
            className='jira-rhs-state'
            data-testid={props.testId}
        >
            <p>{props.message}</p>
            <button
                type='button'
                className='btn btn-secondary btn-sm'
                onClick={props.onRetry}
            >
                {RHS_STRINGS.retry}
            </button>
        </div>
    );
}

export function RHSErrorState(props: {onRetry: () => void}): JSX.Element {
    return (
        <RHSRetryState
            testId='rhs-state-error'
            message={RHS_STRINGS.error}
            onRetry={props.onRetry}
        />
    );
}

export function RHSNotConnectedState(props: {onConnect: () => void}): JSX.Element {
    return (
        <div
            className='jira-rhs-state'
            data-testid='rhs-state-not-connected'
        >
            <p>{RHS_STRINGS.notConnected}</p>
            <button
                type='button'
                className='btn btn-primary'
                data-testid='rhs-connect'
                onClick={props.onConnect}
            >
                {RHS_STRINGS.connect}
            </button>
        </div>
    );
}

export function RHSRateLimitedState(props: {onRetry: () => void}): JSX.Element {
    return (
        <RHSRetryState
            testId='rhs-state-rate-limited'
            message={RHS_STRINGS.rateLimited}
            onRetry={props.onRetry}
        />
    );
}
